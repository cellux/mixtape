package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileBrowserEscapeClearsFilterBeforeExiting(t *testing.T) {
	exitCount := 0
	browser, err := CreateFileBrowser(t.TempDir(), nil, FileBrowserCallbacks{
		onExit: func() { exitCount++ },
	})
	if err != nil {
		t.Fatal(err)
	}

	browser.OnChar('a')
	browser.HandleKey("Escape")
	if browser.SearchText() != "" {
		t.Fatalf("Escape left filter active: %q", browser.SearchText())
	}
	if browser.listDisplay.FilterMode() {
		t.Fatal("Escape did not leave filter mode")
	}
	if exitCount != 0 {
		t.Fatalf("Escape exited while clearing a filter: %d exits", exitCount)
	}

	browser.HandleKey("Escape")
	if exitCount != 1 {
		t.Fatalf("Escape did not exit without a filter: %d exits", exitCount)
	}
}

func TestBufferBrowserEscapeClearsFilterBeforeExiting(t *testing.T) {
	bm := CreateBufferManager()
	bm.CreateBuffer("scratch", "", nil)
	exitCount := 0
	browser := CreateBufferBrowser(bm, BufferBrowserCallbacks{
		onExit: func() { exitCount++ },
	})

	browser.OnChar('s')
	browser.HandleKey("Escape")
	if browser.SearchText() != "" {
		t.Fatalf("Escape left filter active: %q", browser.SearchText())
	}
	if browser.listDisplay.FilterMode() {
		t.Fatal("Escape did not leave filter mode")
	}
	if exitCount != 0 {
		t.Fatalf("Escape exited while clearing a filter: %d exits", exitCount)
	}

	browser.HandleKey("Escape")
	if exitCount != 1 {
		t.Fatalf("Escape did not exit without a filter: %d exits", exitCount)
	}
}

func TestFileBrowserBackspaceKeepsEmptyFilterMode(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "child")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	browser, err := CreateFileBrowser(dir, nil, FileBrowserCallbacks{})
	if err != nil {
		t.Fatal(err)
	}

	browser.OnChar('a')
	browser.HandleKey("Backspace")
	if browser.SearchText() != "" {
		t.Fatalf("Backspace left filter text: %q", browser.SearchText())
	}
	if !browser.listDisplay.FilterMode() {
		t.Fatal("removing the last character left filter mode")
	}
	if browser.Directory() != dir {
		t.Fatalf("Backspace changed directory while filtering: %q", browser.Directory())
	}

	browser.HandleKey("Backspace")
	if browser.Directory() != dir {
		t.Fatalf("Backspace changed directory while empty filter mode was active: %q", browser.Directory())
	}

	browser.HandleKey("Escape")
	if browser.listDisplay.FilterMode() {
		t.Fatal("Escape did not leave filter mode")
	}
	browser.HandleKey("Backspace")
	if browser.Directory() != parent {
		t.Fatalf("Backspace did not navigate to parent after leaving filter mode: %q", browser.Directory())
	}
}

func TestBufferBrowserBackspaceKeepsEmptyFilterMode(t *testing.T) {
	bm := CreateBufferManager()
	bm.CreateBuffer("scratch", "", nil)
	browser := CreateBufferBrowser(bm, BufferBrowserCallbacks{})

	browser.OnChar('s')
	browser.HandleKey("Backspace")
	if browser.SearchText() != "" {
		t.Fatalf("Backspace left filter text: %q", browser.SearchText())
	}
	if !browser.listDisplay.FilterMode() {
		t.Fatal("removing the last character left filter mode")
	}

	browser.HandleKey("Escape")
	if browser.listDisplay.FilterMode() {
		t.Fatal("Escape did not leave filter mode")
	}
}
