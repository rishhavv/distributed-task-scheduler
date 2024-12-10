package tasks

import (
	"fmt"
	"math/rand"
	"runtime"
	"time"
)

// MemoryBoundTasks returns a list of predefined memory-intensive tasks for evaluation
func MemoryBoundTasks() []struct {
	ID          string
	Name        string
	Value       int
	Description string
	MemoryMB    int // Estimated memory usage in MB
} {
	return []struct {
		ID          string
		Name        string
		Value       int
		Description string
		MemoryMB    int
	}{
		{
			ID:          "large-array",
			Name:        "array",
			Value:       100000000,
			Description: "Allocate and manipulate 100M element array",
			MemoryMB:    800,
		},
		{
			ID:          "cache-sim",
			Name:        "cache",
			Value:       1000000,
			Description: "In-memory cache simulation with 1M entries",
			MemoryMB:    500,
		},
		{
			ID:          "graph-proc",
			Name:        "graph",
			Value:       500000,
			Description: "Process large graph with 500K nodes",
			MemoryMB:    1200,
		},
		{
			ID:          "image-batch",
			Name:        "images",
			Value:       1000,
			Description: "Batch process 1000 high-res images in memory",
			MemoryMB:    2000,
		},
		{
			ID:          "string-ops",
			Name:        "strings",
			Value:       2000000,
			Description: "String operations on 2M large strings",
			MemoryMB:    400,
		},
		{
			ID:          "ml-model",
			Name:        "mlmodel",
			Value:       100000,
			Description: "Load ML model with 100K parameters",
			MemoryMB:    1500,
		},
		{
			ID:          "data-dedup",
			Name:        "dedup",
			Value:       5000000,
			Description: "In-memory deduplication of 5M records",
			MemoryMB:    900,
		},
		{
			ID:          "sparse-matrix",
			Name:        "sparse",
			Value:       1000000,
			Description: "Sparse matrix operations with 1M elements",
			MemoryMB:    600,
		},
		{
			ID:          "bloom-filter",
			Name:        "bloom",
			Value:       10000000,
			Description: "Bloom filter with 10M entries",
			MemoryMB:    300,
		},
		{
			ID:          "tree-index",
			Name:        "tree",
			Value:       3000000,
			Description: "Build in-memory tree index with 3M nodes",
			MemoryMB:    700,
		},
	}
}

func RunMemoryTask(taskName string, value int) (int, error) {
	// If no task specified, pick a random one
	if taskName == "" {
		tasks := MemoryBoundTasks()
		randomIndex := rand.Intn(len(tasks))
		taskName = tasks[randomIndex].Name

		// If no value specified, use predefined value
		if value == 0 {
			value = tasks[randomIndex].Value
		}
	}

	switch taskName {
	case "array":
		return SimulateLargeArray(value), nil
	case "cache":
		return SimulateCache(value), nil
	case "graph":
		return SimulateGraphProcessing(value), nil
	case "images":
		return SimulateImageProcessing(value), nil
	case "strings":
		return SimulateStringOperations(value), nil
	case "mlmodel":
		return SimulateMLModel(value), nil
	case "dedup":
		return SimulateDataDeduplication(value), nil
	case "sparse":
		return SimulateSparseMatrix(value), nil
	case "bloom":
		return SimulateBloomFilter(value), nil
	case "tree":
		return SimulateTreeIndex(value), nil
	default:
		return 0, fmt.Errorf("unknown memory task: %s", taskName)
	}
}

// Task simulation functions

func SimulateLargeArray(size int) int {
	data := make([]int, size)
	for i := 0; i < size; i++ {
		data[i] = rand.Int()
	}
	runtime.GC() // Force GC to clean up
	return len(data)
}

func SimulateCache(entries int) int {
	cache := make(map[int][]byte)
	for i := 0; i < entries; i++ {
		data := make([]byte, 512) // 512 bytes per entry
		rand.Read(data)
		cache[i] = data
	}
	runtime.GC()
	return len(cache)
}

func SimulateGraphProcessing(nodes int) int {
	// Simulate graph as adjacency list
	graph := make([][]int, nodes)
	for i := 0; i < nodes; i++ {
		edges := rand.Intn(100)
		graph[i] = make([]int, edges)
		for j := 0; j < edges; j++ {
			graph[i][j] = rand.Intn(nodes)
		}
	}
	runtime.GC()
	return nodes
}

func SimulateImageProcessing(count int) int {
	// Simulate image data (RGBA, 1920x1080)
	imageSize := 1920 * 1080 * 4
	images := make([][]byte, count)
	for i := 0; i < count; i++ {
		images[i] = make([]byte, imageSize)
		rand.Read(images[i])
	}
	runtime.GC()
	return count
}

func SimulateStringOperations(count int) int {
	strings := make([]string, count)
	for i := 0; i < count; i++ {
		bytes := make([]byte, 1024)
		rand.Read(bytes)
		strings[i] = string(bytes)
	}
	runtime.GC()
	return len(strings)
}

func SimulateMLModel(params int) int {
	// Simulate ML model weights as float64 arrays
	weights := make([]float64, params)
	biases := make([]float64, params)
	for i := 0; i < params; i++ {
		weights[i] = rand.Float64()
		biases[i] = rand.Float64()
	}
	runtime.GC()
	return params
}

func SimulateDataDeduplication(records int) int {
	seen := make(map[string]bool)
	for i := 0; i < records; i++ {
		data := make([]byte, 100)
		rand.Read(data)
		seen[string(data)] = true
	}
	runtime.GC()
	return len(seen)
}

func SimulateSparseMatrix(elements int) int {
	// Simulate sparse matrix using map
	matrix := make(map[int]map[int]float64)
	for i := 0; i < elements; i++ {
		row := rand.Intn(10000)
		col := rand.Intn(10000)
		if matrix[row] == nil {
			matrix[row] = make(map[int]float64)
		}
		matrix[row][col] = rand.Float64()
	}
	runtime.GC()
	return elements
}

func SimulateBloomFilter(items int) int {
	// Simple bloom filter simulation
	filterSize := items * 10
	filter := make([]bool, filterSize)
	for i := 0; i < items; i++ {
		// Simulate multiple hash functions
		for j := 0; j < 3; j++ {
			pos := rand.Intn(filterSize)
			filter[pos] = true
		}
	}
	runtime.GC()
	return items
}

func SimulateTreeIndex(nodes int) int {
	type TreeNode struct {
		Value    int
		Children []*TreeNode
	}

	var buildTree func(depth int) *TreeNode
	buildTree = func(depth int) *TreeNode {
		if depth == 0 {
			return nil
		}
		node := &TreeNode{Value: rand.Int()}
		children := rand.Intn(5)
		node.Children = make([]*TreeNode, children)
		for i := 0; i < children; i++ {
			node.Children[i] = buildTree(depth - 1)
		}
		return node
	}

	_ = buildTree(10)            // Build tree with max depth 10
	time.Sleep(time.Millisecond) // Simulate processing
	runtime.GC()
	return nodes
}
