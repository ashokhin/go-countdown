package app

import (
	"sort"
	"strconv"
	"sync/atomic"
	"time"
)

// TimerEvent is a named point in time to count down to, as persisted to
// disk. Unlike the Fyne version's index-based CountdownEvent, this carries
// a stable ID: the Go and JS sides talk across an async boundary (JS calls
// return over IPC, and the periodic tick pushes events independently), so
// an index into the slice can't be trusted to still mean the same timer by
// the time a call resolves — a stable ID sidesteps that entirely.
type TimerEvent struct {
	ID     string    `json:"id"`
	Name   string    `json:"name"`
	Target time.Time `json:"target"`

	// notified tracks whether the expiry notification has already been sent
	// for this event, so it fires once rather than every tick. Unexported
	// (and so never persisted): a timer that already expired before the app
	// was last closed doesn't need to notify again just because it was
	// reloaded from disk.
	notified bool
}

// Expired reports whether the event's target time has passed.
func (e TimerEvent) Expired(now time.Time) bool {
	return !e.Target.After(now)
}

// TimerView is what the frontend actually renders: the countdown already
// broken into display fields, computed fresh at snapshot time rather than
// redone independently in JS, so there's exactly one place that defines
// "how a remaining duration turns into Days/Hours/Minutes/Seconds".
type TimerView struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Target  int64  `json:"target"` // unix milliseconds, for the edit form to prefill from
	Days    int64  `json:"days"`
	Hours   int64  `json:"hours"`
	Minutes int64  `json:"minutes"`
	Seconds int64  `json:"seconds"`
	Expired bool   `json:"expired"`
}

var nextTimerID atomic.Uint64

// newTimerID returns a value unique within this process's lifetime — good
// enough for a single-user local app where IDs never need to survive
// comparison across separate runs.
func newTimerID() string {
	n := nextTimerID.Add(1)
	return time.Now().UTC().Format("20060102150405") + "-" + strconv.FormatUint(n, 10)
}

func sortTimers(events []TimerEvent) {
	sort.Slice(events, func(i, j int) bool {
		return events[i].Target.Before(events[j].Target)
	})
}

// snapshotTimers computes the current display view for every event. Events
// are assumed already sorted by Target (see sortTimers) — this only turns
// each one's remaining duration into display fields, it doesn't reorder.
func snapshotTimers(events []TimerEvent, now time.Time) []TimerView {
	views := make([]TimerView, len(events))
	for i, ev := range events {
		remaining := ev.Target.Sub(now)
		expired := remaining <= 0
		if expired {
			remaining = 0
		}
		totalSeconds := int64(remaining.Seconds())
		views[i] = TimerView{
			ID:      ev.ID,
			Name:    ev.Name,
			Target:  ev.Target.UnixMilli(),
			Days:    totalSeconds / 86400,
			Hours:   (totalSeconds % 86400) / 3600,
			Minutes: (totalSeconds % 3600) / 60,
			Seconds: totalSeconds % 60,
			Expired: expired,
		}
	}
	return views
}
