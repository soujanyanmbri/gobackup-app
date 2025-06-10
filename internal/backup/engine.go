package backup

import (
	"context"
	"fmt"
	"gobackup/internal/metadata"
	"gobackup/internal/utils"
	"gobackup/pkg/models"
	"log"
	"os"
	"path/filepath"
	"sync"
)

/*
This is the main backup engine now.
1. Initialize() - Set up directories and load metadata
2. Start() - Begin background processing
3. ProcessChanges() - Send file changes as they're detected
4. Shutdown() - Clean stop when done
*/
type Engine struct {
	watchPath    string
	backupPath   string
	metadata     *metadata.Manager
	chunker      *Chunker
	compressor   *Compressor
	changeChan   chan []models.FileChange
	shutdownChan chan struct{}
	wg           sync.WaitGroup
	mu           sync.Mutex
}

func NewEngine(watchPath, backupPath string) *Engine {
	return &Engine{
		watchPath:    watchPath,
		backupPath:   backupPath,
		metadata:     metadata.NewManager(backupPath),
		chunker:      NewChunker(),
		compressor:   NewCompressor(),
		changeChan:   make(chan []models.FileChange, 10),
		shutdownChan: make(chan struct{}),
	}
}
func (e *Engine) Initialize() error {
	if err := utils.EnsureDirectoryExists(e.backupPath); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	if err := e.metadata.LoadMetadata(); err != nil {
		return fmt.Errorf("failed to load metadata: %w", err)
	}

	return nil
}
func (e *Engine) Start(ctx context.Context) error {
	e.wg.Add(1)
	go e.processChanges(ctx)

	return nil
}
func (e *Engine) ProcessChanges(changes []models.FileChange) error {
	select {
	case e.changeChan <- changes:
		return nil
	case <-e.shutdownChan:
		return fmt.Errorf("backup engine is shutting down")
	}
}
func (e *Engine) processChanges(ctx context.Context) {
	defer e.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-e.shutdownChan:
			return
		case changes := <-e.changeChan:
			if err := e.handleChanges(changes); err != nil {
				log.Printf("Error processing changes: %v", err)
			}
		}
	}
}

func (e *Engine) handleChanges(changes []models.FileChange) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	var filesToBackup []string

	for _, change := range changes {
		switch change.Operation {
		case "CREATE", "MODIFY":
			if change.FileInfo != nil {
				fullPath := filepath.Join(e.watchPath, change.Path)
				filesToBackup = append(filesToBackup, fullPath)

				e.metadata.UpdateFileInfo(change.Path, *change.FileInfo)
			}
		case "DELETE":
			e.metadata.MarkFileDeleted(change.Path)
		}
	}

	if len(filesToBackup) > 0 {
		if err := e.createBackupChunks(filesToBackup); err != nil {
			return fmt.Errorf("failed to create backup chunks: %w", err)
		}
	}

	return e.metadata.SaveMetadata()
}

func (e *Engine) createBackupChunks(files []string) error {
	chunks, err := e.chunker.CreateChunks(files)
	if err != nil {
		return err
	}

	for _, chunk := range chunks {
		compressed, err := e.compressor.Compress(chunk.Data)
		if err != nil {
			return fmt.Errorf("failed to compress chunk %d: %w", chunk.ID, err)
		}

		chunkFilename := fmt.Sprintf("chunk_%06d.gz", chunk.ID)
		chunkPath := filepath.Join(e.backupPath, chunkFilename)

		if err := os.WriteFile(chunkPath, compressed, 0644); err != nil {
			return fmt.Errorf("failed to write chunk file: %w", err)
		}

		chunkInfo := models.ChunkInfo{
			ID:             chunk.ID,
			Filename:       chunkFilename,
			Size:           int64(len(chunk.Data)),
			Hash:           chunk.Hash,
			CompressedSize: int64(len(compressed)),
		}

		e.metadata.AddChunk(chunkInfo)

		// Update file info with chunk references
		for _, fileInfo := range chunk.Files {
			relPath, _ := filepath.Rel(e.watchPath, fileInfo.Path)
			if storedInfo, exists := e.metadata.GetFileInfo(relPath); exists {
				storedInfo.ChunkRefs = append(storedInfo.ChunkRefs, chunk.ID)
				e.metadata.UpdateFileInfo(relPath, storedInfo)
			}
		}

		log.Printf("Created chunk %s with %d files", chunkFilename, len(chunk.Files))
	}

	return nil
}

func (e *Engine) PerformFullBackup() error {
	changes, err := e.metadata.DetectChanges(e.watchPath)
	if err != nil {
		return fmt.Errorf("failed to detect changes: %w", err)
	}

	log.Printf("Detected %d changes for full backup", len(changes))
	return e.handleChanges(changes)
}

func (e *Engine) Shutdown() {
	close(e.shutdownChan)
	// gracefully shutdown now

	e.wg.Wait()
}
