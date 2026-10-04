package scheduler

import (
	"testing"
	"time"

	"auto-poster/internal/publisher"
)

func TestPublishAllowed(t *testing.T) {
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
		{"at slot start", mk(9, 30), true},
		{"within 15m after slot", mk(9, 40), true},
		{"after 15m window", mk(9, 50), false},
		{"next slot", mk(10, 30), true},
		{"before next slot", mk(10, 5), false},
	}

	for _, tc := range cases {
		if got := PublishAllowed(settings, tc.now); got != tc.want {
			t.Errorf("%s: PublishAllowed(%v) = %v, want %v", tc.name, tc.now, got, tc.want)
		}
	}

	empty := &publisher.AutoPosterSettings{Enabled: true}
	if !PublishAllowed(empty, mk(9, 15)) {
		t.Error("empty daily time should always allow")
	}
}