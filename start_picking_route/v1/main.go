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
var ErrRouteIDEmpty = errors.New("route_id  cannot be empty")
var ErrWrongGathererID = errors.New("this route is assigned to another gatherer")
var ErrWrongUserType = errors.New("user must be of type gatherer")

type Handler func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

type UsersRepository interface {
	Find(userID string) (models.User, error)
}

type RouteRepository interface {
	Find(routeID string) (models.Route, error)
	Initiate(routeID string) error
}

type TimeHelper interface {
	ToLatamFormat(d time.Time) (string, error)
	ToISO8601(d time.Time) (string, error)
}

type Request struct {
	UserID  string `json:"user_id"`
	RouteID string `json:"route_id"`
}

type ResponsePickingPoint struct {
	Country    string   `json:"country"`
	City       string   `json:"city"`
	Address1   string   `json:"address_1"`
	Address2   string   `json:"address_2"`
	LocationID string   `json:"location_id"`
	Name       string   `json:"name"`
	Latitude   float64  `json:"latitude"`
	Longitude  float64  `json:"longitude"`
	Materials  []string `json:"materials"`
}

type ResponseRoute struct {
	ID            string                 `json:"id"`
	Materials     []string               `json:"materials"`
	Sector        string                 `json:"sector"`
	Status        string                 `json:"status"`
	Shift         string                 `json:"shift"`
	Date          string                 `json:"date"`
	PickingPoints []ResponsePickingPoint `json:"picking_points"`
}

type Response struct {
	AssignedRoute ResponseRoute `json:"assigned_route"`
}

func Adapter(
	usersRepo UsersRepository,
	routeRepo RouteRepository,
	timeHelper TimeHelper,
) Handler {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

		reqBody := Request{}
		err := json.Unmarshal([]byte(req.Body), &reqBody)
		if err != nil {
			return internal.Error(http.StatusBadRequest, err), nil
		}

		if reqBody.UserID == "" {
			return internal.Error(http.StatusBadRequest, ErrUserIDEmpty), nil
		}
		if reqBody.RouteID == "" {
			return internal.Error(http.StatusBadRequest, ErrRouteIDEmpty), nil
		}

		user, err := usersRepo.Find(reqBody.UserID)
		if err != nil {
			if err == repositories.ErrUserNotFound {
				return internal.Error(http.StatusNotFound, err), nil
			}
			return internal.Error(http.StatusInternalServerError, err), nil
		}

		route, err := routeRepo.Find(reqBody.RouteID)
		if err != nil {
			if err == repositories.ErrRouteNotFound {
				return internal.Error(http.StatusNotFound, err), nil
			}
			return internal.Error(http.StatusInternalServerError, err), nil
		}

		if user.Type != models.UserTypeGatherer {
			return internal.Error(http.StatusForbidden, ErrWrongUserType), nil
		}
		if route.GathererID != user.ID {
			return internal.Error(http.StatusForbidden, ErrWrongGathererID), nil
		}

		log.Printf("route.InitiatedAt: (%v)\n", route.InitiatedAt)
		if route.InitiatedAt == nil {
			log.Printf("Initiating route\n")
			err := routeRepo.Initiate(route.ID)
			if err != nil {
				return internal.Error(http.StatusInternalServerError, err), nil
			}
		}

		responseRoutePickingPoints := make([]ResponsePickingPoint, len(route.PickingPoints))
		for i, pp := range route.PickingPoints {
			responseRoutePickingPoints[i] = ResponsePickingPoint{
				Country:    pp.Country,
				City:       pp.City,
				Address1:   pp.Address1,
				Address2:   pp.Address2,
				LocationID: pp.LocationID,
				Latitude:   pp.Latitude,
				Longitude:  pp.Longitude,
				Materials:  pp.Materials,
			}
		}

		startsAt, err := timeHelper.ToISO8601(*route.StartsAt)
		if err != nil {
			return internal.Error(http.StatusInternalServerError, err), nil
		}
		responseAssignedRoute := ResponseRoute{
			ID:            route.ID,
			Materials:     route.Materials,
			Sector:        route.Sector,
			Status:        route.Status,
			Shift:         route.Shift,
			Date:          startsAt,
			PickingPoints: responseRoutePickingPoints,
		}

		response := Response{
			AssignedRoute: responseAssignedRoute,
		}
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			return internal.Error(http.StatusInternalServerError, err), nil
		}

		return internal.Respond(http.StatusOK, string(jsonResponse)), nil
	}
}

func main() {
	usersTable := os.Getenv("DYNAMODB_USERS")
	if usersTable == "" {
		panic("DYNAMODB_USERS cannot be empty")
	}

	routesTable := os.Getenv("DYNAMODB_PICKING_ROUTES")
	if routesTable == "" {
		panic("DYNAMODB_PICKING_ROUTES cannot be empty")
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

	handler := Adapter(usersRepo, routesRepo, timeHelper)
	lambda.Start(handler)
}
