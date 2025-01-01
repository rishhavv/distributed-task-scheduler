package tasks

import (
	"crypto/sha256"
	"fmt"
	"log"
	"math"
	"math/cmplx"
	"math/rand"
	"sort"
)

// CPUBoundTasks returns a list of predefined CPU-intensive tasks for evaluation
func CPUBoundTasks() []struct {
	ID          string
	Name        string // Maps to function name in TaskSelector
	Value       int    // Input parameter for the task
	Description string
	CPUTime     int // Estimated execution time in seconds
} {
	return []struct {
		ID          string
		Name        string
		Value       int
		Description string
		CPUTime     int
	}{
		{
			ID:          "fib-40",
			Name:        "fibonacci",
			Value:       40,
			Description: "Calculate 40th Fibonacci number using recursive algorithm",
			CPUTime:     8,
		},
		{
			ID:          "prime-10m",
			Name:        "prime",
			Value:       10000000,
			Description: "Find all prime numbers up to 10 million using Sieve of Eratosthenes",
			CPUTime:     15,
		},
		{
			ID:          "matrix-2000",
			Name:        "matrix",
			Value:       2000,
			Description: "Multiply two 2000x2000 matrices",
			CPUTime:     20,
		},
		{
			ID:          "sort-10m",
			Name:        "sort",
			Value:       10000000,
			Description: "Sort 10 million random integers using quicksort",
			CPUTime:     12,
		},
		{
			ID:          "hash-intensive",
			Name:        "hash",
			Value:       1000000,
			Description: "Perform 1 million SHA-256 hash operations",
			CPUTime:     10,
		},
		{
			ID:          "factorial-100k",
			Name:        "factorial",
			Value:       100000,
			Description: "Calculate factorial of 100,000",
			CPUTime:     5,
		},
		{
			ID:          "mandelbrot",
			Name:        "mandelbrot",
			Value:       4000,
			Description: "Generate 4K resolution Mandelbrot set image",
			CPUTime:     25,
		},
		{
			ID:          "pi-calc",
			Name:        "pi",
			Value:       1000000,
			Description: "Calculate Pi to 1 million digits using BBP formula",
			CPUTime:     18,
		},
		{
			ID:          "compression",
			Name:        "compress",
			Value:       1000000000,
			Description: "Compress 1GB of random data using custom algorithm",
			CPUTime:     30,
		},
		{
			ID:          "neural-sim",
			Name:        "neural",
			Value:       1000,
			Description: "Run basic neural network simulation with 1000 neurons",
			CPUTime:     22,
		},
	}
}

func RunTask(taskName string, value int) (int, error) {
	// If no task specified, pick a random one
	if taskName == "" {
		tasks := CPUBoundTasks()
		randomIndex := rand.Intn(len(tasks))
		taskName = tasks[randomIndex].Name

		// If no value specified, use predefined or random value
		if value == 0 {
			value = tasks[randomIndex].Value
		}
	}

	log.Println("Running task: ", taskName, "with value: ", value)
	switch taskName {
	case "primes":
		return ComputePrimes(value), nil
	case "matrix":
		return MultiplyMatrices(value), nil
	case "sort":
		return SortIntegers(value), nil
	case "hash":
		return PerformHashing(value), nil
	case "factorial":
		return (value), nil
	case "mandelbrot":
		return CalculateMandelbrot(value), nil
	case "pi":
		return CalculatePi(value), nil
	case "compress":
		return CompressData(value), nil
	case "neural":
		return NeuralSimulation(value), nil
	case "fibonacci":
		// 100x easier
		return Fibonacci(value / 100), nil
	default:
		return 0, fmt.Errorf("unknown task: %s", taskName)
	}
}

// Task computation functions

