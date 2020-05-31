package models

import "time"

const (
	MaterialPlastic    = "plastic"
	MaterialMetal      = "metal"
	MaterialGlass      = "glass"
	MaterialPaper      = "paper"
	MaterialTechnology = "technology"

	RouteStatusOpen      = "open"      // Shows up to the user
	RouteStatusClosed    = "closed"    // Shows up to the gatherer
	RouteStatusAssigned  = "assigned"  // Shows up only to the assigned gatherer
	RouteStatusInitiated = "initiated" // Shows up only to the assigned gatherer when it's been initiated
	RouteStatusFinished  = "finished"  // Gathere has finished all the picking points
	RouteStatusCancelled = "cancelled" // to be defined
)

type PickingPoint struct {
	ID         string     `json:"id"`
	LocationID string     `json:"locationid"`
	Country    string     `json:"country"`
	City       string     `json:"city"`
	Latitude   float64    `json:"latitude"`
	Longitude  float64    `json:"longitude"`
	Address1   string     `json:"address1"`
	Address2   string     `json:"address2"`
	Materials  []string   `json:"materials"`
	PickedAt   *time.Time `json:"picked"`
	Created    *time.Time `json:"created"`
}

type Route struct {
	ID            string         `json:"id"`
	Sector        string         `json:"sector"`
	Shift         string         `json:"shift"`
	Materials     []string       `json:"materials"`
	Status        string         `json:"status"`
	GathererID    string         `json:"gatherer_id"`
	StartsAt      *time.Time     `json:"starts_at"`
	InitiatedAt   *time.Time     `json:"initiated_at"`
	FinishedAt    *time.Time     `json:"finished_at"`
	Created       *time.Time     `json:"created"`
	PickingPoints []PickingPoint `json:"picking_points"`
}
