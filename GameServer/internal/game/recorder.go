package game

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// MatchRecorder handles asynchronous, batched writing of match data to disk
type MatchRecorder struct {
	file       *os.File
	recordChan chan HistoricState
	done       chan struct{}
}

// NewMatchRecorder creates the file and starts the background batch worker
func NewMatchRecorder(matchID string) (*MatchRecorder, error) {
	_ = os.MkdirAll("replays", 0755)

	fileName := fmt.Sprintf("replays/match_%s_%d.jsonl", matchID, time.Now().Unix())
	file, err := os.Create(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create replay file: %w", err)
	}

	mr := &MatchRecorder{
		file:       file,
		recordChan: make(chan HistoricState, 600), // Still buffers incoming ticks to prevent blocking
		done:       make(chan struct{}),
	}

	// Start the background writer thread
	go mr.startBatchWriter()

	log.Printf("🔴 Recording match to %s (Batched 60s)", fileName)
	return mr, nil
}

// startBatchWriter lives in the background, accumulating ticks in RAM and flushing every 60s
func (mr *MatchRecorder) startBatchWriter() {
	// Pre-allocate memory for exactly 60 seconds of ticks (60 ticks * 60 seconds)
	// This prevents the Go Garbage Collector from working overtime resizing the array
	batch := make([]HistoricState, 0, 3600)
	
	// Create a ticker that fires once a minute
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	
	encoder := json.NewEncoder(mr.file)

	for {
		select {
		case state, ok := <-mr.recordChan:
			if !ok {
				// The channel was closed (Match is over or server is shutting down)
				// Flush whatever is left in the batch to disk so we don't lose the end of the match
				mr.flushBatch(encoder, batch)
				mr.file.Close()
				close(mr.done)
				return
			}
			
			batch = append(batch, state)

			// Safety Valve: If the batch somehow gets too large (e.g. server ran long), flush it early
			if len(batch) >= 3600 {
				mr.flushBatch(encoder, batch)
				batch = batch[:0]              // Clear the slice but keep the allocated memory!
				ticker.Reset(60 * time.Second) // Reset the clock
			}

		case <-ticker.C:
			// 60 seconds have passed! Write everything in RAM to the SSD
			if len(batch) > 0 {
				mr.flushBatch(encoder, batch)
				batch = batch[:0] // Clear the slice for the next minute
			}
		}
	}
}

// flushBatch writes the accumulated array to the SSD as individual JSON Lines
func (mr *MatchRecorder) flushBatch(encoder *json.Encoder, batch []HistoricState) {
	startTime := time.Now()
	
	for _, state := range batch {
		if err := encoder.Encode(state); err != nil {
			log.Printf("Failed to write tick %d to replay: %v", state.Tick, err)
		}
	}
	
	// Optional: Helpful for monitoring server health during development
	log.Printf("💾 Flushed %d frames to disk in %v", len(batch), time.Since(startTime))
}

// RecordTick queues a snapshot for writing (Called by the 60Hz loop)
func (mr *MatchRecorder) RecordTick(state HistoricState) {
	select {
	case mr.recordChan <- state:
	default:
		log.Println("WARNING: Replay RAM queue is full, dropping frame!")
	}
}

// Stop safely finishes writing the remaining buffer and closes the file
func (mr *MatchRecorder) Stop() {
	close(mr.recordChan) // Tells startBatchWriter to finish up
	<-mr.done            // Block until the final flush is fully written
	log.Println("⏹️ Match recording stopped and saved.")
}