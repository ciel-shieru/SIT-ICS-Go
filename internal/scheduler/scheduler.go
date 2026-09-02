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

	c := cron.New(cron.WithLocation(loc), cron.WithLogger(&logger{}))

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

func (s *Scheduler) Location() *time.Location {
	return s.loc
}

type logger struct{}

func (l *logger) Info(msg string, keysAndValues ...interface{}) {
	log.Printf("cron: %s %v", msg, keysAndValues)
}

func (l *logger) Error(err error, msg string, keysAndValues ...interface{}) {
	log.Printf("cron ERROR: %s err=%v %v", msg, err, keysAndValues)
}
