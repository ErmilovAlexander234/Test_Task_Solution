package handlers

import (
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type taskMutationDTO struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`
}

type taskDTO struct {
	ID               int64             `json:"id"`
	Title            string            `json:"title"`
	Description      string            `json:"description"`
	Status           taskdomain.Status `json:"status"`
	RecurrenceRuleID *int64            `json:"recurrence_rule_id,omitempty"`
	ScheduledDate    *string           `json:"scheduled_date,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

func newTaskDTO(task *taskdomain.Task) taskDTO {
	dto := taskDTO{
		ID:               task.ID,
		Title:            task.Title,
		Description:      task.Description,
		Status:           task.Status,
		RecurrenceRuleID: task.RecurrenceRuleID,
		CreatedAt:        task.CreatedAt,
		UpdatedAt:        task.UpdatedAt,
	}
	if task.ScheduledDate != nil {
		s := task.ScheduledDate.UTC().Format("2006-01-02")
		dto.ScheduledDate = &s
	}
	return dto
}

type createRecurringRequestDTO struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`
	Recurrence  recurrenceRuleDTO `json:"recurrence"`
}

type recurrenceRuleDTO struct {
	Type string `json:"type"`

	EveryNDays *int    `json:"every_n_days,omitempty"`
	StartDate  *string `json:"start_date,omitempty"`

	MonthlyDays []int `json:"monthly_days,omitempty"`

	SpecificDates []string `json:"specific_dates,omitempty"`

	EvenOdd *string `json:"even_odd,omitempty"`
}

type recurrenceRuleResponseDTO struct {
	ID            int64    `json:"id"`
	Type          string   `json:"type"`
	EveryNDays    *int     `json:"every_n_days,omitempty"`
	StartDate     *string  `json:"start_date,omitempty"`
	MonthlyDays   []int    `json:"monthly_days,omitempty"`
	SpecificDates []string `json:"specific_dates,omitempty"`
	EvenOdd       *string  `json:"even_odd,omitempty"`
}

func newRecurrenceRuleResponseDTO(r *taskdomain.RecurrenceRule) recurrenceRuleResponseDTO {
	dto := recurrenceRuleResponseDTO{
		ID:          r.ID,
		Type:        string(r.Type),
		EveryNDays:  r.EveryNDays,
		MonthlyDays: r.MonthlyDays,
	}
	if r.EvenOdd != nil {
		s := string(*r.EvenOdd)
		dto.EvenOdd = &s
	}
	if r.StartDate != nil {
		s := r.StartDate.UTC().Format("2006-01-02")
		dto.StartDate = &s
	}
	for _, d := range r.SpecificDates {
		dto.SpecificDates = append(dto.SpecificDates, d.UTC().Format("2006-01-02"))
	}
	return dto
}
