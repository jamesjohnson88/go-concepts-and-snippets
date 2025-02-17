package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type MyState struct {
	Id     string
	Values []int
}

var states = map[string]*MyState{
	"state#1": {
		Id:     "state#1",
		Values: []int{1, 2, 3},
	},
	"state#2": {
		Id:     "state#2",
		Values: []int{4, 5, 6},
	},
	"state#3": {
		Id:     "state#3",
		Values: []int{7, 8, 9},
	},
	"state#4": {
		Id:     "state#4",
		Values: []int{10, 11, 12},
	},
	"state#5": {
		Id:     "state#5",
		Values: []int{13, 14, 15},
	},
	"state#6": {
		Id:     "state#6",
		Values: []int{16, 17, 18},
	},
}

func main() {
	println("cache started")
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
	println("cache exited")
}

func run() error {
	// Create a ctx with cancel
	ctx, cancel := context.WithCancel(context.Background())

	// EXAMPLE
	cache := NewMyStateCache(ctx)

	backoff := 10 * time.Second
	for _, state := range states {
		err := cache.Set(state, backoff)
		if err != nil {
			log.Printf("cache set error: %s", err)
		}
		backoff = backoff * 2
	}

	// WaitGroup to manage goroutines
	var wg sync.WaitGroup
	wg.Add(1)

	// a goroutine that waits for shutdown
	go func() {
		defer wg.Done()
		<-ctx.Done() // Wait for cancellation
		log.Print("exiting...")
	}()

	// Handle OS interrupt (CTRL+C) for proper shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for a signal
	<-sigChan
	log.Print("SIGTERM received, shutting down...")
	cache.Shutdown()
	cancel() // Trigger ctx cancellation

	wg.Wait() // Ensure all goroutines complete before exiting
	return nil
}
