package main

import (
	"testing"
	"time"
	// "os" // For TestMain if more complex setup/teardown needed
)

// TestRunNode_InitializationAndGracefulStop tests the core node initialization sequence
// and the basic start/stop functionality of the consensus engine.
func TestRunNode_InitializationAndGracefulStop(t *testing.T) {
	t.Log("TestRunNode: Attempting to initialize and run node components...")

	engine, err := runNode() // Calls the refactored core logic

	if err != nil {
		t.Fatalf("runNode() returned an error during initialization: %v", err)
	}
	if engine == nil {
		t.Fatal("runNode() returned a nil engine without an error.")
	}
	t.Log("TestRunNode: Node components initialized and consensus engine started successfully.")

	// Let the engine run for a very short period to catch any immediate panics from its loop
	// This also gives the engine's internal ticker a chance to tick at least once if interval is short.
	time.Sleep(100 * time.Millisecond)

	t.Log("TestRunNode: Attempting to stop consensus engine...")
	engine.Stop() // Test graceful stop
	t.Log("TestRunNode: Consensus engine stopped.")

    // Add any further checks if needed, e.g., check logs for specific messages if possible,
    // or ensure no goroutines leaked (requires more advanced testing techniques).
    // For now, successful execution without panic is the primary goal.
}
