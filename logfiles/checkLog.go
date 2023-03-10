package logfiles

import (
	"os"
	"time"
)

func CheckLog(dir string, filename string) {
	// Check to see if the log file forum.log exist. If it doesn't create it. If it does rename old file with date and create a new one.
	if _, err := os.Stat(dir + filename); os.IsNotExist(err) {
		// Create new file
		file, err := os.Create(filename)
		if err != nil {
			panic(err)
		}
		defer file.Close()
	} else {
		// Rename existing file with timestamp in filename
		timestamp := time.Now().Format("2006-01-02_15-04-05")
		newFilename := "forum_" + timestamp + ".log"
		err := os.Rename(dir+filename, dir+newFilename)
		if err != nil {
			panic(err)
		}
		// Create new file
		file, err := os.Create(dir + filename)
		if err != nil {
			panic(err)
		}
		defer file.Close()
	}
}
