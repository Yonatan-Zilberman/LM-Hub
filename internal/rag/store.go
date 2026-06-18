// Package rag implements retrieval-augmented generation.
package rag

import (
	"encoding/json"
	"fmt"
	"time"

	"go.etcd.io/bbolt"
)

// Chunk represents a chunk of text from a codebase file, with its embedding and metadata.
type Chunk struct {
	ID         string    `json:"id"`
	FilePath   string    `json:"file_path"`
	StartLine  int       `json:"start_line"`
	EndLine    int       `json:"end_line"`
	Content    string    `json:"content"`
	Embedding  []float32 `json:"embedding"`
	TokenCount int       `json:"token_count"`
}

// Store handles persistence of chunks and embeddings using a bbolt database.
type Store struct {
	db *bbolt.DB
}

var (
	chunksBucket = []byte("chunks")
	metaBucket   = []byte("meta")
)

// NewStore initializes a new Store with the provided bbolt database.
func NewStore(db *bbolt.DB) (*Store, error) {
	err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(chunksBucket)
		if err != nil {
			return fmt.Errorf("failed to create chunks bucket: %w", err)
		}
		_, err = tx.CreateBucketIfNotExists(metaBucket)
		if err != nil {
			return fmt.Errorf("failed to create meta bucket: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize db buckets: %w", err)
	}

	return &Store{db: db}, nil
}

// PutChunk stores a chunk in the database.
func (s *Store) PutChunk(chunk *Chunk) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(chunksBucket)
		data, err := json.Marshal(chunk)
		if err != nil {
			return fmt.Errorf("failed to marshal chunk: %w", err)
		}
		err = b.Put([]byte(chunk.ID), data)
		if err != nil {
			return fmt.Errorf("failed to save chunk %s: %w", chunk.ID, err)
		}
		return nil
	})
}

// GetChunk retrieves a single chunk by its ID.
func (s *Store) GetChunk(id string) (*Chunk, error) {
	var chunk Chunk
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(chunksBucket)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("chunk not found: %s", id)
		}
		if err := json.Unmarshal(data, &chunk); err != nil {
			return fmt.Errorf("failed to unmarshal chunk: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &chunk, nil
}

// DeleteChunksForFile deletes all chunks belonging to a specific file.
func (s *Store) DeleteChunksForFile(filePath string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(chunksBucket)
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var chunk Chunk
			if err := json.Unmarshal(v, &chunk); err != nil {
				// Log or ignore corrupted chunks in this sweep
				continue
			}
			if chunk.FilePath == filePath {
				if err := b.Delete(k); err != nil {
					return fmt.Errorf("failed to delete chunk %s: %w", string(k), err)
				}
			}
		}
		return nil
	})
}

// GetAllChunks retrieves all chunks stored in the database.
func (s *Store) GetAllChunks() ([]*Chunk, error) {
	var chunks []*Chunk
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(chunksBucket)
		return b.ForEach(func(k, v []byte) error {
			var chunk Chunk
			if err := json.Unmarshal(v, &chunk); err != nil {
				return fmt.Errorf("failed to unmarshal chunk %s: %w", string(k), err)
			}
			chunks = append(chunks, &chunk)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return chunks, nil
}

// GetStats returns the number of files and chunks indexed.
func (s *Store) GetStats() (int, int, error) {
	var chunkCount int
	uniqueFiles := make(map[string]bool)

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(chunksBucket)
		return b.ForEach(func(k, v []byte) error {
			chunkCount++
			var chunk Chunk
			if err := json.Unmarshal(v, &chunk); err == nil {
				uniqueFiles[chunk.FilePath] = true
			}
			return nil
		})
	})
	if err != nil {
		return 0, 0, err
	}

	return len(uniqueFiles), chunkCount, nil
}

// SetMeta sets a metadata key-value pair.
func (s *Store) SetMeta(key string, val string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(metaBucket)
		return b.Put([]byte(key), []byte(val))
	})
}

// GetMeta gets a metadata value by key.
func (s *Store) GetMeta(key string) (string, error) {
	var val string
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(metaBucket)
		data := b.Get([]byte(key))
		if data != nil {
			val = string(data)
		}
		return nil
	})
	return val, err
}

// SetLastIndexTime records the timestamp of the last complete index pass.
func (s *Store) SetLastIndexTime(t time.Time) error {
	return s.SetMeta("last_index_time", t.Format(time.RFC3339))
}

// GetLastIndexTime retrieves the timestamp of the last index pass.
func (s *Store) GetLastIndexTime() (time.Time, error) {
	val, err := s.GetMeta("last_index_time")
	if err != nil || val == "" {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, val)
}
