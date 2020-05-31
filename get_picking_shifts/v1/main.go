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

var ErrUserIDEmpty = errors.New("user_id cannot be empty")
var ErrUserIDNotFound = errors.New("user_id not found")

type Handler func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

type UserScoreRepository interface {
	GetUserScorebyUserID(userID string) (models.User, error)
}

type ScoreResponse struct {
	Username string `json:"username"`
	Score    int    `json:"score"`
}

func Adapter(userRepo UserScoreRepository) Handler {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

		usertID := req.PathParameters["user_id"]

		if usertID == "" {
			return internal.Error(http.StatusBadRequest, ErrUserIDEmpty), nil
		}

		user, err := userRepo.GetUserScorebyUserID(usertID)
		if err != nil {
			if err == repositories.ErrUserNotFound {
				return internal.Error(http.StatusNotFound, err), nil
			}
			return internal.Error(http.StatusInternalServerError, err), nil
		}

		response := ScoreResponse{
			Username: user.Username,
			Score:    user.Score,
		}
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			return internal.Error(http.StatusInternalServerError, err), nil
		}

		return internal.Respond(http.StatusOK, string(jsonResponse)), nil
	}
}

func main() {
	//routesTable := os.Getenv("DYNAMODB_PICKING_ROUTES")
	// if routesTable == "" {
	// 	panic("DYNAMODB_PICKING_ROUTES cannot be empty")

	// handler := Adapter(userRepo)
	// lambda.Start(handler)
}
