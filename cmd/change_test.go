package cmd

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestReadFile(t *testing.T) {
	var expectedString = "this is a simple file, we need to be able to read it correctly"
	t.Run("ReadFile", func(t *testing.T) {
		err, stringBytes, stringContents := ReadFile("test_ReadFile.txt")
		assert.Equal(t, nil, err)
		assert.Equal(t, []byte(expectedString), stringBytes)
		assert.Equal(t, expectedString, stringContents)
		assert.Equal(t, 1, 1)
	})
}
func TestMatchGlob(t *testing.T) {
	var tests = []struct {
		name     string
		glob     string
		input    string
		expected bool
	}{
		{"develop-match", "develop-*", "develop-123", true},
		{"develop-miss", "develop-*", "other-123", false},
		{"develop-v1-match", "develop-v1.*.*", "develop-v1.2.3", true},
		{"develop-v1-miss", "develop-v1.*.*", "develop-v1.2", false},
	}
	//var expectedString = "this is a simple file, we need to be able to read it correctly"
	t.Run("MatchGlob", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got := MatchGlob(tt.input, tt.glob)
				assert.Equal(t, tt.expected, got)
			})
		}
	})
}
