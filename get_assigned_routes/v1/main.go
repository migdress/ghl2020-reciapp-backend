package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"

	"github.com/Globhack/ghl2020-reciapp-backend/internal"
	"github.com/Globhack/ghl2020-reciapp-backend/internal/models"
	"github.com/Globhack/ghl2020-reciapp-backend/internal/repositories"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var ErrUserIDEmpty = errors.New("user_id cannot be empty")

type Handler func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

type RoutesRepoRepository interface {
	GetAssignedRoutesbyUserID(userID string) ([]models.Route, error)
}

type ResponseRoutePickingPoint struct {
	ID         string  `json:"id"`
	LocationID string  `json:"locationid"`
	Country    string  `json:"country"`
	City       string  `json:"city"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Address1   string  `json:"address1"`
	Address2   string  `json:"address2"`
}

type ResponseAssignedRoutebyUserID struct {
	ID            string   `json:"id"`
	Materials     []string `json:"materials"`
	Sector        string   `json:"sector"`
	Shift         string   `json:"shift"`
	Date          string   `json:"date"`
	PickingPoints []ResponseRoutePickingPoint
}

type Response struct {
	AssignedRoutes []ResponseAssignedRoutebyUserID `json:"assigned_routes"`
}

func Adapter(routesRepo RoutesRepoRepository) Handler {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

		usertID := req.PathParameters["user_id"]

		if usertID == "" {
			return internal.Error(http.StatusBadRequest, ErrUserIDEmpty), nil
		}
		routes, err := routesRepo.GetAssignedRoutesbyUserID(usertID)
		if err != nil {
			if err == repositories.ErrRouteNotFound {
				return internal.Error(http.StatusNotFound, err), nil
			}
			return internal.Error(http.StatusInternalServerError, err), nil
		}

		assignedresponseRoutes := make([]ResponseAssignedRoutebyUserID, len(routes))
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
				}
			}
			assignedresponseRoutes[i] = ResponseAssignedRoutebyUserID{
				ID:        route.ID,
				Materials: route.Materials,
				Sector:    route.Sector,
				Shift:     route.Shift,
				//	Date:          startsAt,
				PickingPoints: responseRoutesPickingPoints,
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

	hoursOffsetString := os.Getenv("HOURS_OFFSET")
	if hoursOffsetString == "" {
		panic("HOURS_OFFSET cannot be empty")
	}

	timezone := os.Getenv("TIMEZONE")
	if timezone == "" {
		panic("TIMEZONE cannot be empty")
	}

	timeHelper, err := internal.NewTimeHelper(timezone)
	if err != nil {
		panic(err)
	}

	session := session.New()
	dynamodbClient := dynamodb.New(session)

	routesRepo := repositories.NewDynamoDBRoutesRepository(
		dynamodbClient,
		routesTable,
		timeHelper,
	)
	handler := Adapter(routesRepo)
	lambda.Start(handler)
}
