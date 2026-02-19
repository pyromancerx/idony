package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/pyromancer/idony/internal/db"
)

type Scheduler struct {
	cron    *cron.Cron
	agent   *Agent
	store   *db.Store
}

func NewScheduler(agent *Agent, store *db.Store) *Scheduler {
	return &Scheduler{
		cron:  cron.New(cron.WithSeconds()), // Support seconds if needed
		agent: agent,
		store: store,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.cron.Start()
	s.loadAndScheduleTasks(ctx)
}

func (s *Scheduler) loadAndScheduleTasks(ctx context.Context) {
	tasks, err := s.store.LoadTasks()
	if err != nil {
		log.Printf("Error loading tasks for scheduler: %v", err)
		return
	}

	for _, task := range tasks {
		s.schedule(ctx, task)
	}
}

func (s *Scheduler) schedule(ctx context.Context, task db.ScheduledTask) {
	if task.Type == "recurring" {
		_, err := s.cron.AddFunc(task.Schedule, func() {
			s.executeTask(ctx, task)
		})
		if err != nil {
			log.Printf("Error adding recurring task %d: %v", task.ID, err)
		}
	} else if task.Type == "one-shot" {
		runTime, err := time.Parse(time.RFC3339, task.Schedule)
		if err != nil {
			log.Printf("Error parsing one-shot time for task %d: %v", task.ID, err)
			return
		}

		delay := time.Until(runTime)
		if delay <= 0 {
			// If it was supposed to run in the past while bot was off, run it now
			go s.executeTask(ctx, task)
		} else {
			time.AfterFunc(delay, func() {
				s.executeTask(ctx, task)
			})
		}
	}
}

func (s *Scheduler) executeTask(ctx context.Context, task db.ScheduledTask) {
	fmt.Printf("\n[Scheduler]: Running scheduled task: %s\n", task.Prompt)
	
	// Run the prompt through the agent
	_, err := s.agent.Run(ctx, fmt.Sprintf("[Scheduled Task]: %s", task.Prompt))
	if err != nil {
		log.Printf("Error executing scheduled task %d: %v", task.ID, err)
		return
	}

	// Update last run time
	s.store.UpdateTaskLastRun(task.ID)

	// If one-shot, delete after execution
	if task.Type == "one-shot" {
		s.store.DeleteTask(task.ID)
	}
}

func (s *Scheduler) AddTask(ctx context.Context, taskType, schedule, prompt string) error {
	err := s.store.SaveTask(taskType, schedule, prompt)
	if err != nil {
		return err
	}
	
	// Reload/Reschedule is easiest for a small number of tasks
	// In a high-perf system we'd find the newly added ID
	s.loadAndScheduleTasks(ctx)
	return nil
}
