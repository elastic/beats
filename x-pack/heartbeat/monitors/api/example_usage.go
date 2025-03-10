package api

import (
	"fmt"
	"log"
	"time"
)

func main() {
	// Start the Node.js process
	process, err := StartNodeProcess("node", "./path/to/your/script.js", "stdin")
	if err != nil {
		log.Fatalf("Failed to start Node.js process: %v", err)
	}
	defer process.Close()

	// Give it some time to initialize
	time.Sleep(1 * time.Second)

	// Initialize the main process
	options := map[string]interface{}{
		"options": map[string]interface{}{
			"headless": true,
			"reporter": "json",
		},
	}

	err = process.InitMainProcess(options)
	if err != nil {
		log.Fatalf("Failed to initialize main process: %v", err)
	}

	// Create a worker
	workerData := map[string]interface{}{
		"options": map[string]interface{}{
			"headless": true,
			"reporter": "json",
		},
	}

	err = process.CreateWorker("worker1", workerData)
	if err != nil {
		log.Fatalf("Failed to create worker: %v", err)
	}

	// Send data to worker
	testOptions := map[string]interface{}{
		"action": "run",
		"options": map[string]interface{}{
			"pattern":  "**/*.journey.js",
			"headless": true,
		},
	}

	err = process.SendToWorker("worker1", testOptions)
	if err != nil {
		log.Fatalf("Failed to send data to worker: %v", err)
	}

	// Wait for the worker to finish
	time.Sleep(10 * time.Second)

	// Stop the worker
	err = process.StopWorker("worker1")
	if err != nil {
		log.Fatalf("Failed to stop worker: %v", err)
	}

	fmt.Println("Job completed successfully")
}
