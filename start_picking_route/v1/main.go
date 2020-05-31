package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Globhack/ghl2020-reciapp-backend/internal"
	"github.com/Globhack/ghl2020-reciapp-backend/internal/models"
	"github.com/Globhack/ghl2020-reciapp-backend/internal/repositories"
	"github.com/aws/aws-lambda-go/events"
)

var ErrUsernameEmpty = errors.New("username cannot be empty")
var ErrRouteIDEmpty = errors.New("route_id  cannot be empty")

type Handler func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

type UsersRepository interface {
	FindByUsername(userID string) (models.User, error)
}

type RouteRepository interface {
	FindByRoute(routeID string) (models.Route, error)
}

type Request struct {
	UserID  string `json:"user_id"`
	RouteID string `json:"route_id"`
}

type ResponsePickingPoint struct {
	Country    string  `json:"country"`
	City       string  `json:"city"`
	Address1   string  `json:"address_1"`
	Address2   string  `json:"address_2"`
	LocationID string  `json:"location_id"`
	Name       string  `json:"name"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
}

type ResponseAssignedRoute struct {
	ID            string                 `json:"id"`
	Materials     []string               `json:"materials"`
	Sector        string                 `json:"sector"`
	Status        string                 `json:"status"`
	Shift         string                 `json:"shift"`
	Date          string                 `json:"date"`
	PickingPoints []ResponsePickingPoint `json:"picking_points"`
}

func Adapter(usersRepo UsersRepository, routeRepo RouteRepository) Handler {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

		reqBody := Request{}
		err := json.Unmarshal([]byte(req.Body), &reqBody)
		if err != nil {
			return internal.Error(http.StatusBadRequest, err), nil
		}

		if reqBody.UserID == "" {
			return internal.Error(http.StatusBadRequest, ErrUsernameEmpty), nil
		}
		if reqBody.RouteID == "" {
			return internal.Error(http.StatusBadRequest, ErrRouteIDEmpty), nil
		}

		_, err = usersRepo.FindByUsername(reqBody.UserID)
		if err != nil {
			if err == repositories.ErrUserNotFound {
				return internal.Error(http.StatusNotFound, err), nil
			}
			return internal.Error(http.StatusInternalServerError, err), nil
		}
		route, err := routeRepo.FindByRoute(reqBody.RouteID)
		if err != nil {
			if err == repositories.ErrRouteNotFound {
				return internal.Error(http.StatusNotFound, err), nil
			}
			return internal.Error(http.StatusInternalServerError, err), nil
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
			}
		}
		responseAssignedRoute := ResponseAssignedRoute{
			ID:        route.ID,
			Materials: route.Materials,
			Sector:    route.Sector,
			Status:    route.Status,
			Shift:     route.Shift,
			//Date:          route.Date,
			PickingPoints: responseRoutePickingPoints,
		}

		jsonResponse, err := json.Marshal(responseAssignedRoute)
		if err != nil {
			return internal.Error(http.StatusInternalServerError, err), nil
		}

		return internal.Respond(http.StatusOK, string(jsonResponse)), nil
	}
}

func main() {
	/*
		usersTable := os.Getenv("DYNAMODB_USERS")
		if usersTable == "" {
			panic("DYNAMODB_USERS cannot be empty")
		}
		locationsTable := os.Getenv("DYNAMODB_LOCATIONS")
		if locationsTable == "" {
			panic("DYNAMODB_LOCATIONS cannot be empty")
		}
		userLocationsTable := os.Getenv("DYNAMODB_USER_LOCATIONS")
		if userLocationsTable == "" {
			panic("DYNAMODB_USER_LOCATIONS cannot be empty")
		}

		session := session.New()
		dynamodbClient := dynamodb.New(session)
		usersRepo := repositories.NewDynamoDBUsersRepository(
			dynamodbClient,
			usersTable,
		)
		locationsRepo := repositories.NewDynamoDBLocationsRepository(
			dynamodbClient,
			userLocationsTable,
			locationsTable,
		)

		handler := Adapter(usersRepo, locationsRepo)
		lambda.Start(handler)
	*/
}
