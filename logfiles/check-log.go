package logfiles

import (
	"log/slog"
	"os"
	"time"
)

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
}
