package models

import "time"

const (
	MaterialPlastic    = "plastic"
	MaterialMetal      = "metal"
	MaterialGlass      = "glass"
	MaterialPaper      = "paper"
	MaterialTechnology = "technology"

	RouteStatusOpen      = "open"
	RouteStatusClosed    = "closed"
	RouteStatusFinished  = "finished"
	RouteStatusCancelled = "finished"
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
	FinishedAt    *time.Time     `json:"finished_at"`
	Created       *time.Time     `json:"created"`
	PickingPoints []PickingPoint `json:"picking_points"`
}
