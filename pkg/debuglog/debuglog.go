// Package debuglog writes a running trace of external commands (aws,
// kubectl) and UI status messages to /tmp/<random>-ssm-me.log, since the
// TUI's single-line status bar truncates anything long — this is where the
// full command, output, and error actually end up.
package debuglog

import (
	"log"
	"os"
	"sync"
)

var (
	mu      sync.Mutex
	logger  *log.Logger
	logPath string
	enabled bool
)

// SetEnabled turns logging on or off. Enabling for the first time lazily
// opens /tmp/<random>-ssm-me.log and returns its path; disabling just stops
// further writes without closing the file. Safe to call repeatedly (e.g.
// from a settings toggle).
func SetEnabled(v bool) (string, error) {
	mu.Lock()
	defer mu.Unlock()
	if v && logger == nil {
		f, err := os.CreateTemp("", "*-ssm-me.log")
		if err != nil {
			return "", err
		}
		logger = log.New(f, "", log.LstdFlags|log.Lmicroseconds)
		logPath = f.Name()
	}
	enabled = v
	return logPath, nil
}

func Enabled() bool {
	mu.Lock()
	defer mu.Unlock()
	return enabled
}

func Path() string {
	mu.Lock()
	defer mu.Unlock()
	return logPath
}

func Printf(format string, args ...any) {
	mu.Lock()
	l, en := logger, enabled
	mu.Unlock()
	if !en || l == nil {
		return
	}
	l.Printf(format, args...)
}
