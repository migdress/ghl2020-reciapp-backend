package repositories

import (
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/Globhack/ghl2020-reciapp-backend/internal/models"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var ErrRouteNotFound = errors.New("route not found")
var ErrRouteAlreadyAssigned = errors.New("route already assigned")
var ErrPickingPointAlreadyPinned = errors.New("picking point already pinned")

type TimeHelper interface {
	ToISO8601(d time.Time) (string, error)
	FromISO8601(d string) (time.Time, error)
}

type DynamoDBRoutesRepository struct {
	client      *dynamodb.DynamoDB
	tableRoutes string
	timeHelper  TimeHelper
}

func NewDynamoDBRoutesRepository(
	client *dynamodb.DynamoDB,
	tableRoutes string,
	timeHelper TimeHelper,
) *DynamoDBRoutesRepository {
	return &DynamoDBRoutesRepository{
		client:      client,
		tableRoutes: tableRoutes,
		timeHelper:  timeHelper,
	}
}

func (r *DynamoDBRoutesRepository) Find(routeID string) (models.Route, error) {
	out, err := r.client.Query(&dynamodb.QueryInput{
		TableName: aws.String(r.tableRoutes),
		KeyConditions: map[string]*dynamodb.Condition{
			"id": {
				ComparisonOperator: aws.String("EQ"),
				AttributeValueList: []*dynamodb.AttributeValue{
					{
						S: aws.String(routeID),
					},
				},
			},
		},
	})
	if err != nil {
		return models.Route{}, err
	}
	if len(out.Items) == 0 {
		return models.Route{}, ErrRouteNotFound
	}

	routes, err := r.hydrateRoutes(out.Items)
	if err != nil {
		return models.Route{}, err
	}
	return routes[0], nil
}

func (r *DynamoDBRoutesRepository) FindAvailableRoutes(
	currentTime time.Time,
	maxTime time.Time,
) ([]models.Route, error) {
	nowString, err := r.timeHelper.ToISO8601(currentTime)
	if err != nil {
		return nil, err
	}

	thenString, err := r.timeHelper.ToISO8601(maxTime)
	if err != nil {
		return nil, err
	}

	out, err := r.client.Query(&dynamodb.QueryInput{
		TableName:              aws.String(r.tableRoutes),
		IndexName:              aws.String("by_status_and_starts_at"),
		KeyConditionExpression: aws.String("#status = :closed AND starts_at BETWEEN :now AND :then"),
		FilterExpression:       aws.String("gatherer_id = :unassigned"),
		ExpressionAttributeNames: map[string]*string{
			"#status": aws.String("status"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":closed": {
				S: aws.String(models.RouteStatusClosed),
			},
			":now": {
				S: aws.String(nowString),
			},
			":then": {
				S: aws.String(thenString),
			},
			":unassigned": {
				S: aws.String("-"),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return r.hydrateRoutes(out.Items)
}

func (r *DynamoDBRoutesRepository) Assign(userID string, routeID string) error {
	_, err := r.client.TransactWriteItems(&dynamodb.TransactWriteItemsInput{
		TransactItems: []*dynamodb.TransactWriteItem{
			{
				Update: &dynamodb.Update{
					TableName: aws.String(r.tableRoutes),
					Key: map[string]*dynamodb.AttributeValue{
						"id": {
							S: aws.String(routeID),
						},
					},
					ConditionExpression: aws.String("gatherer_id = :unassigned"),
					UpdateExpression:    aws.String("set gatherer_id = :userID"),
					ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
						":unassigned": {
							S: aws.String("-"),
						},
						":userID": {
							S: aws.String(userID),
						},
					},
				},
			},
		},
	})
	if err != nil {
		log.Printf("routesRepo Assign error: %v\n", err)
		if strings.Contains(err.Error(), "cancelled") {
			return ErrRouteAlreadyAssigned
		}
		return err
	}
	return nil
}

func (r *DynamoDBRoutesRepository) hydrateRoutes(items []map[string]*dynamodb.AttributeValue) ([]models.Route, error) {
	routes := make([]models.Route, len(items))
	for i, item := range items {

		route := models.Route{}
		if v, ok := item["id"]; ok {
			route.ID = *v.S
		}
		if v, ok := item["sector"]; ok {
			route.Sector = *v.S
		}
		if v, ok := item["shift"]; ok {
			route.Shift = *v.S
		}
		if v, ok := item["gatherer_id"]; ok && *v.S != "-" {
			route.GathererID = *v.S
		}
		if v, ok := item["materials"]; ok {
			materials := make([]string, len(v.L))
			for i, s := range v.L {
				materials[i] = *s.S
			}
			route.Materials = materials
		}
		if v, ok := item["status"]; ok {
			route.Status = *v.S
		}
		if v, ok := item["starts_at"]; ok {
			parsedTime, err := r.timeHelper.FromISO8601(*v.S)
			if err != nil {
				return nil, err
			}
			route.StartsAt = &parsedTime
		}
		if v, ok := item["finished_at"]; ok && *v.S != "-" {
			parsedTime, err := r.timeHelper.FromISO8601(*v.S)
			if err != nil {
				return nil, err
			}
			route.FinishedAt = &parsedTime
		}
		if v, ok := item["created"]; ok && *v.S != "-" {
			parsedTime, err := r.timeHelper.FromISO8601(*v.S)
			if err != nil {
				return nil, err
			}
			route.FinishedAt = &parsedTime
		}
		if v, ok := item["picking_points"]; ok {
			pickingPoints, err := r.hydratePickingPoints(v.L)
			if err != nil {
				return nil, err
			}
			route.PickingPoints = pickingPoints
		}
		routes[i] = route
	}
	return routes, nil
}

func (r *DynamoDBRoutesRepository) hydratePickingPoints(
	items []*dynamodb.AttributeValue,
) ([]models.PickingPoint, error) {
	pickingPoints := make([]models.PickingPoint, len(items))
	for i, item := range items {
		pp := models.PickingPoint{}
		if v, ok := item.M["id"]; ok {
			pp.ID = *v.S
		}
		if v, ok := item.M["name"]; ok {
			pp.ID = *v.S
		}
		if v, ok := item.M["location_id"]; ok {
			pp.ID = *v.S
		}
		if v, ok := item.M["country"]; ok {
			pp.ID = *v.S
		}
		if v, ok := item.M["city"]; ok {
			pp.ID = *v.S
		}
		if v, ok := item.M["latitude"]; ok {
			floatVal, err := strconv.ParseFloat(*v.N, 64)
			if err != nil {
				return nil, err
			}
			pp.Latitude = floatVal
		}
		if v, ok := item.M["longitude"]; ok {
			floatVal, err := strconv.ParseFloat(*v.N, 64)
			if err != nil {
				return nil, err
			}
			pp.Longitude = floatVal
		}
		if v, ok := item.M["address_1"]; ok {
			pp.ID = *v.S
		}
		if v, ok := item.M["address_2"]; ok {
			pp.ID = *v.S
		}
		if v, ok := item.M["picked_at"]; ok && *v.S != "-" {
			timeVal, err := r.timeHelper.FromISO8601(*v.S)
			if err != nil {
				return nil, err
			}
			pp.PickedAt = &timeVal
		}
		if v, ok := item.M["created"]; ok && *v.S != "-" {
			timeVal, err := r.timeHelper.FromISO8601(*v.S)
			if err != nil {
				return nil, err
			}
			pp.PickedAt = &timeVal
		}
		pickingPoints[i] = pp
	}
	return pickingPoints, nil
}
