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

type ResponseLocation struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Country   string  `json:"country"`
	City      string  `json:"city"`
	Address1  string  `json:"address_1"`
	Address2  string  `json:"address_2"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
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
		responseLocations := make([]ResponseLocation, len(locations))
		for i, location := range locations {
			responseLocations[i] = ResponseLocation{
				ID:        location.ID,
				Name:      location.Name,
				Country:   location.Country,
				City:      location.City,
				Address1:  location.Address1,
				Address2:  location.Address2,
				Latitude:  location.Latitude,
				Longitude: location.Longitude,
			}
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
