package repositories

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/Globhack/ghl2020-reciapp-backend/internal/models"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var ErrRouteNotFound = errors.New("route not found")
var ErrNoAssignedRoutes = errors.New("no routes assigned")
var ErrRouteAlreadyAssigned = errors.New("route already assigned")
var ErrPickingPointAlreadyPinned = errors.New("picking point already pinned")
var ErrNoOpenShifts = errors.New("there is no open shifts")

type TimeHelper interface {
	ToISO8601(d time.Time) (string, error)
	FromISO8601(d string) (time.Time, error)
	NowWithTimezoneISO8601() (string, error)
}

type UUIDHelper interface {
	New() string
}

type DynamoDBRoutesRepository struct {
	client      *dynamodb.DynamoDB
	tableRoutes string
	timeHelper  TimeHelper
	uuidHelper  UUIDHelper
}

func NewDynamoDBRoutesRepository(
	client *dynamodb.DynamoDB,
	tableRoutes string,
	timeHelper TimeHelper,
	uuidHelper UUIDHelper,
) *DynamoDBRoutesRepository {
	return &DynamoDBRoutesRepository{
		client:      client,
		tableRoutes: tableRoutes,
		timeHelper:  timeHelper,
		uuidHelper:  uuidHelper,
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

func (r *DynamoDBRoutesRepository) Initiate(routeID string) error {
	log.Printf("routesRepo: Initiating route..")
	nowString, err := r.timeHelper.NowWithTimezoneISO8601()
	if err != nil {
		return err
	}

	_, err = r.client.UpdateItem(&dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableRoutes),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(routeID),
			},
		},
		UpdateExpression: aws.String("set initiated_at = :now, #status = :initiated"),
		ExpressionAttributeNames: map[string]*string{
			"#status": aws.String("status"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":now": {
				S: aws.String(nowString),
			},
			":initiated": {
				S: aws.String(models.RouteStatusInitiated),
			},
		},
	})

	return err
}

