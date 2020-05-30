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
)

var ErrUsernameEmpty = errors.New("username cannot be empty")

type Handler func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

type UsersRepository interface {
	FindByUsername(username string) (models.User, error)
}

type LocationsRepository interface {
	FindByUserID(id string) ([]models.Location, error)
}

type Request struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type Response struct {
	Username  string            `json:"username"`
	FirstName string            `json:"firstname"`
	LastName  string            `json:"lastname"`
	Type      string            `json:"type"`
	Locations []models.Location `json:"location"`
}

func Adapter(usersRepo UsersRepository, locationsRepo LocationsRepository) Handler {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

		reqBody := Request{}
		err := json.Unmarshal([]byte(req.Body), &reqBody)
		if err != nil {
			return internal.Error(http.StatusBadRequest, err), nil
		}

		if reqBody.Username == "" {
			return internal.Error(http.StatusBadRequest, ErrUsernameEmpty), nil
		}

		user, err := usersRepo.FindByUsername(reqBody.Username)
		if err != nil {
			if err == repositories.ErrUserNotFound {
				return internal.Error(http.StatusNotFound, err), nil
			}

			return internal.Error(http.StatusInternalServerError, err), nil
		}

		locations, err := locationsRepo.FindByUserID(user.ID)
		if err != nil && err != repositories.ErrNoLocationsFound {
			return internal.Error(http.StatusInternalServerError, err), nil
		}

		response := Response{
			Username:  user.Username,
			FirstName: user.Firstname,
			LastName:  user.Lastname,
			Type:      user.Type,
			Locations: locations,
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
	usersRepo := repositories.NewDynamoDBUsersRepository(usersTable)

	locationsTable := os.Getenv("DYNAMODB_LOCATIONS")
	if locationsTable == "" {
		panic("DYNAMODB_LOCATIONS cannot be empty")
	}
	locationsRepo := repositories.NewDynamoDBLocationsRepository(locationsTable)

	handler := Adapter(usersRepo, locationsRepo)
	lambda.Start(handler)
}