func ComputePrimes(n int) int {
	// Segmented Sieve of Eratosthenes with wheel factorization
	if n < 2 {
		return 0
	}

	// Use wheel factorization with first 3 primes (2,3,5)
	wheel := []int{4, 2, 4, 2, 4, 6, 2, 6}
	wheelSize := len(wheel)

	// Calculate segment size for better cache efficiency
	segmentSize := 32768

	// Initialize first segment
	segment := make([]bool, segmentSize)
	primes := []int{2, 3, 5, 7}

	count := 4  // Count of first 4 primes
	start := 11 // Start after wheel primes

	for low := start; low <= n; {
		// Reset segment
		for i := range segment {
			segment[i] = true
		}

		// Calculate segment bounds
		high := low + segmentSize - 1
		if high > n {
			high = n
		}

		// Sieve segment using known primes
		for _, prime := range primes {
			firstMultiple := (low + prime - 1) / prime * prime
			if firstMultiple < low {
				firstMultiple += prime
			}

			for multiple := firstMultiple; multiple <= high; multiple += prime {
				segment[multiple-low] = false
			}
		}

		// Find new primes in segment and use them for sieving
		for i := 0; i < segmentSize && low+i <= high; i++ {
			if segment[i] {
				count++
				if low+i <= int(math.Sqrt(float64(n))) {
					primes = append(primes, low+i)
				}
			}
		}

		// Move to next segment using wheel
		wheelIndex := 0
		for low <= n {
			low += wheel[wheelIndex]
			wheelIndex = (wheelIndex + 1) % wheelSize
			if low+segmentSize > n {
				break
			}
		}
	}

	return count
}

func MultiplyMatrices(size int) int {
	// Create two random matrices
	m1 := make([][]int, size)
	m2 := make([][]int, size)
	result := make([][]int, size)

	for i := 0; i < size; i++ {
		m1[i] = make([]int, size)
		m2[i] = make([]int, size)
		result[i] = make([]int, size)
		for j := 0; j < size; j++ {
			m1[i][j] = rand.Intn(100)
			m2[i][j] = rand.Intn(100)
		}
	}

	// Multiply matrices
	for i := 0; i < size; i++ {
		for j := 0; j < size; j++ {
			for k := 0; k < size; k++ {
				result[i][j] += m1[i][k] * m2[k][j]
			}
		}
	}

	return result[0][0] // Return sample value
}

func SortIntegers(n int) int {
	// Generate random integers
	nums := make([]int, n)
	for i := 0; i < n; i++ {
		nums[i] = rand.Intn(1000000)
	}

	// Sort using built-in sort
	sort.Ints(nums)

	return nums[0] // Return first sorted number
}

func PerformHashing(n int) int {
	data := []byte("data to hash")
	h := sha256.New()
	count := 0

	for i := 0; i < n; i++ {
		h.Write(data)
		sum := h.Sum(nil)
		count += int(sum[0]) // Add first byte to count
		h.Reset()
	}

	return count
}

func CalculateMandelbrot(size int) int {
	count := 0
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			a := (float64(x)/float64(size))*4 - 2
			b := (float64(y)/float64(size))*4 - 2
			c := complex(a, b)
			z := complex(0, 0)

			for i := 0; i < 100; i++ {
				z = z*z + c
				if cmplx.Abs(z) > 2 {
					count++
					break
				}
			}
		}
	}
	return count
}

func CalculatePi(digits int) int {
	// BBP formula for calculating pi
	pi := 0.0
	for k := 0; k < digits; k++ {
		pi += (1.0 / math.Pow(16, float64(k))) * ((4.0 / (8*float64(k) + 1)) -
			(2.0 / (8*float64(k) + 4)) -
			(1.0 / (8*float64(k) + 5)) -
			(1.0 / (8*float64(k) + 6)))
	}
	return int(pi * 1e6)
}

func CompressData(size int) int {
	// Simple run-length encoding simulation
	data := make([]byte, size)
	for i := 0; i < size; i++ {
		data[i] = byte(rand.Intn(256))
	}

	count := 1
	compressed := 0
	for i := 1; i < len(data); i++ {
		if data[i] == data[i-1] {
			count++
		} else {
			compressed += 2 // Count + value
			count = 1
		}
	}
	return compressed
}

func NeuralSimulation(neurons int) int {
	// Simple neural network simulation
	weights := make([][]float64, neurons)
	for i := 0; i < neurons; i++ {
		weights[i] = make([]float64, neurons)
		for j := 0; j < neurons; j++ {
			weights[i][j] = rand.Float64()
		}
	}

	activations := make([]float64, neurons)
	for i := 0; i < 100; i++ { // 100 iterations
		for j := 0; j < neurons; j++ {
			sum := 0.0
			for k := 0; k < neurons; k++ {
				sum += weights[j][k] * activations[k]
			}
			activations[j] = math.Tanh(sum)
		}
	}

	return int(activations[0] * 1000)
}

func Fibonacci(n int) int {
	if n <= 1 {
		return n
	}
	return Fibonacci(n-1) + Fibonacci(n-2)
}
