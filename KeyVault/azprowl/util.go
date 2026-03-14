package main

import "time"

// nowString returns the current UTC time as a formatted string.
func nowString() string {
	return time.Now().UTC().Format("2006-01-02 15:04:05 UTC")
}
