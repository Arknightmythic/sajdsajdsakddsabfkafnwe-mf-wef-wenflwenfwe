package document

import "time"

type Document struct {
	ID       int    `db:"id" json:"id"`
	Category string `db:"category" json:"category"`
}

type DocumentDetail struct {
	ID           int       `db:"id" json:"id"`
	DocumentID   int       `db:"document_id" json:"document_id"`
	DocumentName string    `db:"document_name" json:"document_name"`
	Filename     string    `db:"filename" json:"filename"`
	DataType     string    `db:"data_type" json:"data_type"`
	Staff        string    `db:"staff" json:"staff"`
	Team         string    `db:"team" json:"team"`
	Status       *string   `db:"status" json:"status"`
	IsLatest     *bool     `db:"is_latest" json:"is_latest"`
	IsApprove    *bool     `db:"is_approve" json:"is_approve"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	Category     string    `db:"category" json:"category"`
	IngestStatus *string   `db:"ingest_status" json:"ingest_status"`
}

type DocumentWithDetail struct {
	ID           int       `db:"id" json:"id"`
	Category     string    `db:"category" json:"category"`
	DocumentName string    `db:"document_name" json:"document_name"`
	Filename     string    `db:"filename" json:"filename"`
	DataType     string    `db:"data_type" json:"data_type"`
	Staff        string    `db:"staff" json:"staff"`
	Team         string    `db:"team" json:"team"`
	Status       *string   `db:"status" json:"status"`
	IsLatest     *bool     `db:"is_latest" json:"is_latest"`
	IsApprove    *bool     `db:"is_approve" json:"is_approve"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	IngestStatus *string   `db:"ingest_status" json:"ingest_status"`
}

type DocumentFilter struct {
	Search        string
	DataType      string
	Category      string
	Status        string
	Limit         int
	Offset        int
	SortBy        string
	SortDirection string
	StartDate     *time.Time
	EndDate       *time.Time
	IngestStatus  string
}

type DocumentDetailFilter struct {
	Search        string
	DataType      string
	Category      string
	Status        string
	DocumentName  string
	Limit         int
	Offset        int
	SortBy        string
	SortDirection string
	StartDate     *time.Time
	EndDate       *time.Time
}
