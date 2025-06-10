package backup

import (
	"fmt"
	"gobackup/internal/utils"
	"os"
)

func (c *Chunker) CreateChunks(files []string) ([]ChunkData, error) {
	var chunks []ChunkData
	var currentChunk ChunkData
	var currentSize int64

	currentChunk.ID = c.chunkID
	c.chunkID++
	for _, filePath := range files {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			continue
		}

		if fileInfo.IsDir() {
			continue
		}

		if currentSize+fileInfo.Size() > ChunkSize {
			if len(currentChunk.Files) > 0 {
				currentChunk.Hash = utils.CalculateDataHash(currentChunk.Data)
				chunks = append(chunks, currentChunk)
				currentChunk = ChunkData{
					ID: c.chunkID,
				}
				c.chunkID++
				currentSize = 0
			}
		}

		fileData, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		fileHash, err := utils.CalculateFileHash(filePath)
		if err != nil {
			continue
		}

		// Add file to the current chunk
		currentChunk.Files = append(currentChunk.Files, ChunkFileInfo{
			Path:   filePath,
			Offset: int64(len(currentChunk.Data)),
			Size:   fileInfo.Size(),
			Hash:   fileHash,
		})
		currentChunk.Data = append(currentChunk.Data, fileData...)
		currentSize += fileInfo.Size()

	}

	if len(currentChunk.Files) > 0 {
		currentChunk.Hash = utils.CalculateDataHash(currentChunk.Data)
		chunks = append(chunks, currentChunk)
	}

	return chunks, nil
}

func (c *Chunker) ExtractFileFromChunk(chunkData []byte, fileInfo ChunkFileInfo) ([]byte, error) {
	if fileInfo.Offset+fileInfo.Size > int64(len(chunkData)) {
		return nil, fmt.Errorf("file data extends beyond chunk boundary")
	}

	data := chunkData[fileInfo.Offset : fileInfo.Offset+fileInfo.Size]

	if hash := utils.CalculateDataHash(data); hash != fileInfo.Hash {
		return nil, fmt.Errorf("file hash mismatch")
	}

	return data, nil
}
