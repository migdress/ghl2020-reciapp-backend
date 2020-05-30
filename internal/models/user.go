package models

const (
	UsernameTypeUser     = "user"
	UsernameTypeGatherer = "gatherer"
)

type User struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Firstname string `json:"firstname"`
	Lastname  string `json:"lastname"`
	Type      string `json:"type"`
	Country   string `json:"country"`
}
