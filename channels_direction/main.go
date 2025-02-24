package main

import (
	"fmt"
	"os"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

/*
   Directionally Typed Channel Parameters

   Function parameters can specify whether a channel is for sending (chan<- T),
   receiving (<-chan T), or both (chan T). This allows for stricter compile-time checks,
   improving code safety and readability.

   1. Bidirectional Channels (chan T)
      - Can send and receive values.
      - Used when a function needs full control over the channel.

   2. Send-Only Channels (chan<- T)
      - Can only send values into the channel.
      - Prevents accidental reads from the channel, making the intent clear.
      - Useful for functions that only produce data, such as workers or event emitters.

   3. Receive-Only Channels (<-chan T)
      - Can only receive values from the channel.
      - Prevents accidental writes, improving safety in concurrent operations.
      - Useful for functions that only consume data, such as loggers or aggregators.

   Pros of Using Directionally Typed Channels:
   - Improves code clarity and intent, reducing accidental misuse.
   - Enables compile-time checks to catch misuses early.
   - Encourages better separation of concerns in concurrent designs.
   - Enhances safety in multi-goroutine programs by enforcing proper channel use.

   Cons and Limitations:
   - Less flexible: A unidirectional channel cannot be reassigned to a bidirectional one.
   - Can require additional conversions, such as passing a bidirectional channel to a send-only or receive-only function.
   - Might slightly increase verbosity in simpler codebases.

   When to Use:
   - Use bidirectional channels (chan T) when a function needs both read and write access.
   - Use send-only channels (chan<- T) in producer functions that only send data.
   - Use receive-only channels (<-chan T) in consumer functions that only read data.
   - Helps modularize code, ensuring proper separation of message production and consumption.
*/

// notice the lack of arrows around the chan keyword
func doesBoth(standardChannel chan string, msg string) {
	received := <-standardChannel
	fmt.Printf("'doesBoth()' received message '%s' \n", received)

	standardChannel <- msg
	fmt.Printf("'doesBoth()' sent message '%s' \n", msg)
}

// here we see the arrow pointing into the chan keyword, showing that we can only write IN to it
func sendOnly(outbound chan<- string, msg string) {
	outbound <- msg
	fmt.Printf("'sendOnly()' sent message '%s' \n", msg)

	//tryReceive := <- outbound // would not work because outbound is of send-only type
}

// whereas here, the arrow is pointing away from the chan keyword, indicating that we can only read OUT of it
func receiveOnly(inbound <-chan string) {
	msg := <-inbound
	fmt.Printf("'receiveOnly()' received message '%s' \n", msg)

	//inbound <- "trying to send..." // would not work because inbound is of receive-only type
}

func run() error {
	// using a buffer purely to remove the requirement of goroutines,
	// thus, reducing the example's complexity and lines of code
	channel := make(chan string, 3)
	channel <- "hi!"

	// this can, and does, do both operations
	doesBoth(channel, "hello!")

	// this can only write into the channel
	sendOnly(channel, "aloha!")

	// this can only receive from the channel
	receiveOnly(channel)

	return nil
}
