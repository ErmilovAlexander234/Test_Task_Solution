package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

//------------------------Крудик на таску (СИНГЛОВУЮ!!!)---------------------

func (r *Repository) Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		INSERT INTO tasks (title, description, status, recurrence_rule_id, scheduled_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, title, description, status, recurrence_rule_id, scheduled_date, created_at, updated_at
	`
	row := r.pool.QueryRow(ctx, query,
		task.Title, task.Description, task.Status,
		task.RecurrenceRuleID, task.ScheduledDate,
		task.CreatedAt, task.UpdatedAt,
	)
	return scanTask(row)
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, recurrence_rule_id, scheduled_date, created_at, updated_at
		FROM tasks WHERE id = $1
	`
	row := r.pool.QueryRow(ctx, query, id)
	task, err := scanTask(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, taskdomain.ErrNotFound
	}
	return task, err
}

func (r *Repository) Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		UPDATE tasks
		SET title = $1, description = $2, status = $3, updated_at = $4
		WHERE id = $5
		RETURNING id, title, description, status, recurrence_rule_id, scheduled_date, created_at, updated_at
	`
	row := r.pool.QueryRow(ctx, query, task.Title, task.Description, task.Status, task.UpdatedAt, task.ID)
	updated, err := scanTask(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, taskdomain.ErrNotFound
	}
	return updated, err
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM tasks WHERE id = $1`
	cmd, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}
	return nil
}

