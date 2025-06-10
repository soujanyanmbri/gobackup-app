package metadata

import (
	"encoding/json"
	"gobackup/internal/utils"
	"gobackup/pkg/models"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Manager struct {
	backupPath string
	metadata   *models.BackupMetadata
	mu         sync.RWMutex
}

func NewManager(backupPath string) *Manager {
	return &Manager{
		backupPath: backupPath,
		metadata: &models.BackupMetadata{
			Version:   "1.0",
			CreatedAt: time.Now(),
			Files:     make(map[string]models.FileInfo),
			Chunks:    make([]models.ChunkInfo, 0),
		},
	}
}

func (m *Manager) LoadMetadata() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	metadataPath := filepath.Join(m.backupPath, "metadata.json")

	data, err := os.ReadFile(metadataPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	return json.Unmarshal(data, m.metadata)
}

func (m *Manager) SaveMetadata() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := utils.EnsureDirectoryExists(m.backupPath); err != nil {
		return err
	}

	m.metadata.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(m.metadata, "", "  ")
	if err != nil {
		return err
	}

	metadataPath := filepath.Join(m.backupPath, "metadata.json")
	tempPath := metadataPath + ".tmp"

	// Write to temp file first
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return err
	}

	// Atomic rename
	return os.Rename(tempPath, metadataPath)
}

func (m *Manager) GetFileInfo(path string) (models.FileInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	info, exists := m.metadata.Files[path]
	return info, exists
}

func (m *Manager) GetMetadata() *models.BackupMetadata {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metaCopy := *m.metadata
	metaCopy.Files = make(map[string]models.FileInfo)
	for k, v := range m.metadata.Files {
		metaCopy.Files[k] = v
	}
	metaCopy.Chunks = make([]models.ChunkInfo, len(m.metadata.Chunks))
	copy(metaCopy.Chunks, m.metadata.Chunks)

	return &metaCopy
}

func (m *Manager) DetectChanges(watchPath string) ([]models.FileChange, error) {
	var changes []models.FileChange

	// Walk the directory to find all current files
	currentFiles := make(map[string]models.FileInfo)

	err := filepath.Walk(watchPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(watchPath, path)
		if err != nil {
			return nil
		}

		hash, err := utils.CalculateFileHash(path)
		if err != nil {
			return nil
		}

		currentFiles[relPath] = models.FileInfo{
			Path:    relPath,
			Size:    info.Size(),
			ModTime: info.ModTime(),
			Hash:    hash,
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check for new or modified files
	for path, currentInfo := range currentFiles {
		if storedInfo, exists := m.metadata.Files[path]; exists {
			if !storedInfo.IsDeleted && (storedInfo.Hash != currentInfo.Hash || storedInfo.ModTime != currentInfo.ModTime) {
				changes = append(changes, models.FileChange{
					Path:      path,
					Operation: "MODIFY",
					FileInfo:  &currentInfo,
				})
			}
		} else {
			changes = append(changes, models.FileChange{
				Path:      path,
				Operation: "CREATE",
				FileInfo:  &currentInfo,
			})
		}
	}

	// Check for deleted files
	for path, storedInfo := range m.metadata.Files {
		if !storedInfo.IsDeleted {
			if _, exists := currentFiles[path]; !exists {
				changes = append(changes, models.FileChange{
					Path:      path,
					Operation: "DELETE",
					FileInfo:  nil,
				})
			}
		}
	}

	return changes, nil
}
