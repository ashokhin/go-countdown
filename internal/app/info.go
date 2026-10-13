package app

// Version is the app's release version, derived from the nearest git tag at
// build time (see the Makefile's VERSION/-ldflags -X). It must stay a var,
// not a const — -X can only overwrite package-level string vars, not
// constants. "dev" is what it stays as for a plain `go build`/`go run`
// outside the Makefile, where nothing injects a real value.
var Version = "dev"

const (
	appName = "Countdown Timers"
	author  = "Andrei Shokhin"
	year    = "2026"
	license = "MIT"
	repoURL = "https://github.com/ashokhin/go-countdown"
)

// AppInfo is what the frontend's About dialog displays (see GetAppInfo).
type AppInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Author  string `json:"author"`
	Year    string `json:"year"`
	License string `json:"license"`
	RepoURL string `json:"repoUrl"`
}

// GetAppInfo returns the static details shown in the frontend's About
// dialog.
func (a *App) GetAppInfo() AppInfo {
	return AppInfo{
		Name:    appName,
		Version: Version,
		Author:  author,
		Year:    year,
		License: license,
		RepoURL: repoURL,
	}
}
