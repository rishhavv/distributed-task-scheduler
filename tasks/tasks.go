package tasks

import (
	"log"
	"math"
	"math/rand"
	"time"
)

// RunWorkload executes a task with specified characteristics for workload testing
// If parameters are not specified, random values are selected based on common testing patterns
func RunWorkload(taskType string, workerID string, taskName string, value ...interface{}) (int, error) {
	// If no task type specified, randomly pick one with varied resource profiles
	if taskType == "" {
		types := []string{"cpu", "io", "memory", "cpu-io", "cpu-memory", "io-memory", "balanced"}
		taskType = types[rand.Intn(len(types))]
	}

	// Extract optional parameters
	var taskValue int
	var pattern, duration string

	if len(value) > 0 {
		if v, ok := value[0].(int); ok {
			taskValue = v
		}
	}
	if len(value) > 1 {
		if p, ok := value[1].(string); ok {
			pattern = p
		}
	}
	if len(value) > 2 {
		if d, ok := value[2].(string); ok {
			duration = d
		}
	}

	// If no arrival pattern specified, randomly pick one
	if pattern == "" {
		patterns := []string{"steady", "bursty", "ramp-up", "ramp-down", "sinusoidal"}
		pattern = patterns[rand.Intn(len(patterns))]
	}

	// If no duration specified, randomly pick a time horizon
	if duration == "" {
		durations := []string{
			"smoke",  // 5-15 minutes
			"short",  // 30-60 minutes
			"medium", // 2-4 hours
			"long",   // 8-12 hours
			"soak",   // 24+ hours
		}
		duration = durations[rand.Intn(len(durations))]
	}

	// If no value provided, generate based on duration
	if taskValue == 0 {
		switch duration {
		case "smoke":
			taskValue = rand.Intn(2000) + 1
		case "short":
			taskValue = rand.Intn(2000) + 1
		case "medium":
			taskValue = rand.Intn(2000) + 1
		case "long", "soak":
			taskValue = rand.Intn(2000) + 1
		}
	}

	// Apply arrival pattern modifications
	switch pattern {
	case "bursty":
		if rand.Float32() < 0.7 { // 70% chance of burst
			taskValue = min(taskValue+1000, taskValue*3)
		}
	case "ramp-up":
		factor := float64(time.Now().Unix()%3600) / 3600.0
		increase := min(1000, int(float64(taskValue)*factor))
		taskValue = taskValue + increase
	case "ramp-down":
		factor := 1 - (float64(time.Now().Unix()%3600) / 3600.0)
		increase := min(1000, int(float64(taskValue)*factor))
		taskValue = taskValue + increase
	case "sinusoidal":
		factor := 0.5 + 0.5*math.Sin(float64(time.Now().Unix()%3600)/3600.0*2*math.Pi)
		increase := min(1000, int(float64(taskValue)*factor))
		taskValue = taskValue + increase
	}
	log.Printf("Worker %s running task %s with value %d", workerID, taskType, taskValue)

	// Route to appropriate task runner based on type
	switch taskType {
	case "cpu":
		return RunTask("", taskValue)
	case "io":
		return RunIOTask("", taskValue/100) // 100x easier
	case "memory":
		return RunMemoryTask(taskName, taskValue)
	case "cpu-io":
		cpuResult, _ := RunTask(taskName, taskValue/2)
		ioResult, err := RunIOTask(taskName, taskValue/200) // 100x easier
		return cpuResult + ioResult, err
	case "cpu-memory":
		cpuResult, _ := RunTask(taskName, taskValue/2)
		memResult, err := RunMemoryTask(taskName, taskValue/2)
		return cpuResult + memResult, err
	case "io-memory":
		ioResult, _ := RunIOTask(taskName, taskValue/200) // 100x easier
		memResult, err := RunMemoryTask(taskName, taskValue/2)
		return ioResult + memResult, err
	case "balanced":
		cpuResult, _ := RunTask(taskName, taskValue/3)
		ioResult, _ := RunIOTask(taskName, taskValue/300) // 100x easier
		memResult, err := RunMemoryTask(taskName, taskValue/3)
		return cpuResult + ioResult + memResult, err
	default:
		return RunTask("", taskValue)
	}
}
