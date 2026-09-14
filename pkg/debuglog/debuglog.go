// Package debuglog writes a running trace of external commands (aws,
// kubectl) and UI status messages to /tmp/<random>-ssm-me.log, since the
// TUI's single-line status bar truncates anything long — this is where the
// full command, output, and error actually end up.
package debuglog

import (
	"log"
	"os"
)

var logger *log.Logger

// Init opens /tmp/<random>-ssm-me.log and returns its path. Safe to call
// once at startup; if it fails, Printf becomes a silent no-op.
func Init() (string, error) {
	f, err := os.CreateTemp("", "*-ssm-me.log")
	if err != nil {
		return "", err
	}
	logger = log.New(f, "", log.LstdFlags|log.Lmicroseconds)
	return f.Name(), nil
}

func Printf(format string, args ...any) {
	if logger == nil {
		return
	}
	logger.Printf(format, args...)
}
