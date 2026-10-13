package app

import (
	"testing"
	"time"
)

func TestTimerEventExpired(t *testing.T) {
	now := time.Now()
	past := TimerEvent{Target: now.Add(-time.Minute)}
	future := TimerEvent{Target: now.Add(time.Minute)}

	if !past.Expired(now) {
		t.Error("event with a target in the past should be expired")
	}
	if future.Expired(now) {
		t.Error("event with a target in the future should not be expired")
	}
}

func TestSnapshotTimers(t *testing.T) {
	now := time.Now()
	events := []TimerEvent{
		{ID: "a", Name: "Launch", Target: now.Add(25*time.Hour + 3*time.Minute + 4*time.Second)},
		{ID: "b", Name: "Past", Target: now.Add(-time.Hour)},
	}

	views := snapshotTimers(events, now)
	if len(views) != 2 {
		t.Fatalf("got %d views, want 2", len(views))
	}

	future := views[0]
	if future.ID != "a" || future.Name != "Launch" {
		t.Errorf("views[0] = %+v, want ID=a Name=Launch", future)
	}
	if future.Days != 1 || future.Hours != 1 || future.Minutes != 3 || future.Seconds != 4 {
		t.Errorf("views[0] duration parts = %d/%d/%d/%d, want 1/1/3/4",
			future.Days, future.Hours, future.Minutes, future.Seconds)
	}
	if future.Expired {
		t.Error("views[0] should not be expired")
	}

	past := views[1]
	if !past.Expired {
		t.Error("views[1] (target in the past) should be expired")
	}
	if past.Days != 0 || past.Hours != 0 || past.Minutes != 0 || past.Seconds != 0 {
		t.Errorf("expired event's duration parts should clamp to zero, got %d/%d/%d/%d",
			past.Days, past.Hours, past.Minutes, past.Seconds)
	}
}

func TestSortTimersOrdersByTargetAscending(t *testing.T) {
	now := time.Now()
	events := []TimerEvent{
		{ID: "later", Target: now.Add(2 * time.Hour)},
		{ID: "sooner", Target: now.Add(time.Hour)},
	}

	sortTimers(events)

	if events[0].ID != "sooner" || events[1].ID != "later" {
		t.Errorf("order after sort = [%s, %s], want [sooner, later]", events[0].ID, events[1].ID)
	}
}

func TestNewTimerIDIsUnique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := newTimerID()
		if seen[id] {
			t.Fatalf("newTimerID produced a duplicate: %q", id)
		}
		seen[id] = true
	}
}
