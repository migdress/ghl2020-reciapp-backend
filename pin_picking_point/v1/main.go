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
var ErrShiftIdEmpty = errors.New("shift_id cannot be empty")
var ErrLocationIDEmpty = errors.New("location_id cannot be empty")
var ErrMaterialsEmpty = errors.New("materials cannot be empty")

type Handler func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

type RoutesRepository interface {
	Pin(userID string, location models.Location, shiftID string, Materials []string) error
	Find(routeID string) (models.Route, error)
}

type UsersRepository interface {
	Find(userID string) (models.User, error)
}

type LocationssRepository interface {
	Find(locationID string) (models.Location, error)
}

type Request struct {
	UserID     string   `json:"username"`
	ShiftID    string   `json:"shift_id"`
	LocationID string   `json:"location_id"`
	Materials  []string `json:"materials"`
}

func Adapter(routesRepo RoutesRepository, userRepo UsersRepository, locationRepo LocationssRepository) Handler {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

		reqBody := Request{}
		err := json.Unmarshal([]byte(req.Body), &reqBody)
		if err != nil {
			return internal.Error(http.StatusBadRequest, err), nil
		}

		if reqBody.UserID == "" {
			return internal.Error(http.StatusBadRequest, ErrUsernameEmpty), nil
		}
		if reqBody.ShiftID == "" {
			return internal.Error(http.StatusBadRequest, ErrShiftIdEmpty), nil
		}
		if reqBody.LocationID == "" {
			return internal.Error(http.StatusBadRequest, ErrLocationIDEmpty), nil
		}
		if len(reqBody.Materials) == 0 {
			return internal.Error(http.StatusBadRequest, ErrMaterialsEmpty), nil
		}

		_, err = userRepo.Find(reqBody.UserID)
		if err != nil {
			if err == repositories.ErrUserNotFound {
				return internal.Error(http.StatusNotFound, err), nil
			}
		}
		_, err = routesRepo.Find(reqBody.ShiftID)
		if err != nil {
			if err == repositories.ErrRouteNotFound {
				return internal.Error(http.StatusNotFound, err), nil
			}
		}
		location, err := locationRepo.Find(reqBody.LocationID)
		if err != nil {
			if err == repositories.ErrLocationNotFound {
				return internal.Error(http.StatusNotFound, err), nil
			}
		}
		err = routesRepo.Pin(reqBody.UserID, location, reqBody.ShiftID, reqBody.Materials)
		if err != nil {
			if err == repositories.ErrPickingPointAlreadyPinned {
				return internal.Error(http.StatusNotFound, err), nil
			}
			return internal.Error(http.StatusInternalServerError, err), nil
		}

		return internal.Respond(http.StatusOK, ""), nil
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
