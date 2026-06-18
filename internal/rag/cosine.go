// Package rag implements retrieval-augmented generation.
package rag

import (
	"errors"
	"math"
)

// CosineSimilarity calculates the cosine similarity between two float32 slices.
// Returns an error if the dimensions do not match, if vectors are empty,
// or if either vector has a magnitude of zero.
func CosineSimilarity(a, b []float32) (float32, error) {
	if len(a) != len(b) {
		return 0, errors.New("vector dimensions must match")
	}
	if len(a) == 0 {
		return 0, errors.New("vectors cannot be empty")
	}

	var dotProduct float32
	var normA float32
	var normB float32

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0, errors.New("cannot calculate similarity with a zero vector")
	}

	magA := float32(math.Sqrt(float64(normA)))
	magB := float32(math.Sqrt(float64(normB)))

	return dotProduct / (magA * magB), nil
}
