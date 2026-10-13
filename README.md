# go-countdown

[![CI](https://github.com/ashokhin/go-countdown/actions/workflows/ci.yml/badge.svg)](https://github.com/ashokhin/go-countdown/actions/workflows/ci.yml)

A small desktop countdown-timer app: add a name and a target date/time,
watch it count down live (`Xday(s) Xhour(s) Xminute(s) Xsecond(s)`), soonest
timer first.

Built with [Wails](https://wails.io/) — a Go backend driving the OS's native
WebView (WebView2 on Windows, WebKitGTK on Linux, WKWebView on macOS), with
a plain HTML/CSS/JS frontend.

## Features

- Add / edit / delete timers, sorted soonest-first
- Confirmation before delete, expired timers highlighted
- System notification when a timer expires
- Window size and font-size zoom remembered between launches
- Cross-platform build (Linux native, Windows cross-compiled with no extra
  toolchain), with a custom app icon embedded in both the running window and
  the built binary itself

## How it's wired

- **Go → JS**: `App`'s exported methods (`GetTimers`, `AddTimer`,
  `UpdateTimer`, `RemoveTimer`, `GetAppInfo` in
  [`internal/app/app.go`](internal/app/app.go)) are bound via
  `wails.Run(&options.App{Bind: ...})` and become callable from the frontend
  through generated bindings (`frontend/wailsjs/go/app/App.js`), each call
  resolving as a JS `Promise`.
- **Go → JS push**: `runtime.EventsEmit(ctx, "timers:updated", ...)` fires
  only when the timer list actually changes (add/edit/remove) — the frontend
  keeps its own `setInterval` to tick the displayed countdown locally from
  each timer's target timestamp, rather than polling the backend every
  second.
- **Notifications**: `runtime.SendNotification` is a native OS call Go makes
  directly; the frontend has no way to trigger it itself.

## Layout

```
go-countdown/
├── main.go               entry point: embeds frontend/dist + icon, wires window options
├── internal/app/           backend logic (not importable outside this module)
│   ├── app.go                 bound API (App struct) + notification ticker
│   ├── timer.go                  timer data model
│   ├── storage.go                   load/save timers (atomic file writes)
│   └── prefs.go                        window-size persistence
├── assets/                               icon source (svg/png)
├── build/                                  Wails-managed platform resources (icons, manifests)
└── frontend/
    ├── index.html                           page markup
    └── src/
        ├── main.js                            UI logic, Go bindings, local tick loop
        └── style.css                            styling (light/dark, rem-based zoom)
```

`main.go` stays a thin entrypoint — it owns the `go:embed` directives (which
can only reach files in their own directory tree, so `frontend/` and
`assets/` have to stay root-adjacent to it) and the `wails.Run` wiring; all
actual behavior lives in `internal/app`, bound into the frontend as
`backend := app.New()` → `Bind: []interface{}{backend}`.

## Build & run

Requires Go 1.25+, Node.js/npm, the [Wails CLI](https://wails.io/docs/gettingstarted/installation),
and on Linux the `webkit2gtk-4.1` dev package. Windows cross-builds need no
extra toolchain — Wails talks to WebView2 via pure-Go syscalls.

```sh
make           # -> bin/go-countdown and bin/go-countdown.exe (both platforms)
make build-linux   # Linux only
make build-windows # Windows only
make run            # build + launch
make test            # go test ./...
```

For hot-reload frontend development: `wails dev` (Vite dev server with fast
reload, plus a browser-devtools bridge at http://localhost:34115).

Timers are saved to `$XDG_CONFIG_HOME/wails-timer/events.json` (or the OS
equivalent via `os.UserConfigDir`).

## Versioning & releases

The version shown in the app's About dialog comes from the nearest git tag
(`git describe --tags --always --dirty`), injected at link time by the
Makefile — see `VERSION`/`LDFLAGS` there and `Version` in
[`internal/app/info.go`](internal/app/info.go). A build from a commit with
no tag yet, or outside a git checkout entirely, falls back to `dev`.

Pushing a `vX.Y.Z` tag triggers [`release.yml`](.github/workflows/release.yml),
which builds both platforms and publishes them as a GitHub Release:

```sh
git tag v1.1.0
git push origin v1.1.0
```

[`ci.yml`](.github/workflows/ci.yml) runs `go vet`/`go test` on every push
and pull request against `main`.

## License

[MIT](LICENSE)
