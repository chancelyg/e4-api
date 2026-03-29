package pkg

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEmbeddedFiles(t *testing.T) {
	// Test that index.html exists in embedded filesystem
	content, err := WebFS.Open("dist/index.html")
	require.NoError(t, err, "index.html should exist in embedded filesystem")
	defer content.Close()

	// Read content and verify it's HTML
	data, err := io.ReadAll(content)
	require.NoError(t, err)

	html := string(data)
	assert.Contains(t, html, "<!doctype html>", "Should be valid HTML")
	assert.Contains(t, html, "E4 Diary", "Should contain correct title")
}

func TestEmbeddedAssets(t *testing.T) {
	// Test that _app directory exists (contains JS/CSS assets)
	dir, err := WebFS.Open("dist/_app")
	require.NoError(t, err, "_app directory should exist")
	defer dir.Close()

	// Check if it's a directory
	stat, err := dir.Stat()
	require.NoError(t, err)
	assert.True(t, stat.IsDir(), "_app should be a directory")
}

func TestEmbeddedCSS(t *testing.T) {
	// Try to find CSS files in _app directory
	dir, err := WebFS.ReadDir("dist/_app/immutable/assets")
	if err != nil {
		t.Skip("CSS assets directory not found, skipping")
		return
	}

	foundCSS := false
	for _, entry := range dir {
		if !entry.IsDir() {
			name := entry.Name()
			if len(name) > 4 && name[len(name)-4:] == ".css" {
				foundCSS = true
				break
			}
		}
	}

	assert.True(t, foundCSS, "Should find at least one CSS file")
}
