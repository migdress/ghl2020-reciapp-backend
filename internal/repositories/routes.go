package repositories

import (
	"errors"
	"time"

	"github.com/Globhack/ghl2020-reciapp-backend/internal/models"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var ErrRouteNotFound = errors.New("route not found")

var ErrRouteAlreadyAssigned = errors.New("route already assigned")

var ErrPickingPointAlreadyPinned = errors.New("picking point already pinned")

type DynamoDBRoutesRepository struct {
	client      *dynamodb.DynamoDB
	tableRoutes string
}

func NewDynamoDBRoutesRepository(client *dynamodb.DynamoDB, tableRoutes string) *DynamoDBRoutesRepository {
	return &DynamoDBRoutesRepository{
		client:      client,
		tableRoutes: tableRoutes,
	}
}

func (r *DynamoDBRoutesRepository) FindAvailableRoutes(
	currentTime time.Time,
	maxTime time.Time,
) ([]models.Route, error) {
	return []models.Route{}, nil
}
