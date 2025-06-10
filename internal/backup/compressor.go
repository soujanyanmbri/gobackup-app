package backup

import (
	"bytes"
	"compress/gzip"
	"io"
)

type Compressor struct{}

func NewCompressor() *Compressor {
	return &Compressor{}
}

func (c *Compressor) Compress(data []byte) ([]byte, error) {
	var compressed bytes.Buffer

	gzWriter := gzip.NewWriter(&compressed)
	if _, err := gzWriter.Write(data); err != nil {
		gzWriter.Close()
		return nil, err
	}

	if err := gzWriter.Close(); err != nil {
		return nil, err
	}

	return compressed.Bytes(), nil
}

func (c *Compressor) Decompress(data []byte) ([]byte, error) {
	reader := bytes.NewReader(data)
	gzReader, err := gzip.NewReader(reader)
	if err != nil {
		return nil, err
	}
	defer gzReader.Close()

	return io.ReadAll(gzReader)
}