func (r *DynamoDBRoutesRepository) GetAssignedRoutesbyUserID(userID string) ([]models.Route, error) {
	out, err := r.client.Query(&dynamodb.QueryInput{
		TableName:              aws.String(r.tableRoutes),
		IndexName:              aws.String("by_gatherer_id_and_status"),
		KeyConditionExpression: aws.String("gatherer_id = :userID and #status = :assigned"),
		ExpressionAttributeNames: map[string]*string{
			"#status": aws.String("status"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":userID": {
				S: aws.String(userID),
			},
			":assigned": {
				S: aws.String(models.RouteStatusAssigned),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	if len(out.Items) == 0 {
		return nil, ErrNoAssignedRoutes
	}

	return r.hydrateRoutes(out.Items)
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

func (r *DynamoDBRoutesRepository) FindOpenShifts(
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
		KeyConditionExpression: aws.String("#status = :open AND starts_at BETWEEN :now AND :then"),
		ExpressionAttributeNames: map[string]*string{
			"#status": aws.String("status"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":open": {
				S: aws.String(models.RouteStatusOpen),
			},
			":now": {
				S: aws.String(nowString),
			},
			":then": {
				S: aws.String(thenString),
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
					UpdateExpression:    aws.String("set gatherer_id = :userID, #status = :assigned"),
					ExpressionAttributeNames: map[string]*string{
						"#status": aws.String("status"),
					},
					ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
						":unassigned": {
							S: aws.String("-"),
						},
						":userID": {
							S: aws.String(userID),
						},
						":assigned": {
							S: aws.String(models.RouteStatusAssigned),
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

func (r *DynamoDBRoutesRepository) Pin(userID string, location models.Location, shiftID string, materials []string) error {
	route, err := r.Find(shiftID)
	if err != nil {
		return err
	}

	route.PickingPoints = append(route.PickingPoints, models.PickingPoint{
		ID:         r.uuidHelper.New(),
		LocationID: location.ID,
		Country:    location.Country,
		City:       location.City,
		Latitude:   location.Latitude,
		Longitude:  location.Longitude,
		Address1:   location.Address1,
		Address2:   location.Address2,
		Materials:  materials,
	})
	mapPickingPoints, err := r.hydratePickingPointsMap(route.PickingPoints)
	if err != nil {
		return err
	}

	_, err = r.client.UpdateItem(&dynamodb.UpdateItemInput{
		TableName: aws.String(r.tableRoutes),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(route.ID),
			},
		},
		UpdateExpression: aws.String("set picking_points = :pickingPoints"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":pickingPoints": mapPickingPoints,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *DynamoDBRoutesRepository) hydratePickingPointsMap(pickingPoints []models.PickingPoint) (*dynamodb.AttributeValue, error) {
	log.Printf("routes repo: hydratePickingPointsMap: hydrating (%v) pickingPointMaps\n", len(pickingPoints))
	items := make([]*dynamodb.AttributeValue, len(pickingPoints))
	for i, pickingPoint := range pickingPoints {
		nowString, err := r.timeHelper.NowWithTimezoneISO8601()
		if err != nil {
			return nil, err
		}
		items[i] = &dynamodb.AttributeValue{
			M: map[string]*dynamodb.AttributeValue{
				"id": {
					S: aws.String(pickingPoint.ID),
				},
				"location_id": {
					S: aws.String(pickingPoint.LocationID),
				},
				"country": {
					S: aws.String(pickingPoint.Country),
				},
				"city": {
					S: aws.String(pickingPoint.City),
				},
				"latitude": {
					N: aws.String(fmt.Sprintf("%f", pickingPoint.Latitude)),
				},
				"longitude": {
					N: aws.String(fmt.Sprintf("%f", pickingPoint.Longitude)),
				},
				"address_1": {
					S: aws.String(pickingPoint.Address1),
				},
				"materials": {
					L: r.hydratePickingPointMaterials(pickingPoint.Materials),
				},
				"address_2": {
					S: aws.String(pickingPoint.Address2),
				},
				"picked_at": {
					S: aws.String("-"),
				},
				"created": {
					S: aws.String(nowString),
				},
			},
		}
	}
	return &dynamodb.AttributeValue{
		L: items,
	}, nil
}

func (r *DynamoDBRoutesRepository) hydratePickingPointMaterials(materials []string) []*dynamodb.AttributeValue {
	log.Printf("routes repo: hydratePickingPointMaterials: hydrating (%v) pickingPointMaterials\n", len(materials))
	items := make([]*dynamodb.AttributeValue, len(materials))
	for i, material := range materials {
		items[i] = &dynamodb.AttributeValue{
			S: aws.String(material),
		}
	}
	return items
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
		if v, ok := item["initiated_at"]; ok && *v.S != "-" {
			parsedTime, err := r.timeHelper.FromISO8601(*v.S)
			if err != nil {
				return nil, err
			}
			route.InitiatedAt = &parsedTime
		}
		if v, ok := item["created"]; ok && *v.S != "-" {
			parsedTime, err := r.timeHelper.FromISO8601(*v.S)
			if err != nil {
				return nil, err
			}
			route.Created = &parsedTime
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
		if v, ok := item.M["location_id"]; ok {
			pp.LocationID = *v.S
		}
		if v, ok := item.M["country"]; ok {
			pp.Country = *v.S
		}
		if v, ok := item.M["city"]; ok {
			pp.City = *v.S
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
			pp.Address1 = *v.S
		}
		if v, ok := item.M["address_2"]; ok {
			pp.Address2 = *v.S
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
			pp.Created = &timeVal
		}
		if v, ok := item.M["materials"]; ok {
			materials := make([]string, len(v.L))
			for i, s := range v.L {
				materials[i] = *s.S
			}
			pp.Materials = materials

		}
		pickingPoints[i] = pp
	}
	return pickingPoints, nil
}
