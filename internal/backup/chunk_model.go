package backup

// Current chunk size is 5 MB, can be changed accordingly
const ChunkSize = 5 * 1024 * 1024

type Chunker struct {
	chunkID int
}

func NewChunker() *Chunker {
	return &Chunker{chunkID: 1}
}

type ChunkData struct {
	ID    int
	Data  []byte
	Files []ChunkFileInfo
	Hash  string
}

type ChunkFileInfo struct {
	Path   string
	Offset int64
	Size   int64
	Hash   string
}
