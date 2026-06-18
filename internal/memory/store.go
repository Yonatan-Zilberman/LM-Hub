// Package memory implements persistent agent memory.
package memory

import (
	"encoding/json"
	"fmt"
	"time"

	"go.etcd.io/bbolt"
)

// MemoryFact represents a piece of knowledge stored by the agent.
type MemoryFact struct {
	ID         string    `json:"id"`
	Scope      string    `json:"scope"` // "project:{path}" or "global"
	Content    string    `json:"content"`
	Source     string    `json:"source"` // "user" | "extracted"
	Confidence float64   `json:"confidence"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsed   time.Time `json:"last_used"`
	UseCount   int       `json:"use_count"`
}

// Store wraps bbolt.DB for memory facts.
type Store struct {
	db *bbolt.DB
}

var factsBucket = []byte("facts")

// NewStore initializes a Store with the provided bbolt database.
func NewStore(db *bbolt.DB) (*Store, error) {
	err := db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(factsBucket)
		if err != nil {
			return fmt.Errorf("failed to create facts bucket: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize memory database buckets: %w", err)
	}
	return &Store{db: db}, nil
}

// Put stores or updates a memory fact.
func (s *Store) Put(fact *MemoryFact) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(factsBucket)
		data, err := json.Marshal(fact)
		if err != nil {
			return fmt.Errorf("failed to marshal memory fact: %w", err)
		}
		err = b.Put([]byte(fact.ID), data)
		if err != nil {
			return fmt.Errorf("failed to write memory fact %s: %w", fact.ID, err)
		}
		return nil
	})
}

// Get retrieves a memory fact by its ID.
func (s *Store) Get(id string) (*MemoryFact, error) {
	var fact MemoryFact
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(factsBucket)
		data := b.Get([]byte(id))
		if data == nil {
			return fmt.Errorf("memory fact not found: %s", id)
		}
		if err := json.Unmarshal(data, &fact); err != nil {
			return fmt.Errorf("failed to unmarshal memory fact: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &fact, nil
}

// Delete removes a memory fact by ID.
func (s *Store) Delete(id string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(factsBucket)
		return b.Delete([]byte(id))
	})
}

// ListByScope retrieves all facts that match a given scope prefix (e.g., "project:").
func (s *Store) ListByScope(scope string) ([]*MemoryFact, error) {
	var facts []*MemoryFact
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(factsBucket)
		return b.ForEach(func(k, v []byte) error {
			var fact MemoryFact
			if err := json.Unmarshal(v, &fact); err != nil {
				return fmt.Errorf("failed to unmarshal memory fact: %w", err)
			}
			if fact.Scope == scope {
				facts = append(facts, &fact)
			}
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return facts, nil
}

// DeleteByScope deletes all facts that match a given scope.
func (s *Store) DeleteByScope(scope string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(factsBucket)
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			var fact MemoryFact
			if err := json.Unmarshal(v, &fact); err != nil {
				continue
			}
			if fact.Scope == scope {
				if err := b.Delete(k); err != nil {
					return fmt.Errorf("failed to delete fact %s: %w", string(k), err)
				}
			}
		}
		return nil
	})
}
