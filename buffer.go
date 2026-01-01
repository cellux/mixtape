package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

// Buffer represents an editor buffer with optional backing file.
type Buffer struct {
	Name        string
	Path        string
	Data        []byte
	Dirty       bool
	undoStack   []Action
	editorPoint EditorPoint
	editorTop   int
	editorLeft  int
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

func (b *Buffer) WriteFile() error {
	if b.HasPath() {
		if err := os.WriteFile(b.Path, b.Data, 0o644); err != nil {
			return err
		} else {
			b.MarkClean()
		}
	}
	return nil
}

// Clean reports whether the buffer is not dirty.
func (b *Buffer) Clean() bool {
	return !b.Dirty
}

// MarkClean clears the dirty flag.
func (b *Buffer) MarkClean() {
	b.Dirty = false
}

// SetPath updates the buffer path.
func (b *Buffer) SetPath(path string) {
	b.Path = path
	b.Dirty = true
}

type BufferManager struct {
	buffers       []*Buffer
	currentBuffer *Buffer
}

func CreateBufferManager() *BufferManager {
	return &BufferManager{}
}

// CreateBuffer constructs a buffer with the given name, path and
// data, adds it to the set of buffers managed by this manager and
// makes it current.
//
// Ensures that the buffer name is unique among managed buffers,
// appending a numeric suffix if needed.
//
// If name is empty, it is set to the basename of path.
//
// If both name and path are empty, name is set to "<scratch>".
func (bm *BufferManager) CreateBuffer(name string, path string, data []byte) *Buffer {
	if name == "" {
		if path != "" {
			base := filepath.Base(path)
			name = base
		} else {
			name = "<scratch>"
		}
	}
	if bm.findBufferByName(name) != nil {
		for i := 1; ; i++ {
			candidate := fmt.Sprintf("%s.%d", name, i)
			if bm.findBufferByName(candidate) == nil {
				name = candidate
				break
			}
		}
	}
	buf := &Buffer{Name: name, Path: path, Data: data}
	bm.buffers = append(bm.buffers, buf)
	bm.currentBuffer = buf
	return buf
}

func (bm *BufferManager) Empty() bool {
	return len(bm.buffers) == 0
}

func (bm *BufferManager) GetCurrentBuffer() *Buffer {
	return bm.currentBuffer
}

func (bm *BufferManager) SetCurrentBuffer(b *Buffer) {
	bm.currentBuffer = b
}

func (bm *BufferManager) findBufferByName(name string) *Buffer {
	for _, b := range bm.buffers {
		if b.Name == name {
			return b
		}
	}
	return nil
}

func (bm *BufferManager) findBufferByPath(path string) *Buffer {
	for _, b := range bm.buffers {
		if b.Path == path {
			return b
		}
	}
	return nil
}

func (bm *BufferManager) getAdjacentBuffer(delta int) *Buffer {
	n := len(bm.buffers)
	if n < 2 {
		return nil
	}
	currentIndex := -1
	currentBuffer := bm.currentBuffer
	for i, buf := range bm.buffers {
		if buf == currentBuffer {
			currentIndex = i
			break
		}
	}
	if currentIndex == -1 {
		return nil
	}
	nextIndex := (currentIndex + delta + n) % n
	adjacentBuffer := bm.buffers[nextIndex]
	if adjacentBuffer == currentBuffer {
		return nil
	}
	return adjacentBuffer
}

func (bm *BufferManager) RemoveBuffer(target *Buffer) {
	for i, b := range bm.buffers {
		if b == target {
			bm.buffers = append(bm.buffers[:i], bm.buffers[i+1:]...)
			break
		}
	}
}

func (bm *BufferManager) FirstBuffer() *Buffer {
	if len(bm.buffers) == 0 {
		return nil
	}
	return bm.buffers[0]
}
