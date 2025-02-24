package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

/*
Buffered vs Unbuffered Channels
===================================

Unbuffered Channels (make(chan T))
---------------------------------
Pros:
- Guaranteed synchronization between sender and receiver
- Immediate backpressure prevents overwhelming the system
- Very predictable memory usage (only one item in transit at a time)
- Natural "handshake" ensures processing order
- Great for precise control flow

Cons:
- Can be slower due to required synchronization
- Sender must wait for receiver to be ready
- Less throughput for batch operations
- May not utilize system resources fully when processing times vary

Best Use Cases:
- When processing order matters
- Memory-constrained systems
- Real-time processing requirements
- When you need guaranteed delivery
- Rate limiting
- When immediate feedback is important

Buffered Channels (make(chan T, size))
------------------------------------
Pros:
- Better throughput for batch operations
- Sender can continue without waiting (until buffer fills)
- Can smooth out processing spikes
- Good for producer/consumer patterns with varying speeds
- More flexible processing patterns

Cons:
- Less predictable processing order
- Can mask backpressure issues
- Memory usage varies with buffer size
- Harder to reason about synchronization
- May hide deadlocks during development

Best Use Cases:
- Batch processing
- When some queueing is acceptable
- Handling bursty workloads
- Decoupling components with different processing speeds
- Performance optimization when order isn't critical
- Background job processing

General Guidelines
----------------
1. Start with unbuffered channels unless you have a specific reason not to
2. Use buffered channels when:
  - You've identified a specific performance bottleneck
  - You need to handle bursty workloads
  - You want to batch process items
3. Buffer size should be set based on careful analysis:
  - Too small: won't help with bursts
  - Too large: can mask problems and waste memory
4. Consider monitoring channel behavior in production:
  - Buffer utilization
  - Blocking duration
  - Processing latency
*/

func run() error {
	// generate a list of test files with random processing times
	files := generateLargeFileList(200)

	// run both channel types and store their execution times
	var times []time.Duration

	fmt.Println("\n=== Unbuffered Channel ===")
	times = append(times, processFiles(files, make(chan string)))

	fmt.Println("\n=== Buffered Channel ===")
	times = append(times, processFiles(files, make(chan string, 5))) // buffer of 5 files

	// compare the results
	fmt.Println("\n=== Performance Comparison ===")
	fmt.Printf("Unbuffered Channel Total Time: %v\n", times[0])
	fmt.Printf("Buffered Channel Total Time: %v\n", times[1])
	fmt.Printf("Difference: %v\n", times[1]-times[0])

	return nil
}

type FileInfo struct {
	name string
	size int
}

func generateLargeFileList(count int) []FileInfo {
	files := make([]FileInfo, count)
	rand.NewSource(time.Now().UnixNano())

	for i := 0; i < count; i++ {
		files[i] = FileInfo{
			name: fmt.Sprintf("file%d.txt", i+1),
			size: rand.Intn(3) + 1,
		}
	}
	return files
}

func processFiles(files []FileInfo, ch chan string) time.Duration {
	startTime := time.Now()
	var wg sync.WaitGroup

	// start multiple worker goroutines to process files
	for w := 0; w < 3; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// worker keeps taking files from channel until it's closed
			for fileName := range ch {
				fileIndex := strings.Index(fileName, "file")
				fileInfo := files[fileIndex]

				fmt.Printf("[%v] Worker %d starting %s (size: %ds)\n",
					time.Since(startTime), workerID, fileName, fileInfo.size)

				// simulate file processing with sleep
				time.Sleep(time.Duration(fileInfo.size) * time.Second)

				fmt.Printf("[%v] Worker %d completed %s\n",
					time.Since(startTime), workerID, fileName)
			}
		}(w)
	}

	// producer goroutine - sends files to the channel
	go func() {
		for _, file := range files {
			sendStart := time.Now()
			fmt.Printf("[%v] Attempting to send %s (size: %ds) to channel\n",
				time.Since(startTime), file.name, file.size)

			ch <- file.name // this will block if channel is unbuffered, or buffer is full

			fmt.Printf("[%v] Finished sending %s (took: %v)\n",
				time.Since(startTime), file.name, time.Since(sendStart))
		}

		// close the channel to signal that no more files are coming
		fmt.Printf("[%v] All files sent, closing channel\n", time.Since(startTime))
		close(ch)
	}()

	wg.Wait()

	executionTime := time.Since(startTime)
	fmt.Printf("\nExecution completed in %v\n", executionTime)
	return executionTime
}
