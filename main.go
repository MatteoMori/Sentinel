/*
sentinel - A Kubernetes controller that tracks container images across workloads
*/

package main

import (
	"os"
	"os/signal"
	"syscall"

	sentinel "github.com/MatteoMori/sentinel/cmd/sentinel"
)

func main() {
	// Set up channel to listen for interrupt or terminate signals for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	// Run sentinel.Execute in a goroutine so we can listen for signals concurrently
	done := make(chan struct{})
	go func() {
		sentinel.Execute() // Execute the root CLI command (./cmd/sentinel/root.go)
		close(done)
	}()

	select {
	case <-sigs:
		// Handle graceful shutdown here if needed
		// For example, we could call a shutdown function in sentinel package
	case <-done:
		// sentinel.Execute finished execution
	}
}
