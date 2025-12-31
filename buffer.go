package main

import (
	"bytes"
	"fmt"
	"path/filepath"
)

// Buffer represents an editor buffer with optional backing file.
type Buffer struct {
	Name      string
	Path      string
	Data      []byte
	Dirty     bool
	undoStack []Action
}

// SetData replaces the buffer contents and marks it dirty if changed.
func (b *Buffer) SetData(data []byte) {
	if bytes.Equal(b.Data, data) {
		return
	}
	b.Data = data
	b.Dirty = true
}

// HasPath reports whether this buffer is backed by a file.
func (b *Buffer) HasPath() bool {
	return b.Path != ""
}

func NewScratchBuffer() *Buffer {
	return &Buffer{Name: "<scratch>"}
}

// Clean reports whether the buffer is not dirty.
func (b *Buffer) Clean() bool {
	return !b.Dirty
}

// MarkClean clears the dirty flag.
func (b *Buffer) MarkClean() {
	b.Dirty = false
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

// SetPath updates the buffer path.
func (b *Buffer) SetPath(path string) {
	b.Path = path
	b.Dirty = true
}

func hasBufferName(buffers []*Buffer, name string) bool {
	for _, b := range buffers {
		if b.Name == name {
			return true
		}
	}
	return false
}
