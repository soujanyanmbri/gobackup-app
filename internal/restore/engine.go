package restore

import (
	"fmt"
	"gobackup/internal/backup"
	"gobackup/internal/metadata"
	"gobackup/internal/utils"
	"gobackup/pkg/models"
	"log"
	"os"
	"path/filepath"
	"time"
)

type Engine struct {
	backupPath string
	targetPath string
	metadata   *metadata.Manager
	compressor *backup.Compressor
	chunker    *backup.Chunker
}

func NewEngine(backupPath, targetPath string) (*Engine, error) {
	return &Engine{
		backupPath: backupPath,
		targetPath: targetPath,
		metadata:   metadata.NewManager(backupPath),
		compressor: backup.NewCompressor(),
		chunker:    backup.NewChunker(),
	}, nil
}

func (e *Engine) Initialize() error {
	if err := e.metadata.LoadMetadata(); err != nil {
		return fmt.Errorf("failed to load backup metadata: %w", err)
	}

	if err := utils.EnsureDirectoryExists(e.targetPath); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	return nil
}

func (e *Engine) ValidateBackup() error {
	meta := e.metadata.GetMetadata()

	for _, chunk := range meta.Chunks {
		chunkPath := filepath.Join(e.backupPath, chunk.Filename)
		if _, err := os.Stat(chunkPath); os.IsNotExist(err) {
			return fmt.Errorf("chunk file missing: %s", chunk.Filename)
		}
	}

	log.Printf("Backup validation completed: %d chunks verified", len(meta.Chunks))
	return nil
}
func (e *Engine) RestoreAll() error {
	if err := e.ValidateBackup(); err != nil {
		return err
	}

	meta := e.metadata.GetMetadata()

	chunkMap := make(map[int]models.ChunkInfo)
	for _, chunk := range meta.Chunks {
		chunkMap[chunk.ID] = chunk
	}

	for _, fileInfo := range meta.Files {
		if fileInfo.IsDeleted {
			continue
		}

		if err := e.restoreFile(fileInfo, chunkMap); err != nil {
			log.Printf("Failed to restore file %s: %v", fileInfo.Path, err)
			continue
		}

		log.Printf("Restored file: %s", fileInfo.Path)
	}

	return nil
}
func (e *Engine) restoreFile(fileInfo models.FileInfo, chunkMap map[int]models.ChunkInfo) error {
	targetFilePath := filepath.Join(e.targetPath, fileInfo.Path)

	targetDir := filepath.Dir(targetFilePath)
	if err := utils.EnsureDirectoryExists(targetDir); err != nil {
		return err
	}

	var fileData []byte

	for _, chunkID := range fileInfo.ChunkRefs {
		chunkInfo, exists := chunkMap[chunkID]
		if !exists {
			return fmt.Errorf("chunk %d not found", chunkID)
		}

		chunkPath := filepath.Join(e.backupPath, chunkInfo.Filename)
		compressedData, err := os.ReadFile(chunkPath)
		if err != nil {
			return fmt.Errorf("failed to read chunk file: %w", err)
		}

		chunkData, err := e.compressor.Decompress(compressedData)
		if err != nil {
			return fmt.Errorf("failed to decompress chunk: %w", err)
		}

		// Verify chunk hash
		if hash := utils.CalculateDataHash(chunkData); hash != chunkInfo.Hash {
			return fmt.Errorf("chunk %d hash verification failed", chunkID)
		}

		// Find file data within chunk (this is simplified - in reality we'd need to store file boundaries within chunks)
		fileData = append(fileData, chunkData...)
	}

	if err := os.WriteFile(targetFilePath, fileData, 0644); err != nil {
		return fmt.Errorf("failed to write restored file: %w", err)
	}

	if err := os.Chtimes(targetFilePath, fileInfo.ModTime, fileInfo.ModTime); err != nil {
		log.Printf("Warning: failed to restore timestamp for %s: %v", fileInfo.Path, err)
	}

	return nil
}

func (e *Engine) ListFiles() error {
	meta := e.metadata.GetMetadata()

	fmt.Printf("Backup created: %s\n", meta.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Last updated: %s\n", meta.UpdatedAt.Format(time.RFC3339))
	fmt.Printf("Total chunks: %d\n\n", len(meta.Chunks))

	activeFiles := 0
	deletedFiles := 0

	fmt.Println("Files in backup:")
	fmt.Println("================")

	for path, fileInfo := range meta.Files {
		status := "ACTIVE"
		if fileInfo.IsDeleted {
			status = "DELETED"
			deletedFiles++
		} else {
			activeFiles++
		}

		fmt.Printf("%-8s %10d bytes  %s  %s\n",
			status, fileInfo.Size, fileInfo.ModTime.Format("2006-01-02 15:04:05"), path)
	}

	fmt.Printf("\nSummary: %d active files, %d deleted files\n", activeFiles, deletedFiles)
	return nil
}
