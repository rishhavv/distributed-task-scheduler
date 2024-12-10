package tasks

import (
	"fmt"
	"math/rand"
)

// RunWorkload executes a task of specified type (CPU, IO, or Memory bound)
// If no task type is specified, it randomly picks one
// If no specific task name is provided within the type, it randomly picks one
func RunWorkload(taskType string, taskName string, value int) (int, error) {
	// If no task type specified, randomly pick one
	if taskType == "" {
		types := []string{"cpu", "io", "memory"}
		taskType = types[rand.Intn(len(types))]
	}

	// Route to appropriate task runner based on type
	switch taskType {
	case "cpu":
		return RunTask(taskName, value)
	case "io":
		return RunIOTask(taskName, value)
	case "memory":
		return RunMemoryTask(taskName, value)
	default:
		return 0, fmt.Errorf("unknown task type: %s", taskType)
	}
}
