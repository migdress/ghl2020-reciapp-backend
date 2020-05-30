package repositories

import (
	"errors"

	"github.com/Globhack/ghl2020-reciapp-backend/internal/models"
)

var ErrUserNotFound = errors.New("user not found")

type DynamoDBUsersRespository struct {
}

func NewDynamoDBUsersRepository(table string) *DynamoDBUsersRespository {
	return &DynamoDBUsersRespository{}
}

func (r *DynamoDBUsersRespository) FindByUsername(username string) (models.User, error) {
	return models.User{}, nil
}
