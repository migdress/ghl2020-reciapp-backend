package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Globhack/ghl2020-reciapp-backend/internal"
	"github.com/Globhack/ghl2020-reciapp-backend/internal/models"
	"github.com/Globhack/ghl2020-reciapp-backend/internal/repositories"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var ErrUserIDEmpty = errors.New("user_id cannot be empty")
var ErrRouteIDEmpty = errors.New("route_id  cannot be empty")
var ErrPickingPointIDEmpty = errors.New("picking_point_id  cannot be empty")
var ErrPickingPointNotFoundInRoute = errors.New("given picking point does not exist in route")
var ErrWrongUserType = errors.New("user must be of type gatherer")
var ErrWrongGathererID = errors.New("the route is not assigned to the given gatherer id")

type Handler func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

type UsersRepository interface {
	Find(userID string) (models.User, error)
}

type RoutesRepository interface {
	Find(routeID string) (models.Route, error)
	FinishPickingPoint(routeID string, pickingPointIndex int, locationID string, remaining int) error
}

type TimeHelper interface {
	ToLatamFormat(d time.Time) (string, error)
	ToISO8601(d time.Time) (string, error)
}

type Request struct {
	UserID         string `json:"user_id"`
	RouteID        string `json:"route_id"`
	PickingPointId string `json:"picking_point_id"`
}

type ResponsePickingPoint struct {
	ID         string   `json:"id"`
	LocationID string   `json:"location_id"`
	Country    string   `json:"country"`
	City       string   `json:"city"`
	Latitude   float64  `json:"latitude"`
	Longitude  float64  `json:"longitude"`
	Address1   string   `json:"address_1"`
	Address2   string   `json:"address_2"`
	Materials  []string `json:"materials"`
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

func Adapter(
	usersRepo UsersRepository,
	routesRepo RoutesRepository,
	timeHelper TimeHelper,
) Handler {
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
		if reqBody.PickingPointId == "" {
			return internal.Error(http.StatusBadRequest, ErrPickingPointIDEmpty), nil
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

		if user.ID != route.GathererID {
			return internal.Error(http.StatusForbidden, ErrWrongGathererID), nil
		}

		exists := false
		alreadyPicked := false
		pickingPointIndex := -1
		locationID := ""
		now := time.Now()
		remaining := len(route.PickingPoints)
		log.Printf("looping through (%v) picking points\n", len(route.PickingPoints))
		for i, pp := range route.PickingPoints {
			if pp.PickedAt != nil {
				remaining--
			}

			if pp.ID == reqBody.PickingPointId {
				log.Printf("found match! current index is (%v)\n", i)
				exists = true
				pickingPointIndex = i
				locationID = pp.LocationID

				if pp.PickedAt != nil {
					alreadyPicked = true
				}

				route.PickingPoints[i].PickedAt = &now
				break
			}
		}
		if !exists {
			return internal.Error(http.StatusUnprocessableEntity, ErrPickingPointNotFoundInRoute), nil
		}

		if !alreadyPicked {
			err = routesRepo.FinishPickingPoint(route.ID, pickingPointIndex, locationID, remaining)
			if err != nil {
				return internal.Error(http.StatusInternalServerError, err), nil
			}
		}

		responseRoutePickingPoints := []ResponsePickingPoint{}
		for _, pp := range route.PickingPoints {
			if pp.PickedAt == nil {
				responseRoutePickingPoints = append(responseRoutePickingPoints, ResponsePickingPoint{
					ID:         pp.ID,
					LocationID: pp.LocationID,
					Country:    pp.Country,
					City:       pp.City,
					Latitude:   pp.Latitude,
					Longitude:  pp.Longitude,
					Address1:   pp.Address1,
					Address2:   pp.Address2,
					Materials:  pp.Materials,
				})
			}
		}
		startsAt, err := timeHelper.ToISO8601(*route.StartsAt)
		if err != nil {
			return internal.Error(http.StatusInternalServerError, err), nil
		}

		status := route.Status
		if remaining == 1 {
			status = models.RouteStatusFinished
		}
		responseAssignedRoute := ResponseAssignedRoute{
			ID:            route.ID,
			Materials:     route.Materials,
			Sector:        route.Sector,
			Status:        status,
			Shift:         route.Shift,
			Date:          startsAt,
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
	usersTable := os.Getenv("DYNAMODB_USERS")
	if usersTable == "" {
		panic("DYNAMODB_USERS cannot be empty")
	}

	routesTable := os.Getenv("DYNAMODB_PICKING_ROUTES")
	if routesTable == "" {
		panic("DYNAMODB_PICKING_ROUTES cannot be empty")
	}

	timezone := os.Getenv("TIMEZONE")
	if timezone == "" {
		panic("TIMEZONE cannot be empty")
	}

	locationsTable := os.Getenv("DYNAMODB_LOCATIONS")
	if locationsTable == "" {
		panic("DYNAMODB_LOCATIONS cannot be empty")
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
	routesRepo := repositories.NewDynamoDBRoutesRepository(
		dynamodbClient,
		routesTable,
		locationsTable,
		timeHelper,
		uuidHelper,
	)

	handler := Adapter(usersRepo, routesRepo, timeHelper)
	lambda.Start(handler)
}
