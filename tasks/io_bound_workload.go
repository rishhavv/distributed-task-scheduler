package tasks

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"
)

// IOBoundTasks returns a list of predefined I/O-intensive tasks for evaluation
func IOBoundTasks() []struct {
	ID          string
	Name        string
	Value       int
	Description string
	IOTime      int // Estimated I/O wait time in seconds
} {
	return []struct {
		ID          string
		Name        string
		Value       int
		Description string
		IOTime      int
	}{
		{
			ID:          "large-file-read",
			Name:        "fileread", 
			Value:       1000000000,
			Description: "Read 1GB file from disk sequentially",
			IOTime:      10,
		},
		{
			ID:          "large-file-write",
			Name:        "filewrite",
			Value:       1000000000, 
			Description: "Write 1GB file to disk sequentially",
			IOTime:      12,
		},
		{
			ID:          "random-io",
			Name:        "randomio",
			Value:       100000,
			Description: "Perform 100K random disk seeks and reads",
			IOTime:      15,
		},
		{
			ID:          "network-download",
			Name:        "download",
			Value:       500000000,
			Description: "Download 500MB file from remote server",
			IOTime:      20,
		},
		{
			ID:          "network-upload", 
			Name:        "upload",
			Value:       200000000,
			Description: "Upload 200MB file to remote server",
			IOTime:      18,
		},
		{
			ID:          "db-reads",
			Name:        "dbreads",
			Value:       50000,
			Description: "Perform 50K database read operations",
			IOTime:      8,
		},
		{
			ID:          "db-writes",
			Name:        "dbwrites", 
			Value:       20000,
			Description: "Perform 20K database write operations",
			IOTime:      10,
		},
		{
			ID:          "api-calls",
			Name:        "apicalls",
			Value:       1000,
			Description: "Make 1000 HTTP API calls to external service",
			IOTime:      25,
		},
		{
			ID:          "file-copy",
			Name:        "filecopy",
			Value:       500000000,
			Description: "Copy 500MB file between disk locations",
			IOTime:      15,
		},
		{
			ID:          "log-processing",
			Name:        "logprocess",
			Value:       100000,
			Description: "Process and aggregate 100K log file entries",
			IOTime:      12,
		},
	}
}

func RunIOTask(taskName string, value int) (int, error) {
	// If no task specified, pick a random one
	if taskName == "" {
		tasks := IOBoundTasks()
		randomIndex := rand.Intn(len(tasks))
		taskName = tasks[randomIndex].Name

		// If no value specified, use predefined value
		if value == 0 {
			value = tasks[randomIndex].Value
		}
	}

	switch taskName {
	case "fileread":
		return SimulateFileRead(value), nil
	case "filewrite":
		return SimulateFileWrite(value), nil
	case "randomio":
		return SimulateRandomIO(value), nil
	case "download":
		return SimulateDownload(value), nil
	case "upload":
		return SimulateUpload(value), nil
	case "dbreads":
		return SimulateDBReads(value), nil
	case "dbwrites":
		return SimulateDBWrites(value), nil
	case "apicalls":
		return SimulateAPICalls(value), nil
	case "filecopy":
		return SimulateFileCopy(value), nil
	case "logprocess":
		return SimulateLogProcessing(value), nil
	default:
		return 0, fmt.Errorf("unknown IO task: %s", taskName)
	}
}

// Task simulation functions

func SimulateFileRead(size int) int {
	f, err := os.CreateTemp("", "iotest")
	if err != nil {
		return 0
	}
	defer os.Remove(f.Name())
	defer f.Close()

	// Write random data
	written := 0
	buf := make([]byte, 8192)
	for written < size {
		rand.Read(buf)
		n, _ := f.Write(buf)
		written += n
	}

	// Read it back
	f.Seek(0, 0)
	read := 0
	for read < size {
		n, _ := f.Read(buf)
		if n == 0 {
			break
		}
		read += n
	}
	return read
}

func SimulateFileWrite(size int) int {
	f, err := os.CreateTemp("", "iotest")
	if err != nil {
		return 0
	}
	defer os.Remove(f.Name())
	defer f.Close()

	written := 0
	buf := make([]byte, 8192)
	for written < size {
		rand.Read(buf)
		n, _ := f.Write(buf)
		written += n
	}
	return written
}

func SimulateRandomIO(operations int) int {
	f, err := os.CreateTemp("", "iotest")
	if err != nil {
		return 0
	}
	defer os.Remove(f.Name())
	defer f.Close()

	// Create large file first
	written := 0
	buf := make([]byte, 8192)
	for written < 100*1024*1024 { // 100MB file
		rand.Read(buf)
		n, _ := f.Write(buf)
		written += n
	}

	// Perform random seeks and reads
	readBuf := make([]byte, 100)
	totalRead := 0
	for i := 0; i < operations; i++ {
		offset := rand.Int63n(int64(written))
		f.Seek(offset, 0)
		n, _ := f.Read(readBuf)
		totalRead += n
	}
	return totalRead
}

func SimulateDownload(size int) int {
	resp, err := http.Get("http://example.com")
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	n, _ := io.Copy(io.Discard, resp.Body)
	return int(n)
}

func SimulateUpload(size int) int {
	// Simulate upload by sleeping proportional to size
	time.Sleep(time.Duration(size/1000000) * time.Millisecond)
	return size
}

func SimulateDBReads(operations int) int {
	// Simulate DB reads with random delays
	total := 0
	for i := 0; i < operations; i++ {
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(5)))
		total++
	}
	return total
}

func SimulateDBWrites(operations int) int {
	// Simulate DB writes with random delays
	total := 0
	for i := 0; i < operations; i++ {
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(8)))
		total++
	}
	return total
}

func SimulateAPICalls(calls int) int {
	total := 0
	for i := 0; i < calls; i++ {
		time.Sleep(time.Millisecond * time.Duration(20+rand.Intn(30)))
		total++
	}
	return total
}

func SimulateFileCopy(size int) int {
	src, err := os.CreateTemp("", "src")
	if err != nil {
		return 0
	}
	defer os.Remove(src.Name())
	defer src.Close()

	dst, err := os.CreateTemp("", "dst") 
	if err != nil {
		return 0
	}
	defer os.Remove(dst.Name())
	defer dst.Close()

	written := 0
	buf := make([]byte, 8192)
	for written < size {
		rand.Read(buf)
		n, _ := io.Copy(dst, src)
		written += int(n)
	}
	return written
}

func SimulateLogProcessing(entries int) int {
	f, err := os.CreateTemp("", "logs")
	if err != nil {
		return 0
	}
	defer os.Remove(f.Name())
	defer f.Close()

	processed := 0
	buf := make([]byte, 100)
	for i := 0; i < entries; i++ {
		rand.Read(buf)
		f.Write(buf)
		processed++
		if i%100 == 0 {
			time.Sleep(time.Millisecond) // Simulate processing time
		}
	}
	return processed
}
