package repositories

import (
	"errors"

	"github.com/Globhack/ghl2020-reciapp-backend/internal/models"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var ErrUserNotFound = errors.New("user not found")

type DynamoDBUsersRepository struct {
	client     *dynamodb.DynamoDB
	tableUsers string
}

func NewDynamoDBUsersRepository(client *dynamodb.DynamoDB, tableUsers string) *DynamoDBUsersRepository {
	return &DynamoDBUsersRepository{
		client:     client,
		tableUsers: tableUsers,
	}
}

func (r *DynamoDBUsersRepository) FindByUsername(username string) (models.User, error) {
	out, err := r.client.Query(&dynamodb.QueryInput{
		TableName:              aws.String(r.tableUsers),
		IndexName:              aws.String("by_username"),
		KeyConditionExpression: aws.String("username = :username"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":username": {
				S: aws.String(username),
			},
		},
	})
	if err != nil {
		return models.User{}, err
	}
	if len(out.Items) == 0 {
		return models.User{}, ErrUserNotFound
	}

	return r.hydrate(out.Items[0]), nil
}

func (r *DynamoDBUsersRepository) hydrate(
	item map[string]*dynamodb.AttributeValue,
) models.User {
	user := models.User{}
	if v, ok := item["id"]; ok {
		user.ID = *v.S
	}
	if v, ok := item["username"]; ok {
		user.Username = *v.S
	}
	if v, ok := item["firstname"]; ok {
		user.Firstname = *v.S
	}
	if v, ok := item["lastname"]; ok {
		user.Lastname = *v.S
	}
	if v, ok := item["type"]; ok {
		user.Type = *v.S
	}
	if v, ok := item["country"]; ok {
		user.Country = *v.S
	}
	return user
}
