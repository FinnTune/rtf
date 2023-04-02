package logfiles

import (
	"log"
	"os"
	"time"
)

func CheckLog(dir string, filename string) {
	// Check to see if the log file forum.log exist. If it doesn't create it. If it does rename old file with date and create a new one.
	if _, err := os.Stat(dir + filename); os.IsNotExist(err) {
		// Create new file
		file, err := os.Create(dir + filename)
		if err != nil {
			log.Fatalf("Log file could not be created: %s", err)
		}
		log.Printf("Log file created.")
		defer file.Close()
	} else {
		// Rename existing file with timestamp in filename
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		newFilename := "forum_" + timestamp + ".log"
		err := os.Rename(dir+filename, dir+newFilename)
		if err != nil {
			log.Fatalf("Log file could not be renamed: %s", err)
		}
		// Create new file
		file, err := os.Create(dir + filename)
		if err != nil {
			log.Fatalf("Log file could not be created after renaming: %s", err)
		}
		log.Printf("Previous log file renamed and new log created.")
		defer file.Close()
	}
}
