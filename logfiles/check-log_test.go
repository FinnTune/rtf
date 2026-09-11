package logfiles

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// touchLogFile creates an empty file named name inside dir, failing the
// test on error. Rotated log filenames embed a sortable timestamp
// (forum_2006-01-02_15-04-05.log), so tests can control ordering directly
// via the name rather than needing real, distinct creation times.
func touchLogFile(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("log line\n"), 0644); err != nil {
		t.Fatalf("failed to create %s: %v", name, err)
	}
}

func remainingRotatedLogs(t *testing.T, dir string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "forum_*.log"))
	if err != nil {
		t.Fatalf("failed to glob rotated logs: %v", err)
	}
	return matches
}

func TestCleanupRotatedLogs_NoopWhenUnderLimit(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < maxRotatedLogs-1; i++ {
		touchLogFile(t, dir, fmt.Sprintf("forum_2026-01-01_00-00-%02d.log", i))
	}

	cleanupRotatedLogs(dir)

	remaining := remainingRotatedLogs(t, dir)
	if len(remaining) != maxRotatedLogs-1 {
		t.Fatalf("expected all %d files to survive (under the limit), got %d remaining", maxRotatedLogs-1, len(remaining))
	}
}

func TestCleanupRotatedLogs_KeepsOnlyMostRecentN(t *testing.T) {
	dir := t.TempDir()
	const total = maxRotatedLogs + 5
	var names []string
	for i := 0; i < total; i++ {
		// Zero-padded so lexicographic order matches creation order exactly,
		// the same way the real timestamp format
		// (2006-01-02_15-04-05) sorts chronologically.
		name := fmt.Sprintf("forum_2026-01-01_00-00-%02d.log", i)
		names = append(names, name)
		touchLogFile(t, dir, name)
	}

	cleanupRotatedLogs(dir)

	remaining := remainingRotatedLogs(t, dir)
	if len(remaining) != maxRotatedLogs {
		t.Fatalf("expected exactly %d files to remain, got %d: %v", maxRotatedLogs, len(remaining), remaining)
	}

	// The survivors must be the newest maxRotatedLogs names (the last ones
	// created), not an arbitrary subset.
	wantSurvivors := make(map[string]bool, maxRotatedLogs)
	for _, name := range names[total-maxRotatedLogs:] {
		wantSurvivors[filepath.Join(dir, name)] = true
	}
	for _, path := range remaining {
		if !wantSurvivors[path] {
			t.Fatalf("expected %s to have been deleted (older than the %d most recent), but it survived", path, maxRotatedLogs)
		}
	}

	// And the oldest ones must be genuinely gone, not just excluded from
	// the glob for some other reason.
	for _, name := range names[:total-maxRotatedLogs] {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("expected %s to have been deleted, stat error: %v", name, err)
		}
	}
}

// TestCheckLog_CleansUpOldRotatedLogsOnRotation covers CheckLog's actual
// production entry point: a directory that already has more than
// maxRotatedLogs old rotated files (e.g. from many prior restarts, before
// this retention existed) must settle back down to maxRotatedLogs once
// CheckLog runs and performs one more rotation — not accumulate forever.
func TestCheckLog_CleansUpOldRotatedLogsOnRotation(t *testing.T) {
	dir := t.TempDir() + string(filepath.Separator)

	const preExisting = maxRotatedLogs + 20
	for i := 0; i < preExisting; i++ {
		touchLogFile(t, dir, fmt.Sprintf("forum_2026-01-01_00-01-%02d.log", i))
	}
	touchLogFile(t, dir, "forum.log")

	CheckLog(dir, "forum.log")

	remaining := remainingRotatedLogs(t, dir)
	if len(remaining) != maxRotatedLogs {
		t.Fatalf("expected exactly %d rotated logs to remain after CheckLog, got %d", maxRotatedLogs, len(remaining))
	}
	if _, err := os.Stat(dir + "forum.log"); err != nil {
		t.Fatalf("expected a fresh forum.log to exist after CheckLog: %v", err)
	}
}
