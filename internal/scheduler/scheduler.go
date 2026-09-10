package scheduler

import (
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron *cron.Cron
	loc  *time.Location
}

func New(tz string) (*Scheduler, error) {
	loc, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %q: %w", tz, err)
	}

	c := cron.New(cron.WithLocation(loc), cron.WithLogger(&cronLogger{}))

	return &Scheduler{cron: c, loc: loc}, nil
}

func (s *Scheduler) Start() {
	s.cron.Start()
}

func (s *Scheduler) Stop() {
	s.cron.Stop()
}

func (s *Scheduler) AddJob(spec string, job func()) error {
	_, err := s.cron.AddFunc(spec, job)
	if err != nil {
		return fmt.Errorf("add cron job %q: %w", spec, err)
	}
	return nil
}

type cronLogger struct{}

func (l *cronLogger) Info(msg string, keysAndValues ...interface{}) {
	log.Printf("cron: %s %v", msg, keysAndValues)
}

func (l *cronLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	log.Printf("cron ERROR: %s err=%v %v", msg, err, keysAndValues)
}
