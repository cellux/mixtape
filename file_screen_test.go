package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileScreenEnterTapeFileOpensItInEditor(t *testing.T) {
	path := filepath.Join(t.TempDir(), "patch.tape")
	data := []byte("440 f 1s take")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	bm := CreateBufferManager()
	bm.CreateBuffer("", "", nil)
	app := &App{bm: bm}
	editor := &EditScreen{app: app, bm: bm, editor: CreateEditor()}
	fileScreen := &FileScreen{app: app}
	app.screens = map[string]Screen{"edit": editor}

	fileScreen.handleFileBrowserSelection(FileEntry{name: "patch.tape", path: path})

	if app.currentScreen != editor {
		t.Fatal("opening a tape file did not select the editor screen")
	}
	buffer := bm.GetCurrentBuffer()
	if buffer.Path != path {
		t.Fatalf("opened buffer path = %q, want %q", buffer.Path, path)
	}
	if string(buffer.Data) != string(data) {
		t.Fatalf("opened buffer data = %q, want %q", buffer.Data, data)
	}
}
