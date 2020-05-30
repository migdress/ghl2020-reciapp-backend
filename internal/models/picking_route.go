package models

const (
	Plastic    = "plastic"
	Metal      = "metal"
	Glass      = "glass"
	Paper      = "paper"
	Technology = "technology"
)

type PickingPoints struct {
	Name       string  `json:"name"`
	LocationID string  `json:"locationid"`
	Country    string  `json:"country"`
	City       string  `json:"city"`
	Latitude   float64 `json:"latitude"`
	Longitude  float64 `json:"longitude"`
	Address1   string  `json:"address1"`
	Address2   string  `json:"address2"`
}

type Routes struct {
	ID            string `json:"id"`
	Materials     []string
	Sector        string `json:"sector"`
	Shift         string `json:"shift"`
	Date          string `json:"date"`
	PickingPoints []PickingPoints
}
