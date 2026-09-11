package logfiles

import (
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// maxRotatedLogs bounds how many rotated forum_*.log files accumulate in a
// log directory. Without this, every process restart (each container
// redeploy/crash/restart, under docker-compose.yml's bind-mounted
// ./logfiles host volume) leaves one more file behind forever — this repo's
// own logfiles/ directory had accumulated well over 100 of them, some going
// back months, before this existed. A crash-restart loop (the exact
// scenario these logs exist to help diagnose) would fill disk fastest of
// all, risking taking the app down via ENOSPC while trying to debug why it
// keeps crashing.
const maxRotatedLogs = 10

// CheckLog runs before the default slog logger is pointed at forum.log (see
// main.go), so these calls land on slog's built-in default (stderr) — the
// same destination the old log package used at this point in startup.
func CheckLog(dir string, filename string) {
	// Check to see if the log file forum.log exist. If it doesn't create it. If it does rename old file with date and create a new one.
	if _, err := os.Stat(dir + filename); os.IsNotExist(err) {
		// Create new file
		file, err := os.Create(dir + filename)
		if err != nil {
			slog.Error("log file could not be created", "error", err)
			os.Exit(1)
		}
		slog.Info("log file created")
		defer file.Close()
	} else {
		// Rename existing file with timestamp in filename
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		newFilename := "forum_" + timestamp + ".log"
		err := os.Rename(dir+filename, dir+newFilename)
		if err != nil {
			slog.Error("log file could not be renamed", "error", err)
			os.Exit(1)
		}
		// Create new file
		file, err := os.Create(dir + filename)
		if err != nil {
			slog.Error("log file could not be created after renaming", "error", err)
			os.Exit(1)
		}
		slog.Info("previous log file renamed and new log created")
		defer file.Close()
	}

	cleanupRotatedLogs(dir)
}

// cleanupRotatedLogs keeps at most the maxRotatedLogs most recently rotated
// forum_*.log files in dir, deleting the rest. Rotated filenames embed a
// sortable timestamp (forum_2006-01-02_15-04-05.log), so a plain
// lexicographic sort orders them chronologically without needing to stat
// each file's mtime. Best-effort: a failure to list or delete is logged,
// not fatal — an over-long log retention is a much smaller problem than
// crashing the server outright over disk cleanup.
func cleanupRotatedLogs(dir string) {
	matches, err := filepath.Glob(filepath.Join(dir, "forum_*.log"))
	if err != nil {
		slog.Error("failed to list rotated log files", "error", err)
		return
	}
	if len(matches) <= maxRotatedLogs {
		return
	}

	sort.Strings(matches)
	toDelete := matches[:len(matches)-maxRotatedLogs]
	for _, path := range toDelete {
		if err := os.Remove(path); err != nil {
			slog.Error("failed to delete old rotated log file", "path", path, "error", err)
			continue
		}
		slog.Info("deleted old rotated log file", "path", path)
	}
}
