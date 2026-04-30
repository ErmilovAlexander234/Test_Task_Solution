package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

const generationHorizonDays = 365

type Service struct {
	repo Repository
	now  func() time.Time
}

func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

// -----------------сингл операции--------------------------------------------------

func (s *Service) Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error) {
	normalized, err := validateCreateInput(input)
	if err != nil {
		return nil, err
	}
	now := s.now()
	model := &taskdomain.Task{
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	return s.repo.Create(ctx, model)
}

func (s *Service) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}
	normalized, err := validateUpdateInput(input)
	if err != nil {
		return nil, err
	}
	model := &taskdomain.Task{
		ID:          id,
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		UpdatedAt:   s.now(),
	}
	return s.repo.Update(ctx, model)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}
	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context) ([]taskdomain.Task, error) {
	return s.repo.List(ctx)
}

// ----------------------таски с повтором-----------------------------------

func (s *Service) CreateRecurring(ctx context.Context, input CreateRecurringInput) ([]taskdomain.Task, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return nil, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	if input.Status == "" {
		input.Status = taskdomain.StatusNew
	}
	if !input.Status.Valid() {
		return nil, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}
	if err := input.Recurrence.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	today := truncateToDay(s.now())
	horizon := today.AddDate(0, 0, generationHorizonDays)
	dates := input.Recurrence.GenerateDates(today, horizon)

	now := s.now()
	tasks := make([]taskdomain.Task, 0, len(dates))
	for _, d := range dates {
		sched := d
		tasks = append(tasks, taskdomain.Task{
			Title:            title,
			Description:      strings.TrimSpace(input.Description),
			Status:           input.Status,
			RecurrenceRuleID: nil,
			ScheduledDate:    &sched,
			CreatedAt:        now,
			UpdatedAt:        now,
		})
	}

	_, createdTasks, err := s.repo.CreateRecurringTransaction(ctx, &input.Recurrence, tasks)
	if err != nil {
		return nil, err
	}
	return createdTasks, nil
}

func (s *Service) GetRecurrenceRule(ctx context.Context, id int64) (*taskdomain.RecurrenceRule, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}
	return s.repo.GetRecurrenceRule(ctx, id)
}

func (s *Service) ListRecurrenceRules(ctx context.Context) ([]taskdomain.RecurrenceRule, error) {
	return s.repo.ListRecurrenceRules(ctx)
}

// ------валида---------------------------------------

func validateCreateInput(input CreateInput) (CreateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	if input.Title == "" {
		return CreateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	if input.Status == "" {
		input.Status = taskdomain.StatusNew
	}
	if !input.Status.Valid() {
		return CreateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}
	return input, nil
}

func validateUpdateInput(input UpdateInput) (UpdateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)
	if input.Title == "" {
		return UpdateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}
	if !input.Status.Valid() {
		return UpdateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}
	return input, nil
}

func truncateToDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// ------------------удаляй-----------------------

func (s *Service) DeleteRuleTasks(ctx context.Context, ruleID int64) error {
	if ruleID <= 0 {
		return fmt.Errorf("%w: rule id must be positive", ErrInvalidInput)
	}
	if _, err := s.repo.GetRecurrenceRule(ctx, ruleID); err != nil {
		return err
	}
	return s.repo.DeleteTasksByRuleID(ctx, ruleID)
}

func (s *Service) DeleteRecurrenceRule(ctx context.Context, ruleID int64) error {
	if ruleID <= 0 {
		return fmt.Errorf("%w: rule id must be positive", ErrInvalidInput)
	}
	return s.repo.DeleteRecurrenceRuleWithTasks(ctx, ruleID)
}

// -------возврат тасок по правилу-----------
func (s *Service) ListTasksByRuleID(ctx context.Context, ruleID int64) ([]taskdomain.Task, error) {
	if ruleID <= 0 {
		return nil, fmt.Errorf("%w: rule id must be positive", ErrInvalidInput)
	}
	if _, err := s.repo.GetRecurrenceRule(ctx, ruleID); err != nil {
		return nil, err
	}
	return s.repo.ListTasksByRuleID(ctx, ruleID)
}
