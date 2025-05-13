package main

import (
	"fmt"
	"time"
)

func main() {
	// Current time
	now := time.Now().UTC()
	fmt.Printf("Current time (UTC): %v\n", now)
	
	// Get time 24 hours from now
	minDeadline := now.AddDate(0, 0, 1)
	fmt.Printf("Minimum deadline (UTC): %v\n", minDeadline)
	
	// Parse an example deadline
	exampleDeadline := "2025-05-29T23:59:59Z"
	deadline, _ := time.Parse(time.RFC3339, exampleDeadline)
	fmt.Printf("Example deadline (UTC): %v\n", deadline)
	
	// Is example deadline valid?
	fmt.Printf("Is example deadline valid? %v\n", deadline.After(minDeadline))
	
	// Current time in RFC3339
	fmt.Printf("\nFor testing, use this timestamp (2 days from now):\n")
	twoDaysLater := now.AddDate(0, 0, 2)
	fmt.Printf("%s\n", twoDaysLater.Format(time.RFC3339))
}