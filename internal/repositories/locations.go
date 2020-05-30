package repositories

import (
	"errors"
	"log"
	"strconv"

	"github.com/Globhack/ghl2020-reciapp-backend/internal/models"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var ErrNoLocationsFound = errors.New("no_locations_found")

type DynamoDBLocationsRespository struct {
	client             *dynamodb.DynamoDB
	tableUserLocations string
	tableLocations     string
}

func NewDynamoDBLocationsRepository(client *dynamodb.DynamoDB, tableUserLocations string, tableLocations string) *DynamoDBLocationsRespository {
	return &DynamoDBLocationsRespository{
		client:             client,
		tableUserLocations: tableUserLocations,
		tableLocations:     tableLocations,
	}
}

func (r *DynamoDBLocationsRespository) FindByUserID(id string) ([]models.Location, error) {
	log.Printf("Finding user_locations by user id (%s)\n", id)
	out, err := r.client.Query(&dynamodb.QueryInput{
		TableName:              aws.String(r.tableUserLocations),
		IndexName:              aws.String("by_user_id"),
		KeyConditionExpression: aws.String("user_id = :userID"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":userID": {
				S: aws.String(id),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(out.Items) == 0 {
		return nil, ErrNoLocationsFound
	}

	log.Printf("Finding locations..\n")
	userLocations := make([]models.Location, len(out.Items))
	for i, item := range out.Items {
		locationID := item["location_id"].S
		out, err := r.client.Query(&dynamodb.QueryInput{
			TableName: aws.String(r.tableLocations),
			KeyConditions: map[string]*dynamodb.Condition{
				"id": {
					ComparisonOperator: aws.String("EQ"),
					AttributeValueList: []*dynamodb.AttributeValue{
						{
							S: locationID,
						},
					},
				},
			},
		})
		if err != nil {
			return nil, err
		}

		userLocations[i], err = r.hydrateLocation(out.Items[0])
		if err != nil {
			return nil, err
		}
	}

	return userLocations, nil
}

func (r *DynamoDBLocationsRespository) hydrateLocation(
	item map[string]*dynamodb.AttributeValue,
) (models.Location, error) {
	location := models.Location{}
	if v, ok := item["id"]; ok {
		location.ID = *v.S
	}
	if v, ok := item["created_by"]; ok {
		location.CreatedBy = *v.S
	}
	if v, ok := item["balance"]; ok {
		floatVal, err := strconv.ParseFloat(*v.N, 64)
		if err != nil {
			return models.Location{}, err
		}
		location.Balance = floatVal
	}
	if v, ok := item["name"]; ok {
		location.Name = *v.S
	}
	if v, ok := item["country"]; ok {
		location.Country = *v.S
	}
	if v, ok := item["city"]; ok {
		location.City = *v.S
	}
	if v, ok := item["state"]; ok {
		location.State = *v.S
	}
	if v, ok := item["address_1"]; ok {
		location.Address1 = *v.S
	}
	if v, ok := item["address_2"]; ok {
		location.Address2 = *v.S
	}
	if v, ok := item["latitude"]; ok {
		floatVal, err := strconv.ParseFloat(*v.N, 64)
		if err != nil {
			return models.Location{}, err
		}
		location.Latitude = floatVal
	}
	if v, ok := item["longitude"]; ok {
		floatVal, err := strconv.ParseFloat(*v.N, 64)
		if err != nil {
			return models.Location{}, err
		}
		location.Longitude = floatVal

	}
	return location, nil
}
