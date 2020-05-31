package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Globhack/ghl2020-reciapp-backend/internal"
	"github.com/Globhack/ghl2020-reciapp-backend/internal/models"
	"github.com/Globhack/ghl2020-reciapp-backend/internal/repositories"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var ErrUsernameEmpty = errors.New("username cannot be empty")

type Handler func(ctx context.Context) (events.APIGatewayProxyResponse, error)

type RoutesRepoRepository interface {
	FindOpenShifts(currentTime time.Time, maxTime time.Time) ([]models.Route, error)
}

type TimeHelper interface {
	NowWithTimezone() (time.Time, error)
	ToISO8601(d time.Time) (string, error)
	ToLatamFormat(d time.Time) (string, error)
}

type ResponseShift struct {
	ID            string   `json:"id"`
	Materials     []string `json:"materials"`
	Sector        string   `json:"sector"`
	Shift         string   `json:"shift"`
	Date          string   `json:"date"`
	FormattedDate string   `json:"formatted_date"`
}

type Response struct {
	Shifts []ResponseShift `json:"shifts"`
}

func Adapter(
	routesRepo RoutesRepoRepository,
	daysOffset int,
	timeHelper TimeHelper,
) Handler {
	return func(ctx context.Context) (events.APIGatewayProxyResponse, error) {

		// Calculate window time to query for shifts
		now, err := timeHelper.NowWithTimezone()
		if err != nil {
			return internal.Error(http.StatusInternalServerError, err), nil
		}
		maxTime := now.AddDate(0, 0, daysOffset)

		// Query for routes
		log.Printf("finding shifts between (%v) and (%v)\n", now, maxTime)
		shifts, err := routesRepo.FindOpenShifts(now, maxTime)
		if err != nil {
			jsonResponse, _ := json.Marshal(Response{})
			if err == repositories.ErrNoOpenShifts {
				return internal.Respond(http.StatusOK, string(jsonResponse)), nil
			}

			return internal.Error(http.StatusInternalServerError, err), nil
		}
		log.Printf("got %v shifts\n %#v", len(shifts), shifts)

		// Prepare response
		responseRoutes := make([]ResponseShift, len(shifts))
		for i, route := range shifts {
			startsAt, err := timeHelper.ToISO8601(*route.StartsAt)
			if err != nil {
				return internal.Error(http.StatusInternalServerError, err), nil
			}

			latamDateFormat, err := timeHelper.ToLatamFormat(*route.StartsAt)
			if err != nil {
				return internal.Error(http.StatusInternalServerError, err), nil
			}

			responseRoutes[i] = ResponseShift{
				ID:            route.ID,
				Materials:     route.Materials,
				Sector:        route.Sector,
				Shift:         route.Shift,
				Date:          startsAt,
				FormattedDate: latamDateFormat,
			}
		}
		response := Response{
			Shifts: responseRoutes,
		}
		jsonResponse, err := json.Marshal(response)
		if err != nil {
			return internal.Error(http.StatusInternalServerError, err), nil
		}

		return internal.Respond(http.StatusOK, string(jsonResponse)), nil
	}
}

func main() {
	routesTable := os.Getenv("DYNAMODB_PICKING_ROUTES")
	if routesTable == "" {
		panic("DYNAMODB_PICKING_ROUTES cannot be empty")
	}

	locationsTable := os.Getenv("DYNAMODB_LOCATIONS")
	if locationsTable == "" {
		panic("DYNAMODB_LOCATIONS cannot be empty")
	}

	daysOffsetString := os.Getenv("DAYS_OFFSET")
	if daysOffsetString == "" {
		panic("DAYS_OFFSET cannot be empty")
	}

	daysOffset, err := strconv.Atoi(daysOffsetString)
	if err != nil {
		panic("DAYS_OFFSET must be an integer")
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

	routesRepo := repositories.NewDynamoDBRoutesRepository(
		dynamodbClient,
		routesTable,
		locationsTable,
		timeHelper,
		uuidHelper,
	)
	handler := Adapter(routesRepo, daysOffset, timeHelper)
	lambda.Start(handler)
}
