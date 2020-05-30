package models

type Location struct {
	ID        string `json:"id"`
	CreatedBy string `json:"created_by"`
	Balance   string `json:"balance"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Country   string `json:"country"`
	City      string `json:"city"`
	State     string `json:"state"`
	Address1  string `json:"address1"`
	Address2  string `json:"address2"`
}
