package helpdesk

import "time"

type Helpdesk struct {
	ID               int       `db:"id" json:"id"`
	SessionID        string    `db:"session_id" json:"session_id"`
	Platform         string    `db:"platform" json:"platform"`
	PlatformUniqueID *string   `db:"platform_unique_id" json:"platform_unique_id"`
	Status           string    `db:"status" json:"status"`
	UserID           int       `db:"user_id" json:"user_id"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}


type SwitchHelpdesk struct {
	ID     int  `db:"id" json:"id"`
	Status bool `db:"status" json:"status"`
}

type HelpdeskSummary struct {
	Queue    int `db:"queue" json:"queue"`
	Active   int `db:"active" json:"active"`
	Pending  int `db:"pending" json:"pending"`
	Resolved int `db:"resolved" json:"resolved"`
}