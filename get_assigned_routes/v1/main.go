package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Globhack/ghl2020-reciapp-backend/internal"
	"github.com/Globhack/ghl2020-reciapp-backend/internal/models"
	"github.com/Globhack/ghl2020-reciapp-backend/internal/repositories"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var ErrUserIDEmpty = errors.New("user_id cannot be empty")
var ErrWrongUserType = errors.New("user must be of type gatherer")

type Handler func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

type RoutesRepoRepository interface {
	GetAssignedRoutesbyUserID(userID string) ([]models.Route, error)
}

type UsersRepository interface {
	Find(userID string) (models.User, error)
}

type TimeHelper interface {
	ToLatamFormat(d time.Time) (string, error)
	ToISO8601(d time.Time) (string, error)
}

type ResponseRoutePickingPoint struct {
	ID         string   `json:"id"`
	LocationID string   `json:"locationid"`
	Country    string   `json:"country"`
	City       string   `json:"city"`
	Latitude   float64  `json:"latitude"`
	Longitude  float64  `json:"longitude"`
	Address1   string   `json:"address1"`
	Address2   string   `json:"address2"`
	Materials  []string `json:"materials"`
}

type ResponseRoute struct {
	ID            string                      `json:"id"`
	Materials     []string                    `json:"materials"`
	Sector        string                      `json:"sector"`
	Shift         string                      `json:"shift"`
	Date          string                      `json:"date"`
	Status        string                      `json:"status"`
	PickingPoints []ResponseRoutePickingPoint `json:"picking_points"`
}

type Response struct {
	AssignedRoutes []ResponseRoute `json:"assigned_routes"`
}

func Adapter(
	routesRepo RoutesRepoRepository,
	usersRepo UsersRepository,
	timeHelper TimeHelper,
) Handler {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		userID := req.PathParameters["user_id"]

		if userID == "" {
			return internal.Error(http.StatusBadRequest, ErrUserIDEmpty), nil
		}

		user, err := usersRepo.Find(userID)
		if err != nil {
			if err == repositories.ErrUserNotFound {
				return internal.Error(http.StatusNotFound, err), nil
			}

			return internal.Error(http.StatusInternalServerError, err), nil
		}

		if user.Type != models.UserTypeGatherer {
			return internal.Error(http.StatusForbidden, ErrWrongUserType), nil
		}

		log.Printf("looking for routes assigned to gatherer_id(%v)\n", user.ID)
		routes, err := routesRepo.GetAssignedRoutesbyUserID(userID)
		if err != nil {
			if err == repositories.ErrNoAssignedRoutes {
				log.Printf("no assigned route found")
				responseBytes, _ := json.Marshal(Response{
					AssignedRoutes: []ResponseRoute{},
				})
				return internal.Respond(http.StatusOK, string(responseBytes)), nil
			}
			return internal.Error(http.StatusInternalServerError, err), nil
		}
		log.Printf("found (%v) routes assigned\n", len(routes))

		assignedresponseRoutes := make([]ResponseRoute, len(routes))
		for i, route := range routes {
			responseRoutesPickingPoints := make([]ResponseRoutePickingPoint, len(route.PickingPoints))
			for j, pp := range route.PickingPoints {
				responseRoutesPickingPoints[j] = ResponseRoutePickingPoint{
					ID:         pp.ID,
					LocationID: pp.LocationID,
					Country:    pp.Country,
					City:       pp.City,
					Latitude:   pp.Latitude,
					Longitude:  pp.Longitude,
					Address1:   pp.Address1,
					Address2:   pp.Address2,
					Materials:  pp.Materials,
				}
			}

			startsAt, err := timeHelper.ToISO8601(*route.StartsAt)
			if err != nil {
				return internal.Error(http.StatusInternalServerError, err), nil
			}
			assignedresponseRoutes[i] = ResponseRoute{
				ID:            route.ID,
				Materials:     route.Materials,
				Sector:        route.Sector,
				Shift:         route.Shift,
				Date:          startsAt,
				PickingPoints: responseRoutesPickingPoints,
				Status:        route.Status,
			}
		}
		response := Response{
			AssignedRoutes: assignedresponseRoutes,
		}
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			return internal.Error(http.StatusInternalServerError, err), nil
		}

		return internal.Respond(http.StatusOK, string(jsonResponse)), nil
	}
}

func main() {
	routesTable := os.Getenv("DYNAMODB_PICKING_ROUTES")
	if routesTable == "" {
		panic("DYNAMODB_PICKING_ROUTES cannot be empty")
	}

	usersTable := os.Getenv("DYNAMODB_USERS")
	if usersTable == "" {
		panic("DYNAMODB_USERS cannot be empty")
	}

	timezone := os.Getenv("TIMEZONE")
	if timezone == "" {
		panic("TIMEZONE cannot be empty")
	}

	timeHelper, err := internal.NewTimeHelper(timezone)
	if err != nil {
		panic(err)
	}

	uuidHelper := internal.NewUUIDHelper()

	session := session.New()
	dynamodbClient := dynamodb.New(session)
	usersRepo := repositories.NewDynamoDBUsersRepository(
		dynamodbClient,
		usersTable,
	)

	routesRepo := repositories.NewDynamoDBRoutesRepository(
		dynamodbClient,
		routesTable,
		timeHelper,
		uuidHelper,
	)

	handler := Adapter(routesRepo, usersRepo, timeHelper)
	lambda.Start(handler)
}
