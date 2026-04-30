package task

import (
	"testing"
	"time"
)

func TestGenerateDates_Daily(t *testing.T) {
	start := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	rule := RecurrenceRule{
		Type:       RecurrenceDaily,
		EveryNDays: ptrInt(2),
		StartDate:  &start,
	}
	from := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 5, 10, 0, 0, 0, 0, time.UTC)

	dates := rule.GenerateDates(from, to)
	expected := []int{1, 3, 5, 7, 9}
	if len(dates) != len(expected) {
		t.Fatalf("expected %d dates, got %d", len(expected), len(dates))
	}
	for i, d := range dates {
		if d.Day() != expected[i] {
			t.Errorf("day mismatch: expected %d, got %d", expected[i], d.Day())
		}
	}
}

func TestGenerateDates_MonthlyDays_31(t *testing.T) {
	rule := RecurrenceRule{
		Type:        RecurrenceMonthlyDays,
		MonthlyDays: []int{31},
	}
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 3, 31, 0, 0, 0, 0, time.UTC)

	dates := rule.GenerateDates(from, to)
	if len(dates) != 2 {
		t.Fatalf("expected 2 dates, got %d", len(dates))
	}
	if dates[0].Month() != time.January || dates[0].Day() != 31 {
		t.Errorf("bad date: %v", dates[0])
	}
	if dates[1].Month() != time.March || dates[1].Day() != 31 {
		t.Errorf("bad date: %v", dates[1])
	}
}

func TestGenerateDates_MonthlyDays_30_Feb(t *testing.T) {
	rule := RecurrenceRule{
		Type:        RecurrenceMonthlyDays,
		MonthlyDays: []int{30},
	}
	from := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)

	dates := rule.GenerateDates(from, to)
	if len(dates) != 0 {
		t.Errorf("expected 0 dates in Feb, got %d", len(dates))
	}
}

func TestGenerateDates_SpecificDates(t *testing.T) {
	d1 := time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2025, 9, 15, 0, 0, 0, 0, time.UTC)
	rule := RecurrenceRule{
		Type:          RecurrenceSpecificDates,
		SpecificDates: []time.Time{d1, d2},
	}
	from := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 9, 30, 0, 0, 0, 0, time.UTC)

	dates := rule.GenerateDates(from, to)
	if len(dates) != 2 {
		t.Fatalf("expected 2, got %d", len(dates))
	}
}

func TestGenerateDates_EvenOdd(t *testing.T) {
	start := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	even := EvenDays
	rule := RecurrenceRule{
		Type:      RecurrenceEvenOddDays,
		EvenOdd:   &even,
		StartDate: &start,
	}
	from := time.Date(2025, 5, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 5, 10, 0, 0, 0, 0, time.UTC)

	dates := rule.GenerateDates(from, to)
	if len(dates) != 5 {
		t.Fatalf("expected 5 dates, got %d", len(dates))
	}
	for _, d := range dates {
		if d.Day()%2 != 0 {
			t.Errorf("odd day %d returned for even rule", d.Day())
		}
	}
}

func TestValidate_MonthlyDays_31_Valid(t *testing.T) {
	rule := RecurrenceRule{
		Type:        RecurrenceMonthlyDays,
		MonthlyDays: []int{31},
	}
	if err := rule.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_MonthlyDays_32_Invalid(t *testing.T) {
	rule := RecurrenceRule{
		Type:        RecurrenceMonthlyDays,
		MonthlyDays: []int{32},
	}
	if err := rule.Validate(); err == nil {
		t.Error("expected validation error for day 32")
	}
}

func TestValidate_MonthlyDays_0_Invalid(t *testing.T) {
	rule := RecurrenceRule{
		Type:        RecurrenceMonthlyDays,
		MonthlyDays: []int{0},
	}
	if err := rule.Validate(); err == nil {
		t.Error("expected validation error for day 0")
	}
}

func ptrInt(x int) *int { return &x }
