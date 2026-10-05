package scheduler

import (
	"testing"
	"time"

	"auto-poster/internal/publisher"
)

func TestPublishAllowedDailyStart(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Dhaka")
	mk := func(h, m int) time.Time {
		return time.Date(2026, 10, 5, h, m, 0, 0, loc)
	}

	settings := &publisher.AutoPosterSettings{
		Enabled:             true,
		DailyPublishTime:    "09:30",
		ScheduleIntervalMin: 60,
		Timezone:            "Asia/Dhaka",
	}

	cases := []struct {
		name string
		now  time.Time
		want bool
	}{
		{"before daily start", mk(9, 15), false},
		{"at daily start", mk(9, 30), true},
		{"well after daily start", mk(9, 50), true},
		{"late run still allowed", mk(14, 47), true},
	}

	for _, tc := range cases {
		if got := PublishAllowed(settings, tc.now, nil); got != tc.want {
			t.Errorf("%s: PublishAllowed(%v) = %v, want %v", tc.name, tc.now, got, tc.want)
		}
	}
}

func TestPublishAllowedInterval(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Dhaka")
	at := func(h, m int) time.Time {
		return time.Date(2026, 10, 5, h, m, 0, 0, loc)
	}

	settings := &publisher.AutoPosterSettings{
		Enabled:             true,
		DailyPublishTime:    "09:00",
		ScheduleIntervalMin: 60,
		Timezone:            "Asia/Dhaka",
	}

	recent := at(12, 30)
	old := at(11, 0)

	if PublishAllowed(settings, at(12, 45), &recent) {
		t.Error("should skip when interval has not elapsed since last publish")
	}
	if !PublishAllowed(settings, at(13, 31), &recent) {
		t.Error("should allow once the interval has elapsed")
	}
	if !PublishAllowed(settings, at(12, 45), &old) {
		t.Error("should allow when last publish was longer ago than the interval")
	}
}

func TestPublishAllowedDefaults(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Dhaka")
	now := time.Date(2026, 10, 5, 9, 15, 0, 0, loc)

	empty := &publisher.AutoPosterSettings{Enabled: true}
	if !PublishAllowed(empty, now, nil) {
		t.Error("empty daily time should always allow")
	}
	if !PublishAllowed(nil, now, nil) {
		t.Error("nil settings should always allow")
	}
}
