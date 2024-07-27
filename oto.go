package main

import (
	"github.com/ebitengine/oto/v3"
)

var otoContext *oto.Context

func InitOtoContext() error {
	otoContextOptions := &oto.NewContextOptions{
		SampleRate:   DefaultSampleRate,
		ChannelCount: AudioChannelCount,
		Format:       oto.FormatFloat32LE,
		BufferSize:   0,
	}
	ctx, readyChan, err := oto.NewContext(otoContextOptions)
	if err != nil {
		return err
	}
	<-readyChan
	otoContext = ctx
	return nil
}
