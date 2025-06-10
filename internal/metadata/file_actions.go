package metadata

import "gobackup/pkg/models"

func (m *Manager) UpdateFileInfo(path string, info models.FileInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metadata.Files[path] = info
}

func (m *Manager) MarkFileDeleted(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if info, exists := m.metadata.Files[path]; exists {
		info.IsDeleted = true
		m.metadata.Files[path] = info
	}
}
func (m *Manager) AddChunk(chunk models.ChunkInfo) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.metadata.Chunks = append(m.metadata.Chunks, chunk)
}
