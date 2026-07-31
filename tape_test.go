package main

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/dh1tw/gosamplerate"
)

func TestCancelableReadSeekerStopsAfterCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	reader := cancelableReadSeeker{
		Context:    ctx,
		ReadSeeker: bytes.NewReader([]byte("sample")),
	}
	cancel()

	buf := make([]byte, 1)
	if _, err := reader.Read(buf); !errors.Is(err, context.Canceled) {
		t.Fatalf("Read error = %v, want context.Canceled", err)
	}
	if _, err := reader.Seek(0, 0); !errors.Is(err, context.Canceled) {
		t.Fatalf("Seek error = %v, want context.Canceled", err)
	}
}

func TestResampleWithProgressReportsEachChunk(t *testing.T) {
	data := make([]float32, decodeChunkSamples*2)
	var progress []float64
	result, err := resampleWithProgress(context.Background(), data, 48000.0/44100.0, 2, gosamplerate.SRC_SINC_FASTEST, func(value float64) {
		progress = append(progress, value)
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result) == 0 {
		t.Fatal("resampling produced no data")
	}
	if len(progress) != 2 || progress[0] <= 0 || progress[0] >= 1 || progress[1] != 1 {
		t.Fatalf("unexpected resampling progress: %v", progress)
	}
}
