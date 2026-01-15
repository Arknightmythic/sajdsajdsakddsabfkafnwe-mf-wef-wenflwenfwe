package cron

import (
	"dokuprime-be/config"
	"log"
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
	period := config.AppConfig.HelpdeskQueuePeriodMinutes

	threshold := time.Now().UTC().Add(-period)
	log.Println(threshold, "is the threshold time for updating helpdesk queue status")

	query := `
		UPDATE bkpm.helpdesk
		SET status = 'pending'
		WHERE (status = 'Queue' OR status = 'queue')
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
