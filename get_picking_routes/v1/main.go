package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Globhack/ghl2020-reciapp-backend/internal"
	"github.com/Globhack/ghl2020-reciapp-backend/internal/models"
	"github.com/Globhack/ghl2020-reciapp-backend/internal/repositories"
	"github.com/aws/aws-lambda-go/events"
)

var ErrUsernameEmpty = errors.New("username cannot be empty")

type Handler func(ctx context.Context) (events.APIGatewayProxyResponse, error)

type RoutesRepoRepository interface {
	FindAvailableRoutes(currentTime time.Time) ([]models.Routes, error)
}

type ResponseRoutePickingPoint struct {
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
	Routes []ResponseRoute
}

func Adapter(routesRepo RoutesRepoRepository) Handler {
	return func(ctx context.Context) (events.APIGatewayProxyResponse, error) {

		routes, err := routesRepo.FindAvailableRoutes(time.Now().UTC())
		if err != nil {
			if err == repositories.ErrUserNotFound {
				return internal.Error(http.StatusNotFound, err), nil
			}

			return internal.Error(http.StatusInternalServerError, err), nil
		}

		responseRoutes := make([]ResponseRoute, len(routes))
		for i, route := range routes {
			responseRoutesPickingPoints := make([]ResponseRoutePickingPoint, len(route.PickingPoints))
			for j, pp := range route.PickingPoints {
				responseRoutesPickingPoints[j] = ResponseRoutePickingPoint{
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
			responseRoutes[i] = ResponseRoute{
				ID:            route.ID,
				Materials:     route.Materials,
				Sector:        route.Sector,
				Shift:         route.Shift,
				Date:          route.Date,
				PickingPoints: responseRoutesPickingPoints,
			}
		}
		response := Response{
			responseRoutes,
		}
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			return internal.Error(http.StatusInternalServerError, err), nil
		}

		return internal.Respond(http.StatusOK, string(jsonResponse)), nil
	}
}

func main() {
	// pickingRouteTable := os.Getenv("DYNAMODB_PICKING_ROUTES")
	// if pickingRouteTable == "" {
	// 	panic("DYNAMODB_PICKING_ROUTES cannot be empty")
	// }
	// usersRepo := repositories.NewDynamoDBUsersRepository(usersTable)

	// handler := Adapter(usersRepo, locationsRepo)
	// lambda.Start(handler)
}