func (r *Repository) List(ctx context.Context) ([]taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, recurrence_rule_id, scheduled_date, created_at, updated_at
		FROM tasks
		ORDER BY COALESCE(scheduled_date, created_at) ASC, id ASC
	`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []taskdomain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, *t)
	}
	return tasks, rows.Err()
}

// ------------Вставка с копированием------------------------------

func (r *Repository) CreateMany(ctx context.Context, tasks []taskdomain.Task) ([]taskdomain.Task, error) {
	if len(tasks) == 0 {
		return []taskdomain.Task{}, nil
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	rows := make([][]any, len(tasks))
	for i, t := range tasks {
		rows[i] = []any{
			t.Title, t.Description, t.Status,
			t.RecurrenceRuleID, t.ScheduledDate,
			t.CreatedAt, t.UpdatedAt,
		}
	}
	_, err = tx.CopyFrom(
		ctx,
		pgx.Identifier{"tasks"},
		[]string{"title", "description", "status", "recurrence_rule_id", "scheduled_date", "created_at", "updated_at"},
		pgx.CopyFromRows(rows),
	)
	if err != nil {
		return nil, fmt.Errorf("copy from: %w", err)
	}

	now := tasks[0].CreatedAt
	query := `SELECT id, title, description, status, recurrence_rule_id, scheduled_date, created_at, updated_at FROM tasks WHERE created_at = $1 ORDER BY id ASC`
	rows2, err := tx.Query(ctx, query, now)
	if err != nil {
		return nil, fmt.Errorf("fetch created tasks: %w", err)
	}
	defer rows2.Close()

	var created []taskdomain.Task
	for rows2.Next() {
		t, err := scanTask(rows2)
		if err != nil {
			return nil, err
		}
		created = append(created, *t)
	}
	if err := rows2.Err(); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return created, nil
}

//--------------------Правило повторения------------------------------

func (r *Repository) CreateRecurrenceRule(ctx context.Context, rule *taskdomain.RecurrenceRule) (*taskdomain.RecurrenceRule, error) {
	payload, err := marshalRule(rule)
	if err != nil {
		return nil, err
	}
	const q = `INSERT INTO recurrence_rules (type, payload) VALUES ($1, $2) RETURNING id`
	var id int64
	if err := r.pool.QueryRow(ctx, q, string(rule.Type), payload).Scan(&id); err != nil {
		return nil, fmt.Errorf("insert recurrence_rule: %w", err)
	}
	rule.ID = id
	return rule, nil
}

func (r *Repository) GetRecurrenceRule(ctx context.Context, id int64) (*taskdomain.RecurrenceRule, error) {
	const q = `SELECT id, type, payload FROM recurrence_rules WHERE id = $1`
	var ruleID int64
	var ruleType string
	var payload []byte
	err := r.pool.QueryRow(ctx, q, id).Scan(&ruleID, &ruleType, &payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, taskdomain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	rule, err := unmarshalRule(taskdomain.RecurrenceType(ruleType), payload)
	if err != nil {
		return nil, err
	}
	rule.ID = ruleID
	return rule, nil
}

func (r *Repository) ListRecurrenceRules(ctx context.Context) ([]taskdomain.RecurrenceRule, error) {
	const q = `SELECT id, type, payload FROM recurrence_rules ORDER BY id`
	rows, err := r.pool.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var rules []taskdomain.RecurrenceRule
	for rows.Next() {
		var id int64
		var ruleType string
		var payload []byte
		if err := rows.Scan(&id, &ruleType, &payload); err != nil {
			return nil, err
		}
		rule, err := unmarshalRule(taskdomain.RecurrenceType(ruleType), payload)
		if err != nil {
			return nil, err
		}
		rule.ID = id
		rules = append(rules, *rule)
	}
	return rules, rows.Err()
}

// ------------------CreateRecurringTransaction + атомарность тасок и правил
func (r *Repository) CreateRecurringTransaction(ctx context.Context, rule *taskdomain.RecurrenceRule, tasks []taskdomain.Task) (*taskdomain.RecurrenceRule, []taskdomain.Task, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("begin: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	payload, err := marshalRule(rule)
	if err != nil {
		return nil, nil, err
	}
	const insertRule = `INSERT INTO recurrence_rules (type, payload) VALUES ($1, $2) RETURNING id`
	var ruleID int64
	if err := tx.QueryRow(ctx, insertRule, string(rule.Type), payload).Scan(&ruleID); err != nil {
		return nil, nil, fmt.Errorf("insert rule: %w", err)
	}
	rule.ID = ruleID

	if len(tasks) > 0 {
		rows := make([][]any, len(tasks))
		for i, t := range tasks {
			rows[i] = []any{t.Title, t.Description, t.Status, &ruleID, t.ScheduledDate, t.CreatedAt, t.UpdatedAt}
		}
		_, err = tx.CopyFrom(ctx, pgx.Identifier{"tasks"},
			[]string{"title", "description", "status", "recurrence_rule_id", "scheduled_date", "created_at", "updated_at"},
			pgx.CopyFromRows(rows))
		if err != nil {
			return nil, nil, fmt.Errorf("copy tasks: %w", err)
		}
	}

	var createdTasks []taskdomain.Task
	if len(tasks) > 0 {
		selectQ := `SELECT id, title, description, status, recurrence_rule_id, scheduled_date, created_at, updated_at FROM tasks WHERE recurrence_rule_id = $1 ORDER BY scheduled_date`
		rows2, err := tx.Query(ctx, selectQ, ruleID)
		if err != nil {
			return nil, nil, fmt.Errorf("select tasks: %w", err)
		}
		defer rows2.Close()
		for rows2.Next() {
			t, err := scanTask(rows2)
			if err != nil {
				return nil, nil, err
			}
			createdTasks = append(createdTasks, *t)
		}
		if err := rows2.Err(); err != nil {
			return nil, nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, nil, fmt.Errorf("commit: %w", err)
	}
	return rule, createdTasks, nil
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTask(scanner taskScanner) (*taskdomain.Task, error) {
	var t taskdomain.Task
	var status string
	var scheduledDate *time.Time
	err := scanner.Scan(
		&t.ID, &t.Title, &t.Description, &status,
		&t.RecurrenceRuleID, &scheduledDate,
		&t.CreatedAt, &t.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	t.Status = taskdomain.Status(status)
	t.ScheduledDate = scheduledDate
	return &t, nil
}

type rulePayload struct {
	EveryNDays    *int                `json:"every_n_days,omitempty"`
	MonthlyDays   []int               `json:"monthly_days,omitempty"`
	SpecificDates []string            `json:"specific_dates,omitempty"`
	EvenOdd       *taskdomain.EvenOdd `json:"even_odd,omitempty"`
	StartDate     *string             `json:"start_date,omitempty"`
}

func marshalRule(rule *taskdomain.RecurrenceRule) ([]byte, error) {
	p := rulePayload{
		EveryNDays:  rule.EveryNDays,
		MonthlyDays: rule.MonthlyDays,
		EvenOdd:     rule.EvenOdd,
	}
	for _, d := range rule.SpecificDates {
		p.SpecificDates = append(p.SpecificDates, d.UTC().Format("2006-01-02"))
	}
	if rule.StartDate != nil {
		s := rule.StartDate.UTC().Format("2006-01-02")
		p.StartDate = &s
	}
	return json.Marshal(p)
}

func unmarshalRule(t taskdomain.RecurrenceType, data []byte) (*taskdomain.RecurrenceRule, error) {
	var p rulePayload
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	rule := &taskdomain.RecurrenceRule{
		Type:        t,
		EveryNDays:  p.EveryNDays,
		MonthlyDays: p.MonthlyDays,
		EvenOdd:     p.EvenOdd,
	}
	for _, s := range p.SpecificDates {
		d, err := time.Parse("2006-01-02", s)
		if err != nil {
			return nil, err
		}
		rule.SpecificDates = append(rule.SpecificDates, d.UTC())
	}
	if p.StartDate != nil {
		d, err := time.Parse("2006-01-02", *p.StartDate)
		if err != nil {
			return nil, err
		}
		d = d.UTC()
		rule.StartDate = &d
	}
	return rule, nil
}

func (r *Repository) DeleteTasksByRuleID(ctx context.Context, ruleID int64) error {
	const query = `DELETE FROM tasks WHERE recurrence_rule_id = $1`
	_, err := r.pool.Exec(ctx, query, ruleID)
	return err
}

func (r *Repository) DeleteRecurrenceRule(ctx context.Context, ruleID int64) error {
	const query = `DELETE FROM recurrence_rules WHERE id = $1`
	cmd, err := r.pool.Exec(ctx, query, ruleID)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteRecurrenceRuleWithTasks(ctx context.Context, ruleID int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// Удалить задачи
	const delTasks = `DELETE FROM tasks WHERE recurrence_rule_id = $1`
	if _, err := tx.Exec(ctx, delTasks, ruleID); err != nil {
		return fmt.Errorf("delete tasks: %w", err)
	}

	// Удалить правило
	const delRule = `DELETE FROM recurrence_rules WHERE id = $1`
	cmd, err := tx.Exec(ctx, delRule, ruleID)
	if err != nil {
		return fmt.Errorf("delete rule: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}

// ----возврат тасок по правилу---------
func (r *Repository) ListTasksByRuleID(ctx context.Context, ruleID int64) ([]taskdomain.Task, error) {
	const query = `
        SELECT id, title, description, status, recurrence_rule_id, scheduled_date, created_at, updated_at
        FROM tasks
        WHERE recurrence_rule_id = $1
        ORDER BY scheduled_date ASC, id ASC
    `
	rows, err := r.pool.Query(ctx, query, ruleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []taskdomain.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, *t)
	}
	return tasks, rows.Err()
}
