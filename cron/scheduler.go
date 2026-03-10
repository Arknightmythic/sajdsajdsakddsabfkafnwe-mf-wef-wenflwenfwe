package cron

import (
	"log"
	"time"
	_ "time/tzdata"
	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron *cron.Cron
}


func NewScheduler() *Scheduler {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		loc = time.Local
	}
	
	c := cron.New(cron.WithSeconds(), cron.WithLocation(loc))
	return &Scheduler{
		cron: c,
	}
}


func (s *Scheduler) Start() {
	log.Println("Starting scheduler...")
	s.cron.Start()
}


func (s *Scheduler) Stop() {
	log.Println("Stopping scheduler...")
	ctx := s.cron.Stop()

	
	select {
	case <-ctx.Done():
		log.Println("Scheduler stopped successfully")
	case <-time.After(30 * time.Second):
		log.Println("Scheduler stop timeout reached")
	}
}


func (s *Scheduler) AddJob(spec string, cmd func()) error {
	_, err := s.cron.AddFunc(spec, cmd)
	if err != nil {
		log.Printf("Error adding cron job: %v", err)
		return err
	}
	return nil
}
