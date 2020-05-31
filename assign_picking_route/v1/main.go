package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
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

var ErrRouteIDEmpty = errors.New("route_id cannot be empty")
var ErrUserIDEmpty = errors.New("user_id cannot be empty")
var ErrWrongUserType = errors.New("user must of type gatherer")

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

		user, err := usersRepo.Find(reqBody.UserID)
		if err != nil {
			if err == repositories.ErrUserNotFound {
				return internal.Error(http.StatusNotFound, err), nil
			}

			return internal.Error(http.StatusInternalServerError, err), nil
		}

		if user.Type != models.UserTypeGatherer {
			return internal.Error(http.StatusForbidden, ErrWrongUserType), nil
		}

		route, err := routesRepo.Find(reqBody.RouteID)
		if err != nil {
			if err == repositories.ErrRouteNotFound {
				return internal.Error(http.StatusNotFound, err), nil
			}

			return internal.Error(http.StatusInternalServerError, err), nil
		}

		log.Printf("route.GathererID: (%v), user.ID: (%v)\n", route.GathererID, user.ID)
		if route.GathererID == user.ID {
			return internal.Respond(http.StatusOK, ""), nil
		}

		err = routesRepo.Assign(reqBody.UserID, reqBody.RouteID)
		if err != nil {
			if err == repositories.ErrRouteAlreadyAssigned {
				return internal.Error(http.StatusUnprocessableEntity, err), nil
			}
			return internal.Error(http.StatusInternalServerError, err), nil
		}

		return internal.Respond(http.StatusOK, ""), nil
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

	locationsTable := os.Getenv("DYNAMODB_LOCATIONS")
	if locationsTable == "" {
		panic("DYNAMODB_LOCATIONS cannot be empty")
	}

	timezone := os.Getenv("TIMEZONE")
	if timezone == "" {
		panic("TIMEZONE cannot be empty")
	}

	session := session.New()
	dynamodbClient := dynamodb.New(session)

	timeHelper, err := internal.NewTimeHelper(timezone)
	if err != nil {
		panic(err)
	}

	uuidHelper := internal.NewUUIDHelper()

	usersRepo := repositories.NewDynamoDBUsersRepository(
		dynamodbClient,
		usersTable,
	)
	routesRepo := repositories.NewDynamoDBRoutesRepository(
		dynamodbClient,
		routesTable,
		locationsTable,
		timeHelper,
		uuidHelper,
	)

	handler := Adapter(usersRepo, routesRepo)
	lambda.Start(handler)

}
