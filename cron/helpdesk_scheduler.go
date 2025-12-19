package cron

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/jmoiron/sqlx"
)

type HelpdeskScheduler struct {
	db *sqlx.DB
}

func NewHelpdeskScheduler(db *sqlx.DB) *HelpdeskScheduler {
	return &HelpdeskScheduler{
		db: db,
	}
}

func (h *HelpdeskScheduler) UpdateQueuedToPending() {
	periodStr := os.Getenv("HELPDESK_QUEUE_PERIOD_MINUTES")
	period, err := strconv.Atoi(periodStr)
	if err != nil || period <= 0 {
		period = 15
		log.Printf("Using default period: %d minutes", period)
	}

	threshold := time.Now().UTC().Add(-time.Duration(period) * time.Minute)
	query := `
		UPDATE bkpm.helpdesk
		SET status = 'pending'
		WHERE (status = 'Queue' OR status = 'queue') -- Tambahkan kurung di sini
		AND created_at <= $1
	`

	result, err := h.db.Exec(query, threshold)
	if err != nil {
		log.Printf("Error updating helpdesk queue status: %v", err)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Error getting rows affected: %v", err)
		return
	}

	if rowsAffected > 0 {
		log.Printf("Updated %d helpdesk record(s) from queue to pending", rowsAffected)
	}
}

func (h *HelpdeskScheduler) RegisterJobs(scheduler *Scheduler) error {

	err := scheduler.AddJob("0 * * * * *", h.UpdateQueuedToPending)
	if err != nil {
		return err
	}

	log.Println("Helpdesk scheduler jobs registered successfully")
	return nil
}
