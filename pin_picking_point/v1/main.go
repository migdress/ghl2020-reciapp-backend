package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/Globhack/ghl2020-reciapp-backend/internal"
	"github.com/Globhack/ghl2020-reciapp-backend/internal/models"
	"github.com/Globhack/ghl2020-reciapp-backend/internal/repositories"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var ErrUserIDEmpty = errors.New("user_id cannot be empty")
var ErrShiftIdEmpty = errors.New("shift_id cannot be empty")
var ErrShiftNotFound = errors.New("shift not found")
var ErrLocationIDEmpty = errors.New("location_id cannot be empty")
var ErrMaterialsEmpty = errors.New("materials cannot be empty")
var ErrMaterialNotAllowed = errors.New("one or more materials are not allowed")
var ErrShiftIsClosed = errors.New("the shift has been closed and it's not receiving more picking_points")

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
	UserID     string   `json:"user_id"`
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
			return internal.Error(http.StatusBadRequest, ErrUserIDEmpty), nil
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
			return internal.Error(http.StatusInternalServerError, err), nil
		}

		route, err := routesRepo.Find(reqBody.ShiftID)
		if err != nil {
			if err == repositories.ErrRouteNotFound {
				return internal.Error(http.StatusNotFound, ErrShiftNotFound), nil
			}
			return internal.Error(http.StatusInternalServerError, err), nil
		}

		// material validation
		for _, material := range reqBody.Materials {
			if !isMaterialAllowed(strings.TrimSpace(material), route.Materials) {
				return internal.Error(http.StatusUnprocessableEntity, ErrMaterialNotAllowed), nil
			}
		}

		// Check if the shift (picking_route) is still open
		if route.Status != models.RouteStatusOpen {
			return internal.Error(http.StatusConflict, ErrShiftIsClosed), nil
		}

		location, err := locationRepo.Find(reqBody.LocationID)
		if err != nil {
			if err == repositories.ErrLocationNotFound {
				return internal.Error(http.StatusNotFound, err), nil
			}
			return internal.Error(http.StatusInternalServerError, err), nil
		}

		// Check if the location is already on the route.picking_points
		log.Printf("checking if location is already on route.picking_points\n")
		for _, pickingPoint := range route.PickingPoints {

			log.Printf("pickingPoint.LocationID (%v) == location.ID(%v)\n", pickingPoint.LocationID, location.ID)
			if pickingPoint.LocationID == location.ID {
				log.Printf("match! returning 200\n")
				return internal.Respond(http.StatusOK, ""), nil
			}
		}

		err = routesRepo.Pin(reqBody.UserID, location, reqBody.ShiftID, reqBody.Materials)
		if err != nil {
			return internal.Error(http.StatusInternalServerError, err), nil
		}

		return internal.Respond(http.StatusOK, ""), nil
	}
}

func isMaterialAllowed(material string, allowedMaterials []string) bool {
	for _, allowed := range allowedMaterials {
		if material == allowed {
			return true
		}
	}
	return false
}

func main() {
	routesTable := os.Getenv("DYNAMODB_PICKING_ROUTES")
	if routesTable == "" {
		panic("DYNAMODB_PICKING_ROUTES cannot be empty")
	}

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

	timezone := os.Getenv("TIMEZONE")
	if timezone == "" {
		panic("TIMEZONE cannot be empty")
	}

	timeHelper, err := internal.NewTimeHelper(timezone)
	if err != nil {
		panic(err)
	}

	uuidHelper := internal.NewUUIDHelper()

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
	routesRepo := repositories.NewDynamoDBRoutesRepository(
		dynamodbClient,
		routesTable,
		timeHelper,
		uuidHelper,
	)

	handler := Adapter(routesRepo, usersRepo, locationsRepo)
	lambda.Start(handler)
}
