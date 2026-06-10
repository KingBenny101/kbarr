package models

import "github.com/uptrace/bun"

// Setting represents a key-value pair for application settings.
type Setting struct {
	bun.BaseModel `bun:"table:settings,alias:s"`
	Model
	Key   string  `bun:"key,notnull,unique" json:"key"`
	Value *string `bun:"value" json:"value"`
}
