package models

import "time"

// Chunkinfo:
type ChunkInfo struct {
	ID             int    `json:"id"`
	Filename       string `json:"filename"`
	Size           int64  `json:"size"`
	Hash           string `json:"hash"`
	CompressedSize int64  `json:"compressed_size"`
}

type BackupMetadata struct {
	Version   string              `json:"version"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
	Files     map[string]FileInfo `json:"files"`
	Chunks    []ChunkInfo         `json:"chunks"`
}

type FileInfo struct {
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	ModTime   time.Time `json:"mod_time"`
	Hash      string    `json:"hash"`
	ChunkRefs []int     `json:"chunk_refs"`
	IsDeleted bool      `json:"is_deleted"`
}

type FileEvent struct {
	Path      string
	Operation string // CREATE, MODIFY, DELETE
	Timestamp time.Time
}

type FileChange struct {
	Path      string
	Operation string
	FileInfo  *FileInfo
}
