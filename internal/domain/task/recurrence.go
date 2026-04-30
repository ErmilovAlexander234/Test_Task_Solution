package task

import (
	"errors"
	"fmt"
	"time"
)

type RecurrenceType string

const (
	RecurrenceDaily         RecurrenceType = "daily"
	RecurrenceMonthlyDays   RecurrenceType = "monthly_days"
	RecurrenceSpecificDates RecurrenceType = "specific_dates"
	RecurrenceEvenOddDays   RecurrenceType = "even_odd_days"
)

type EvenOdd string

const (
	EvenDays EvenOdd = "even"
	OddDays  EvenOdd = "odd"
)

type RecurrenceRule struct {
	ID            int64          `json:"id"`
	Type          RecurrenceType `json:"type"`
	EveryNDays    *int           `json:"every_n_days,omitempty"`
	MonthlyDays   []int          `json:"monthly_days,omitempty"`
	SpecificDates []time.Time    `json:"specific_dates,omitempty"`
	EvenOdd       *EvenOdd       `json:"even_odd,omitempty"`
	StartDate     *time.Time     `json:"start_date,omitempty"`
}

var ErrInvalidRecurrence = errors.New("invalid recurrence rule")

func (r RecurrenceRule) Validate() error {
	switch r.Type {
	case RecurrenceDaily:
		if r.EveryNDays == nil || *r.EveryNDays < 1 {
			return fmt.Errorf("%w: every_n_days must be >= 1 for type 'daily'", ErrInvalidRecurrence)
		}
		if r.StartDate == nil {
			return fmt.Errorf("%w: start_date is required for type 'daily'", ErrInvalidRecurrence)
		}

	case RecurrenceMonthlyDays:
		if len(r.MonthlyDays) == 0 {
			return fmt.Errorf("%w: monthly_days must not be empty for type 'monthly_days'", ErrInvalidRecurrence)
		}
		seen := make(map[int]struct{})
		for _, d := range r.MonthlyDays {
			if d < 1 || d > 31 {
				return fmt.Errorf("%w: each day in monthly_days must be between 1 and 31, got %d", ErrInvalidRecurrence, d)
			}
			if _, dup := seen[d]; dup {
				return fmt.Errorf("%w: duplicate day %d in monthly_days", ErrInvalidRecurrence, d)
			}
			seen[d] = struct{}{}
		}

	case RecurrenceSpecificDates:
		if len(r.SpecificDates) == 0 {
			return fmt.Errorf("%w: specific_dates must not be empty for type 'specific_dates'", ErrInvalidRecurrence)
		}
		seen := make(map[string]struct{})
		for _, d := range r.SpecificDates {
			key := d.UTC().Format("2006-01-02")
			if _, dup := seen[key]; dup {
				return fmt.Errorf("%w: duplicate date %s in specific_dates", ErrInvalidRecurrence, key)
			}
			seen[key] = struct{}{}
		}

	case RecurrenceEvenOddDays:
		if r.EvenOdd == nil {
			return fmt.Errorf("%w: even_odd is required for type 'even_odd_days'", ErrInvalidRecurrence)
		}
		if *r.EvenOdd != EvenDays && *r.EvenOdd != OddDays {
			return fmt.Errorf("%w: even_odd must be 'even' or 'odd'", ErrInvalidRecurrence)
		}
		if r.StartDate == nil {
			return fmt.Errorf("%w: start_date is required for type 'even_odd_days'", ErrInvalidRecurrence)
		}

	default:
		return fmt.Errorf("%w: unknown type %q", ErrInvalidRecurrence, r.Type)
	}
	return nil
}

func (r RecurrenceRule) GenerateDates(from, to time.Time) []time.Time {
	from = truncateToDay(from)
	to = truncateToDay(to)

	var out []time.Time

	switch r.Type {
	case RecurrenceDaily:
		start := truncateToDay(*r.StartDate)
		if start.After(to) {
			return nil
		}
		step := *r.EveryNDays
		cur := start
		if cur.Before(from) {
			diff := int(from.Sub(cur).Hours() / 24)
			skips := diff / step
			cur = cur.AddDate(0, 0, skips*step)
			if cur.Before(from) {
				cur = cur.AddDate(0, 0, step)
			}
		}
		for !cur.After(to) {
			out = append(out, cur)
			cur = cur.AddDate(0, 0, step)
		}

	case RecurrenceMonthlyDays:
		cur := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
		last := time.Date(to.Year(), to.Month(), 1, 0, 0, 0, 0, time.UTC)
		for !cur.After(last) {
			daysInMonth := time.Date(cur.Year(), cur.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
			for _, day := range r.MonthlyDays {
				if day > daysInMonth {
					continue
				}
				d := time.Date(cur.Year(), cur.Month(), day, 0, 0, 0, 0, time.UTC)
				if !d.Before(from) && !d.After(to) {
					out = append(out, d)
				}
			}
			cur = cur.AddDate(0, 1, 0)
		}

	case RecurrenceSpecificDates:
		for _, sd := range r.SpecificDates {
			d := truncateToDay(sd)
			if !d.Before(from) && !d.After(to) {
				out = append(out, d)
			}
		}

	case RecurrenceEvenOddDays:
		start := truncateToDay(*r.StartDate)
		cur := start
		if cur.Before(from) {
			cur = from
		}
		for !cur.After(to) {
			dayNum := cur.Day()
			isEven := dayNum%2 == 0
			if (*r.EvenOdd == EvenDays && isEven) || (*r.EvenOdd == OddDays && !isEven) {
				out = append(out, cur)
			}
			cur = cur.AddDate(0, 0, 1)
		}
	}
	return out
}

func truncateToDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
