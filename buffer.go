package main

import (
	"fmt"
	"path/filepath"
)

// Buffer represents an editor buffer with optional backing file.
type Buffer struct {
	Name      string
	Path      string
	Data      []byte
	undoStack []Action
}

// HasPath reports whether this buffer is backed by a file.
func (b *Buffer) HasPath() bool {
	return b.Path != ""
}

func NewScratchBuffer() *Buffer {
	return &Buffer{Name: "<scratch>"}
}

// CreateBuffer constructs a buffer backed by the given path and ensures the
// buffer name is unique among the provided buffers by appending a numeric
// suffix if needed.
func CreateBuffer(buffers []*Buffer, path string, data []byte) *Buffer {
	base := filepath.Base(path)
	name := base
	if hasBufferName(buffers, name) {
		for i := 1; ; i++ {
			candidate := fmt.Sprintf("%s.%d", base, i)
			if !hasBufferName(buffers, candidate) {
				name = candidate
				break
			}
		}
	}
	return &Buffer{Name: name, Path: path, Data: data}
}

func hasBufferName(buffers []*Buffer, name string) bool {
	for _, b := range buffers {
		if b.Name == name {
			return true
		}
	}
	return false
}
