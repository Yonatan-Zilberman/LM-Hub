package rag

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name      string
		a         []float32
		b         []float32
		want      float32
		expectErr bool
	}{
		{
			name:      "exact match",
			a:         []float32{1.0, 0.0, 0.0},
			b:         []float32{1.0, 0.0, 0.0},
			want:      1.0,
			expectErr: false,
		},
		{
			name:      "opposite vectors",
			a:         []float32{1.0, 2.0, 3.0},
			b:         []float32{-1.0, -2.0, -3.0},
			want:      -1.0,
			expectErr: false,
		},
		{
			name:      "orthogonal vectors",
			a:         []float32{1.0, 0.0},
			b:         []float32{0.0, 1.0},
			want:      0.0,
			expectErr: false,
		},
		{
			name:      "different dimensions error",
			a:         []float32{1.0, 0.0},
			b:         []float32{1.0, 0.0, 0.0},
			want:      0.0,
			expectErr: true,
		},
		{
			name:      "empty vector error",
			a:         []float32{},
			b:         []float32{},
			want:      0.0,
			expectErr: true,
		},
		{
			name:      "zero vector error",
			a:         []float32{0.0, 0.0},
			b:         []float32{1.0, 1.0},
			want:      0.0,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CosineSimilarity(tt.a, tt.b)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.InDelta(t, tt.want, got, 1e-5)
			}
		})
	}
}
