package services

import (
	"testing"
)

func TestFormatDocumentNumber(t *testing.T) {
	tests := []struct {
		name     string
		index    string
		number   int
		expected string
	}{
		{
			name:     "simple case",
			index:    "01-02",
			number:   123,
			expected: "01-02/123",
		},
		{
			name:     "empty index",
			index:    "",
			number:   456,
			expected: "/456",
		},
		{
			name:     "zero number",
			index:    "ABC",
			number:   0,
			expected: "ABC/0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatDocumentNumber(tt.index, tt.number)
			if result != tt.expected {
				t.Errorf("formatDocumentNumber(%q, %d) = %q; want %q", tt.index, tt.number, result, tt.expected)
			}
		})
	}
}
