package cron

import (
	"log"
	"time"
	"dokuprime-be/config"

	"github.com/jmoiron/sqlx"
)

type AuditScheduler struct {
	db *sqlx.DB
}

func NewAuditScheduler(db *sqlx.DB) *AuditScheduler {
	return &AuditScheduler{
		db: db,
	}
}

// Logika pembersihan disatukan langsung di sini
func (s *AuditScheduler) CleanupOldLogs() error {
	log.Println("[AuditScheduler] Memulai proses retensi audit_logs (Menghapus data > 30 hari)...")
	
	batchSize := 5000
	totalDeleted := 0
	batchCounter := 1

	for {
		query := `
			DELETE FROM bkpm.audit_logs
			WHERE id IN (
				SELECT id FROM bkpm.audit_logs
				WHERE created_at < CURRENT_DATE - INTERVAL '15 days'
				LIMIT $1
			)
		`
		
		result, err := s.db.Exec(query, batchSize)
		if err != nil {
			log.Printf("[AuditScheduler] Error saat mengeksekusi batch ke-%d: %v", batchCounter, err)
			return err
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return err
		}

		if rowsAffected > 0 {
			log.Printf("[AuditScheduler] Proses Retensi - Batch ke-%d: Menghapus %d baris...", batchCounter, rowsAffected)
		}

		totalDeleted += int(rowsAffected)

		if rowsAffected == 0 {
			break
		}
		
		batchCounter++
		time.Sleep(1 * time.Second)
	}

	log.Printf("[AuditScheduler] Selesai! Total %d baris data log lama berhasil dihapus.", totalDeleted)
	
	_, _ = s.db.Exec("VACUUM ANALYZE bkpm.audit_logs;")
	
	return nil
}

func (s *AuditScheduler) RegisterJobs(scheduler *Scheduler) error {
	cleanupCron := config.AppConfig.CronCleanupSchedule

	if cleanupCron == "" {
		cleanupCron = "0 0 1 * * *" // Setelan default jam 1 pagi
	}
	
	err := scheduler.AddJob(cleanupCron, func() {
		_ = s.CleanupOldLogs()
	})
	if err != nil {
		log.Printf("Gagal mendaftarkan cron job retensi audit: %v", err)
		return err
	}
	
	log.Printf("Audit cron job registered successfully - Retention: [%s]", cleanupCron)
	return nil
}