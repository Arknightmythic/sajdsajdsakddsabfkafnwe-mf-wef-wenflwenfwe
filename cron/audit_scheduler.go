package cron

import (
	"dokuprime-be/audit" // Sesuaikan nama modul 'dokuprime-be' jika berbeda
	"log"

	"github.com/jmoiron/sqlx"
)

type AuditScheduler struct {
	auditService *audit.AuditService
}

func NewAuditScheduler(db *sqlx.DB) *AuditScheduler {
	// Inisialisasi repository dan service audit khusus untuk scheduler ini
	repo := audit.NewAuditRepository(db)
	service := audit.NewAuditService(repo)
	
	return &AuditScheduler{
		auditService: service,
	}
}

func (s *AuditScheduler) RegisterJobs(scheduler *Scheduler) error {
	err := scheduler.AddJob("0 15 17 * * *", func() {
		_ = s.auditService.ArchiveOldLogs()
	})
	
	if err != nil {
		log.Printf("Gagal mendaftarkan cron job arsip audit: %v", err)
		return err
	}
	
	log.Println("Audit archive cron job registered successfully")
	return nil
}