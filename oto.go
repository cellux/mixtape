package main

import (
	"github.com/ebitengine/oto/v3"
)

var otoContext *oto.Context

type OtoPlayer = oto.Player

func InitOtoContext(sampleRate int) error {
	otoContextOptions := &oto.NewContextOptions{
		SampleRate:   sampleRate,
		ChannelCount: 2,
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
