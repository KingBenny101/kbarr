package models

type Setting struct {
	ID    int    `json:"id"    gorm:"primaryKey;autoIncrement"`
	Key   string `json:"key"   gorm:"uniqueIndex;not null"`
	Value string `json:"value"`
}
