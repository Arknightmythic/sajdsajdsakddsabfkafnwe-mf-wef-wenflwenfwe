package team

import "github.com/lib/pq"

type Team struct {
	ID    int            `db:"id" json:"id"`
	Name  string         `db:"name" json:"name"`
	Pages pq.StringArray `db:"pages" json:"pages"`
}
