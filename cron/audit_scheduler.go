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

func (s *AuditScheduler) CleanupOldLogs() error {
	log.Println("[AuditScheduler] Memulai proses retensi audit_logs (Menghapus data > 15 hari)...")
	
	// 1. Ambil waktu saat ini dalam zona waktu Jakarta (WIB)
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.Local
	}
	now := time.Now().In(loc)

	// 2. Tentukan batas akhir (Cutoff): 15 Hari lalu, tepat di jam 23:59:59 WIB.
	// Jika cron jalan 16 Maret, maka cutoff ini akan mengarah ke 1 Maret 23:59:59 WIB.
	cutoffDateWIB := time.Date(now.Year(), now.Month(), now.Day()-15, 23, 59, 59, 999999999, loc)
	
	// 3. Konversi ke UTC karena data created_at di database menyimpannya dalam UTC
	cutoffUTC := cutoffDateWIB.UTC() 

	batchSize := 5000
	totalDeleted := 0
	batchCounter := 1

	for {
		// 4. Ubah query untuk menggunakan parameter $1 dan $2
		query := `
			DELETE FROM bkpm.audit_logs
			WHERE id IN (
				SELECT id FROM bkpm.audit_logs
				WHERE created_at <= $1
				LIMIT $2
			)
		`
		
		// 5. Eksekusi query dengan mengirim cutoffUTC secara eksplisit
		result, err := s.db.Exec(query, cutoffUTC, batchSize)
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