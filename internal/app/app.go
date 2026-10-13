package app

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// timersUpdatedEvent is the Wails runtime event name the frontend listens
// on (see frontend/src/main.js) for fresh timer data, emitted after every
// successful Add/Update/Remove. It is NOT pushed once a second — each
// TimerView carries a plain unix-millis Target, so the frontend recomputes
// the live days/hours/minutes/seconds itself with its own setInterval
// instead of waiting on the backend to tick. That keeps a once-a-second
// JSON round-trip through the WebView bridge off the hot path; the backend
// only needs to talk to the frontend when the underlying list actually
// changes.
const timersUpdatedEvent = "timers:updated"

// App is the Wails-bound backend. Every exported method becomes callable
// from the frontend as window.go.app.App.<Method> (see
// frontend/wailsjs/go/app/App.js, generated at build time).
//
// mu guards events: Wails invokes bound methods on whatever goroutine the
// webview dispatches them from, which runs concurrently with the ticker
// goroutine below.
type App struct {
	ctx    context.Context
	mu     sync.Mutex
	events []TimerEvent
}

// New constructs a backend ready to be passed to wails.Run's OnStartup/Bind.
func New() *App {
	return &App{}
}

// Startup is called once by Wails when the window is ready (wire it up as
// OnStartup). Events are sorted by Target, a fixed instant, so their
// relative order never changes as real time passes — only Add/Update/Remove
// can change it, and each already re-sorts after mutating.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	events, err := loadTimers()
	if err != nil {
		runtime.LogErrorf(ctx, "could not load saved timers: %v", err)
		events = nil
	}
	sortTimers(events)

	a.mu.Lock()
	a.events = events
	a.mu.Unlock()

	if err := runtime.InitializeNotifications(ctx); err != nil {
		// Not fatal: the app is just as usable without OS notifications,
		// e.g. on a Linux system with no notification daemon running.
		runtime.LogWarningf(ctx, "notifications unavailable: %v", err)
	}

	go a.tick()
}

// tick runs once a second purely to catch newly-expired timers and fire OS
// notifications for them — something only Go can do (runtime.SendNotification
// is a native OS call, unreachable from JS). It does NOT push a snapshot to
// the frontend; the frontend already has each timer's target time and ticks
// its own countdown display independently (see timersUpdatedEvent above).
func (a *App) tick() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for range ticker.C {
		a.notifyExpirations()
	}
}

// notifyExpirations sends an OS notification for every event that has
// expired but hasn't been notified about yet, then marks it as notified so
// it only fires once (not on every tick for as long as the timer stays in
// the list).
func (a *App) notifyExpirations() {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	for i := range a.events {
		ev := &a.events[i]
		if !ev.Expired(now) || ev.notified {
			continue
		}
		ev.notified = true

		name := ev.Name
		if name == "" {
			name = "Timer"
		}
		if err := runtime.SendNotification(a.ctx, runtime.NotificationOptions{
			Title: name,
			Body:  "has expired",
		}); err != nil {
			runtime.LogWarningf(a.ctx, "sending notification: %v", err)
		}
	}
}

func (a *App) emitSnapshot() {
	a.mu.Lock()
	views := snapshotTimers(a.events, time.Now())
	a.mu.Unlock()
	runtime.EventsEmit(a.ctx, timersUpdatedEvent, views)
}

// GetTimers returns the current countdowns. The frontend calls this once on
// load to get its first paint before the first tick arrives.
func (a *App) GetTimers() []TimerView {
	a.mu.Lock()
	defer a.mu.Unlock()
	return snapshotTimers(a.events, time.Now())
}

// AddTimer appends a new timer with the given name and target time (unix
// milliseconds, matching JS's Date.getTime()).
func (a *App) AddTimer(name string, targetUnixMillis int64) error {
	ev := TimerEvent{ID: newTimerID(), Name: name, Target: time.UnixMilli(targetUnixMillis)}

	a.mu.Lock()
	a.events = append(a.events, ev)
	sortTimers(a.events)
	err := saveTimers(a.events)
	a.mu.Unlock()
	if err != nil {
		return fmt.Errorf("saving timers: %w", err)
	}

	a.emitSnapshot()
	return nil
}

// UpdateTimer overwrites the name/target of the timer with the given ID —
// used for editing rather than a remove-then-add, so the same timer keeps
// its identity (and its position doesn't visually reset) across an edit.
func (a *App) UpdateTimer(id, name string, targetUnixMillis int64) error {
	a.mu.Lock()
	found := false
	for i := range a.events {
		if a.events[i].ID == id {
			a.events[i].Name = name
			a.events[i].Target = time.UnixMilli(targetUnixMillis)
			// The new target may no longer be in the past, so this should
			// get a chance to notify again rather than staying silenced
			// from before the edit.
			a.events[i].notified = false
			found = true
			break
		}
	}
	if !found {
		a.mu.Unlock()
		return fmt.Errorf("no timer with id %q", id)
	}
	sortTimers(a.events)
	err := saveTimers(a.events)
	a.mu.Unlock()
	if err != nil {
		return fmt.Errorf("saving timers: %w", err)
	}

	a.emitSnapshot()
	return nil
}

// RemoveTimer deletes the timer with the given ID.
func (a *App) RemoveTimer(id string) error {
	a.mu.Lock()
	kept := a.events[:0:0]
	for _, ev := range a.events {
		if ev.ID != id {
			kept = append(kept, ev)
		}
	}
	a.events = kept
	err := saveTimers(a.events)
	a.mu.Unlock()
	if err != nil {
		return fmt.Errorf("saving timers: %w", err)
	}

	a.emitSnapshot()
	return nil
}
