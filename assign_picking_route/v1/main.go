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

var ErrRouteIDEmpty = errors.New("route_id cannot be empty")
var ErrUserIDEmpty = errors.New("user_id cannot be empty")

type Handler func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

type UsersRepository interface {
	Find(userID string) (models.User, error)
}

type RoutesRepository interface {
	Assign(userID string, routeID string) error
	Find(routeID string) (models.Route, error)
}

type Request struct {
	UserID  string `json:"user_id"`
	RouteID string `json:"route_id"`
}

func Adapter(usersRepo UsersRepository, routesRepo RoutesRepository) Handler {
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

		_, err = usersRepo.Find(reqBody.UserID)
		if err != nil {
			if err == repositories.ErrUserNotFound {
				return internal.Error(http.StatusNotFound, err), nil
			}

			return internal.Error(http.StatusInternalServerError, err), nil
		}

		_, err = routesRepo.Find(reqBody.RouteID)
		if err != nil {
			if err == repositories.ErrRouteNotFound {
				return internal.Error(http.StatusNotFound, err), nil
			}

			return internal.Error(http.StatusInternalServerError, err), nil
		}

		err = routesRepo.Assign(reqBody.UserID, reqBody.RouteID)
		if err != nil {
			if err == repositories.ErrRouteAlreadyAssigned {
				return internal.Error(http.StatusConflict, err), nil
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
