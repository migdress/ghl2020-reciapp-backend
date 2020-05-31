package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Globhack/ghl2020-reciapp-backend/internal"
	"github.com/Globhack/ghl2020-reciapp-backend/internal/models"
	"github.com/Globhack/ghl2020-reciapp-backend/internal/repositories"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var ErrUsernameEmpty = errors.New("username cannot be empty")

type Handler func(ctx context.Context) (events.APIGatewayProxyResponse, error)

type RoutesRepoRepository interface {
	FindAvailableRoutes(currentTime time.Time, maxTime time.Time) ([]models.Route, error)
}

type TimeHelper interface {
	NowWithTimezone() (time.Time, error)
	ToISO8601(d time.Time) (string, error)
}

type ResponseRoutePickingPoint struct {
	ID         string  `json:"id"`
	Name       string  `json:"name"`
	LocationID string  `json:"locationid"`
	Country    string  `json:"country"`
	City       string  `json:"city"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Address1   string  `json:"address1"`
	Address2   string  `json:"address2"`
}

type ResponseRoute struct {
	ID            string   `json:"id"`
	Materials     []string `json:"materials"`
	Sector        string   `json:"sector"`
	Shift         string   `json:"shift"`
	Date          string   `json:"date"`
	PickingPoints []ResponseRoutePickingPoint
}

type Response struct {
	Routes []ResponseRoute `json:"routes"`
}

func Adapter(
	routesRepo RoutesRepoRepository,
	hoursOffset int,
	timeHelper TimeHelper,
) Handler {
	return func(ctx context.Context) (events.APIGatewayProxyResponse, error) {

		// Calculate window time to query for routes
		now, err := timeHelper.NowWithTimezone()
		if err != nil {
			return internal.Error(http.StatusInternalServerError, err), nil
		}
		maxTime := now.Add(time.Hour * time.Duration(hoursOffset))

		// Query for routes
		routes, err := routesRepo.FindAvailableRoutes(now, maxTime)
		if err != nil {
			if err == repositories.ErrUserNotFound {
				return internal.Error(http.StatusNotFound, err), nil
			}

			return internal.Error(http.StatusInternalServerError, err), nil
		}

		// Prepare response
		responseRoutes := make([]ResponseRoute, len(routes))
		for i, route := range routes {
			responseRoutesPickingPoints := make([]ResponseRoutePickingPoint, len(route.PickingPoints))
			for j, pp := range route.PickingPoints {
				responseRoutesPickingPoints[j] = ResponseRoutePickingPoint{
					ID:         pp.ID,
					Name:       pp.Name,
					LocationID: pp.LocationID,
					Country:    pp.Country,
					City:       pp.City,
					Latitude:   pp.Latitude,
					Longitude:  pp.Longitude,
					Address1:   pp.Address1,
					Address2:   pp.Address2,
				}
			}

			startsAt, err := timeHelper.ToISO8601(*route.StartsAt)
			if err != nil {
				return internal.Error(http.StatusInternalServerError, err), nil
			}
			responseRoutes[i] = ResponseRoute{
				ID:            route.ID,
				Materials:     route.Materials,
				Sector:        route.Sector,
				Shift:         route.Shift,
				Date:          startsAt,
				PickingPoints: responseRoutesPickingPoints,
			}
		}
		response := Response{
			Routes: responseRoutes,
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

	hoursOffset, err := strconv.Atoi(hoursOffsetString)
	if err != nil {
		panic("HOURS_OFFSET must be an integer")
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
	handler := Adapter(routesRepo, hoursOffset, timeHelper)
	lambda.Start(handler)
}
