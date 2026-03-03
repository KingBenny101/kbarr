package models

type Setting struct {
	ID    int    `db:"id"`
	Key   string `db:"key"`
	Value string `db:"value"`
}
