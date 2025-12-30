package main

// Buffer represents an editor buffer with optional backing file.
type Buffer struct {
	Name string
	Path string
	Data []byte
}

// HasPath reports whether this buffer is backed by a file.
func (b *Buffer) HasPath() bool {
	return b.Path != ""
}

func NewScratchBuffer() *Buffer {
	return &Buffer{Name: "<scratch>"}
}
