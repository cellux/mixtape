package main

import "fmt"

// BufferEntry adapts Buffer to the ListEntry interface.
type BufferEntry struct {
	buffer *Buffer
}

func (be BufferEntry) GetUniqueId() any {
	return be.buffer
}

func (be BufferEntry) Format() string {
	path := be.buffer.Path
	if path == "" {
		path = "(scratch)"
	}
	return fmt.Sprintf("%-20s %s", be.buffer.Name, path)
}

type BufferBrowserCallbacks struct {
	onSelect func(*Buffer)
	onExit   func()
}

// BufferBrowser provides a searchable list of buffers.
type BufferBrowser struct {
	bm *BufferManager
	*ListBrowser
	callbacks BufferBrowserCallbacks
}

func CreateBufferBrowser(bm *BufferManager, callbacks BufferBrowserCallbacks) *BufferBrowser {
	bb := &BufferBrowser{
		bm:        bm,
		callbacks: callbacks,
	}
	bb.ListBrowser = CreateListBrowser(func() string { return "Buffers" }, ListBrowserCallbacks{
		onEnter: bb.handleEnter,
		onExit:  bb.Exit,
	})
	bb.Reload()
	return bb
}

func (bb *BufferBrowser) Reload() {
	bm := bb.bm
	entries := make([]ListEntry, len(bm.buffers))
	for i, buf := range bm.buffers {
		entries[i] = BufferEntry{buffer: buf}
	}
	bb.ListBrowser.SetEntries(entries)
	if bm.currentBuffer != nil {
		_ = bb.ListBrowser.SelectById(bm.currentBuffer)
	}
}

func (bb *BufferBrowser) CurrentFilteredEntry() *Buffer {
	filtered := bb.ListBrowser.GetFilteredEntries()
	if len(filtered) == 0 {
		return nil
	}
	idx := bb.ListBrowser.GetFilteredSelectionIndex()
	be := filtered[idx].(BufferEntry)
	return be.buffer
}

func (bb *BufferBrowser) Reset() {
	bb.ListBrowser.Reset()
	bb.Reload()
}

func (bb *BufferBrowser) Exit() {
	if bb.callbacks.onExit != nil {
		bb.callbacks.onExit()
	}
}

func (bb *BufferBrowser) handleEnter() {
	buf := bb.CurrentFilteredEntry()
	if buf == nil {
		return
	}
	if bb.callbacks.onSelect != nil {
		bb.callbacks.onSelect(buf)
	}
}
