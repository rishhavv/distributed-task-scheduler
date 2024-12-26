package tasks

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

// RunWorkload executes a task with specified characteristics for workload testing
// If parameters are not specified, random values are selected based on common testing patterns
func RunWorkload(taskType string, taskName string, value ...interface{}) (int, error) {
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
			taskValue = rand.Intn(1000) + 100
		case "short":
			taskValue = rand.Intn(10000) + 1000
		case "medium":
			taskValue = rand.Intn(100000) + 10000
		case "long", "soak":
			taskValue = rand.Intn(1000000) + 100000
		}
	}

	// Apply arrival pattern modifications
	switch pattern {
	case "bursty":
		if rand.Float32() < 0.7 { // 70% chance of burst
			taskValue = taskValue * 3
		}
	case "ramp-up":
		factor := float64(time.Now().Unix()%3600) / 3600.0
		taskValue = int(float64(taskValue) * factor)
	case "ramp-down":
		factor := 1 - (float64(time.Now().Unix()%3600) / 3600.0)
		taskValue = int(float64(taskValue) * factor)
	case "sinusoidal":
		factor := 0.5 + 0.5*math.Sin(float64(time.Now().Unix()%3600)/3600.0*2*math.Pi)
		taskValue = int(float64(taskValue) * factor)
	}

	// Route to appropriate task runner based on type
	switch taskType {
	case "cpu":
		return RunTask(taskName, taskValue)
	case "io":
		return RunIOTask(taskName, taskValue)
	case "memory":
		return RunMemoryTask(taskName, taskValue)
	case "cpu-io":
		cpuResult, _ := RunTask(taskName, taskValue/2)
		ioResult, err := RunIOTask(taskName, taskValue/2)
		return cpuResult + ioResult, err
	case "cpu-memory":
		cpuResult, _ := RunTask(taskName, taskValue/2)
		memResult, err := RunMemoryTask(taskName, taskValue/2)
		return cpuResult + memResult, err
	case "io-memory":
		ioResult, _ := RunIOTask(taskName, taskValue/2)
		memResult, err := RunMemoryTask(taskName, taskValue/2)
		return ioResult + memResult, err
	case "balanced":
		cpuResult, _ := RunTask(taskName, taskValue/3)
		ioResult, _ := RunIOTask(taskName, taskValue/3)
		memResult, err := RunMemoryTask(taskName, taskValue/3)
		return cpuResult + ioResult + memResult, err
	default:
		return 0, fmt.Errorf("unknown task type: %s", taskType)
	}
}
