package tools

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// UndoStack manages the list of undoable operations in a thread-safe manner.
type UndoStack struct {
	mu      sync.Mutex
	records []UndoRecord
}

// NewUndoStack creates a new initialized UndoStack.
func NewUndoStack() *UndoStack {
	return &UndoStack{
		records: make([]UndoRecord, 0),
	}
}

// Push adds an UndoRecord to the top of the stack.
func (s *UndoStack) Push(record UndoRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, record)
}

// Pop removes and returns the top UndoRecord from the stack.
func (s *UndoStack) Pop() (UndoRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.records) == 0 {
		return UndoRecord{}, false
	}

	idx := len(s.records) - 1
	record := s.records[idx]
	s.records = s.records[:idx]
	return record, true
}

// Peek returns the top UndoRecord from the stack without removing it.
func (s *UndoStack) Peek() (UndoRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.records) == 0 {
		return UndoRecord{}, false
	}

	return s.records[len(s.records)-1], true
}

// List returns a copy of all records in the stack, ordered newest to oldest.
func (s *UndoStack) List() []UndoRecord {
	s.mu.Lock()
	defer s.mu.Unlock()

	copyRecords := make([]UndoRecord, len(s.records))
	for i, r := range s.records {
		// Reverse order
		copyRecords[len(s.records)-1-i] = r
	}
	return copyRecords
}

// Len returns the number of records in the stack.
func (s *UndoStack) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.records)
}

// UndoLast pops the top record and executes its inverse operation using the registry.
func (s *UndoStack) UndoLast(ctx context.Context, r *Registry) error {
	record, ok := s.Pop()
	if !ok {
		return errors.New("no actions to undo")
	}

	if record.InverseOp == "" {
		return fmt.Errorf("operation %s does not support rollback", record.ToolName)
	}

	res, err := r.Execute(ctx, record.InverseOp, record.InverseArgs)
	if err != nil {
		return fmt.Errorf("failed to run rollback command %s: %w", record.InverseOp, err)
	}

	if res.IsError {
		return fmt.Errorf("rollback command returned error: %s", res.Content)
	}

	return nil
}

// UndoAll rolls back all actions in the stack from newest to oldest.
func (s *UndoStack) UndoAll(ctx context.Context, r *Registry) error {
	for {
		s.mu.Lock()
		empty := len(s.records) == 0
		s.mu.Unlock()

		if empty {
			break
		}

		if err := s.UndoLast(ctx, r); err != nil {
			return err
		}
	}
	return nil
}
