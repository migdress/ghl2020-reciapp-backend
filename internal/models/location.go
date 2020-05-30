package models

type Location struct {
	ID        string  `json:"id"`
	CreatedBy string  `json:"created_by"`
	Balance   float64 `json:"balance"`
	Name      string  `json:"name"`
	Country   string  `json:"country"`
	City      string  `json:"city"`
	State     string  `json:"state"`
	Address1  string  `json:"address1"`
	Address2  string  `json:"address2"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}
