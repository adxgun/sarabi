package backup

import (
	"context"
	"fmt"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"sarabi/internal/types"
	"sarabi/logger"
	"time"
)

type (
	// Schedule(ctx, job)
	// job -> interval(cron expression), task(ctx, settings)
	// jobs <- job
	// for each jobs
	//    listen on j.interval
	// 		on j.interval -> launch a gc to run task(ctx, settings)

	Scheduler interface {
		Schedule(j Job) error
		Start()
	}

	Task func(ctx context.Context, settings *types.BackupSettings) error

	Job struct {
		Name     string
		Interval string
		Task     Task
		Ctx      context.Context
		Setting  *types.BackupSettings
	}

	scheduler struct {
		jobs chan Job
	}
)

func NewScheduler() Scheduler {
	return &scheduler{jobs: make(chan Job, 5)}
}

func (s scheduler) Schedule(j Job) error {
	logger.Info("scheduling...",
		zap.String("ID", j.Name))
	s.jobs <- j
	return nil
}

func (s scheduler) Start() {
	go func() {
		for next := range s.jobs {
			go func() {
				if err := s.schedule(next); err != nil {
					logger.Error("job returned error",
						zap.Error(err))
				}
			}()
		}
	}()
}

func (s scheduler) schedule(next Job) error {
	interval, err := s.parseDuration(next.Interval)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := next.Task(next.Ctx, next.Setting); err != nil {
				return err
			}
		case <-next.Ctx.Done():
			break
		}
	}
}

func (s scheduler) parseDuration(cronExpression string) (time.Duration, error) {
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(cronExpression)
	if err != nil {
		return 0, fmt.Errorf("invalid cron expression: %w", err)
	}

	now := time.Now()
	next := schedule.Next(now)
	return time.Until(next), nil
}
