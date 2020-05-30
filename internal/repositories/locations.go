package repositories

import (
	"errors"

	"github.com/Globhack/ghl2020-reciapp-backend/internal/models"
)

var ErrNoLocationsFound = errors.New("no_locations_found")

type DynamoDBLocationsRespository struct {
}

func NewDynamoDBLocationsRepository(table string) *DynamoDBLocationsRespository {
	return &DynamoDBLocationsRespository{}
}

func (r *DynamoDBLocationsRespository) FindByUserID(id string) ([]models.Location, error) {
	return []models.Location{}, nil
}
