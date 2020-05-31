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
var ErrUserIDNotFound = errors.New("user_id not found")

type Handler func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

type UsersRepository interface {
	Find(userID string) (models.User, error)
}

type LocationsRepository interface {
	GetScoreByUserID(userID string) (int, error)
}

type Response struct {
	Username string `json:"username"`
	Score    int    `json:"score"`
}

func Adapter(
	usersRepo UsersRepository,
	locationsRepo LocationsRepository,
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

		score, err := locationsRepo.GetScoreByUserID(userID)
		if err != nil {
			if err == repositories.ErrNoLocationsFound {
				jsonResponse, _ := json.Marshal(Response{
					Username: user.Username,
					Score:    score,
				})
				return internal.Respond(http.StatusOK, string(jsonResponse)), nil
			}
			return internal.Error(http.StatusInternalServerError, err), nil
		}

		response := Response{
			Username: user.Username,
			Score:    score,
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
}
