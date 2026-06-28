package storage

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FileMeta struct {
	ID           string `json:"id"`
	OriginalName string `json:"original_name"`
	Size         int64  `json:"size"`
	UploadTime   int64  `json:"upload_time"`
}

type FileEngine struct {
	baseDir  string
	metaFile string
	mu       sync.RWMutex
	metaDB   map[string]FileMeta
}

func NewFileEngine(baseDir string) (*FileEngine, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}

	engine := &FileEngine{
		baseDir:  baseDir,
		metaFile: filepath.Join(baseDir, "metadata.json"),
		metaDB:   make(map[string]FileMeta),
	}

	if err := engine.loadMetadata(); err != nil {
		return nil, err
	}
	return engine, nil
}

func (e *FileEngine) loadMetadata() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	data, err := os.ReadFile(e.metaFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, &e.metaDB)
}

func (e *FileEngine) saveMetadataSync() error {
	data, err := json.MarshalIndent(e.metaDB, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(e.metaFile, data, 0644)
}

func (e *FileEngine) SaveStream(src io.Reader, originalName string) (FileMeta, error) {
	tempFile, err := os.CreateTemp(e.baseDir, "upload-*")
	if err != nil {
		return FileMeta{}, err
	}
	tempPath := tempFile.Name()
	defer tempFile.Close()

	hasher := sha256.New()
	tee := io.TeeReader(src, hasher)

	written, err := io.Copy(tempFile, tee)
	if err != nil {
		os.Remove(tempPath)
		return FileMeta{}, err
	}

	hashString := hex.EncodeToString(hasher.Sum(nil))
	finalPath := filepath.Join(e.baseDir, hashString)

	if _, err := os.Stat(finalPath); os.IsNotExist(err) {
		if err := os.Rename(tempPath, finalPath); err != nil {
			os.Remove(tempPath)
			return FileMeta{}, err
		}
	} else {
		os.Remove(tempPath)
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	meta := FileMeta{
		ID:           hashString,
		OriginalName: originalName,
		Size:         written,
		UploadTime:   time.Now().Unix(),
	}
	e.metaDB[hashString] = meta

	if err := e.saveMetadataSync(); err != nil {
		return FileMeta{}, err
	}

	return meta, nil
}

func (e *FileEngine) GetMeta(id string) (FileMeta, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	meta, exists := e.metaDB[id]
	if !exists {
		return FileMeta{}, errors.New("file not found in metadata")
	}
	return meta, nil
}

func (e *FileEngine) GetFilePath(id string) string {
	return filepath.Join(e.baseDir, id)
}

func (e *FileEngine) GetValidIDs() map[string]struct{} {
	e.mu.RLock()
	defer e.mu.RUnlock()

	valid := make(map[string]struct{}, len(e.metaDB))
	for id := range e.metaDB {
		valid[id] = struct{}{}
	}
	return valid
}
