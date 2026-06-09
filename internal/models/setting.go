package models

// Setting represents a key-value pair for application settings
type Setting struct {
	Model
	Key   string `json:"key"`
	Value string `json:"value"`
}
