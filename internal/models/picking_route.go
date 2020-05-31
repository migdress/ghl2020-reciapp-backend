package models

const (
	Plastic    = "plastic"
	Metal      = "metal"
	Glass      = "glass"
	Paper      = "paper"
	Technology = "technology"
)

type PickingPoint struct {
	Name       string  `json:"name"`
	LocationID string  `json:"locationid"`
	Country    string  `json:"country"`
	City       string  `json:"city"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Address1   string  `json:"address1"`
	Address2   string  `json:"address2"`
}

type Route struct {
	ID            string `json:"id"`
	Materials     []string
	Status        string `json:"status"`
	Sector        string `json:"sector"`
	Shift         string `json:"shift"`
	Date          string `json:"date"`
	PickingPoints []PickingPoint
}
