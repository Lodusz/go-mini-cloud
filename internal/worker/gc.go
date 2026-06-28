package worker

import (
	"context"
	"go-mini-cloud/internal/storage"
	"log"
	"os"
	"path/filepath"
	"time"
)

type GCWorker struct {
	engine   *storage.FileEngine
	interval time.Duration
	baseDir  string
}

func NewGCWorker(engine *storage.FileEngine, baseDir string, interval time.Duration) *GCWorker {
	return &GCWorker{
		engine:   engine,
		interval: interval,
		baseDir:  baseDir,
	}
}

func (gc *GCWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(gc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("STOP GARBAGE COLLECTOR...")
			return
		case <-ticker.C:
			gc.runSweep()
		}
	}
}

func (gc *GCWorker) runSweep() {
	validIDs := gc.engine.GetValidIDs()
	entries, err := os.ReadDir(gc.baseDir)
	if err != nil {
		log.Printf("Error reading directory GARBAGE COLLECTOR: %v\n", err)
		return
	}

	deleted := 0
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "metadata.json" {
			continue
		}

		if _, exists := validIDs[entry.Name()]; !exists {
			fullPath := filepath.Join(gc.baseDir, entry.Name())
			if err := os.Remove(fullPath); err == nil {
				deleted++
			}
		}
	}

	if deleted > 0 {
		log.Printf("Cleanup complete GARBAGE COLLECTOR: %d\n", deleted)
	}
}
