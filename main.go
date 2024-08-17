package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/shirou/gopsutil/mem"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Process struct {
	Id      int
	User    string
	Host    string
	Db      string
	Command string
	Time    int
	State   string
	Info    string
}

func main() {
	// Database connection details
	dsn := "root:@tcp(127.0.0.1:3306)/thevantive_daily_b1?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}

	// Track the last time the endpoint was hit
	var lastHitTime time.Time

	for {
		// Get the current time
		now := time.Now()
		hour := now.Hour()

		// Check if the current time is within the range 15:00 to 17:00
		if hour >= 13 && hour <= 17 {
			var processlist []Process

			// Execute the query to get the process list
			err := db.Raw("SHOW PROCESSLIST").Scan(&processlist).Error
			if err != nil {
				log.Printf("Error querying process list: %v", err)
			}

			// Get and print RAM information
			v, err := mem.VirtualMemory()
			if err != nil {
				log.Printf("Error getting memory info: %v", err)
			}

			// Calculate memory usage percentage and connection length
			memoryUsage := v.UsedPercent
			connections := len(processlist)

			fmt.Printf("memory usage: %.2f%% connections: %d \n", memoryUsage, connections)

			// Check if memory usage is more than 60% or connections more than 200
			if memoryUsage > 60 || connections > 200 {
				// Check if 5 minutes have passed since the last hit
				if time.Since(lastHitTime) >= 5*time.Minute {
					// Hit the endpoint
					go hitEndpoint(connections, memoryUsage)

					// Update the last hit time
					lastHitTime = time.Now()
				}
			}
		}

		// Sleep for 2 seconds before the next iteration
		time.Sleep(1 * time.Second)
	}
}

// hitEndpoint sends a GET request to the specified endpoint with the connections and memory percentage as parameters
func hitEndpoint(connections int, memoryUsage float64) {
	url := "http://localhost:8081/monitoring/alert" // Replace with your actual endpoint URL

	// Create the data to send in the POST request body
	data := map[string]interface{}{
		"connections":  connections,
		"memory_usage": memoryUsage,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshalling data: %v", err)
		return
	}

	// Create the POST request with the JSON body
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return
	}

	// Set the content type to application/json
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Error hitting endpoint: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Endpoint hit successfully: %v\n", resp.Status)
}
