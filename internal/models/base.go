package models

import "time"

// Model provides common persistence fields embedded in all domain models.
type Model struct {
	ID        uint       `bun:"id,pk,autoincrement" json:"ID"`
	CreatedAt time.Time  `bun:"created_at,notnull,default:now()" json:"CreatedAt"`
	UpdatedAt time.Time  `bun:"updated_at,notnull,default:now()" json:"UpdatedAt"`
	DeletedAt *time.Time `bun:"deleted_at,soft_delete,nullzero" json:"DeletedAt,omitempty"`
}
