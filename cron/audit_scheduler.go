package cron

import (
	"dokuprime-be/audit" 
	"log"
	"dokuprime-be/config"
	"github.com/jmoiron/sqlx"
)

type AuditScheduler struct {
	auditService *audit.AuditService
}

func NewAuditScheduler(db *sqlx.DB) *AuditScheduler {
	
	repo := audit.NewAuditRepository(db)
	service := audit.NewAuditService(repo)
	
	return &AuditScheduler{
		auditService: service,
	}
}

func (s *AuditScheduler) RegisterJobs(scheduler *Scheduler) error {
	archiveCron := config.AppConfig.CronArchiveSchedule
	cleanupCron := config.AppConfig.CronCleanupSchedule

	if archiveCron == "" {
		archiveCron = "0 0 1 * * *"
	}
	if cleanupCron == "" {
		cleanupCron = "0 0 21 * * *"
	}

	
	err := scheduler.AddJob(archiveCron, func() {
		_ = s.auditService.ArchiveOldLogs()
	})
	if err != nil {
		log.Printf("Gagal mendaftarkan cron job arsip audit: %v", err)
		return err
	}
	
	
	err = scheduler.AddJob(cleanupCron, func() {
		_ = s.auditService.CleanupArchiveLogs()
	})
	if err != nil {
		log.Printf("Gagal mendaftarkan cron job retensi audit arsip: %v", err)
		return err
	}
	
	
	log.Printf("Audit cron jobs registered successfully - Archive: [%s], Cleanup: [%s]", archiveCron, cleanupCron)
	
	return nil
}