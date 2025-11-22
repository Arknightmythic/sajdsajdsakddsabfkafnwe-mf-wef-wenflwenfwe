package guide

import "time"

type Guide struct {
	ID               int       `db:"id" json:"id"`
	Title            string    `db:"title" json:"title"`
	Description      string    `db:"description" json:"description"`
	Filename         string    `db:"filename" json:"filename"`          // Nama file unik di server
	OriginalFilename string    `db:"original_filename" json:"original_filename"` // Nama asli file
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
	UpdatedAt        time.Time `db:"updated_at" json:"updated_at"`
}

type GuideFilter struct {
	Search        string
	Limit         int
	Offset        int
	SortBy        string
	SortDirection string
}