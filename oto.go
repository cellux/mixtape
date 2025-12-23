package main

import (
	"github.com/ebitengine/oto/v3"
	"sync"
)

type TapePlayer struct {
	reader *TapeReader
	player *oto.Player
}

func (tp *TapePlayer) GetCurrentFrame() int {
	numBytesStillInOtoBuffer := tp.player.BufferedSize()
	return tp.reader.GetCurrentFrame(numBytesStillInOtoBuffer)
}

type OtoState struct {
	mu          sync.Mutex
	ctx         *oto.Context
	tapePlayers []*TapePlayer
}

func NewOtoState(sampleRate int) (*OtoState, error) {
	otoContextOptions := &oto.NewContextOptions{
		SampleRate:   sampleRate,
		ChannelCount: 2,
		Format:       oto.FormatFloat32LE,
		BufferSize:   0,
	}
	ctx, readyChan, err := oto.NewContext(otoContextOptions)
	if err != nil {
		return nil, err
	}
	<-readyChan
	otoState := &OtoState{
		ctx: ctx,
	}
	return otoState, nil
}

func (os *OtoState) GetTapePlayers() []*TapePlayer {
	os.mu.Lock()
	result := os.tapePlayers[:]
	os.mu.Unlock()
	return result
}

func (os *OtoState) PlayTape(x any) {
	if streamable, ok := x.(Streamable); ok {
		stream := streamable.Stream()
		if stream.nframes > 0 {
			tape := stream.Take(nil, stream.nframes)
			reader := MakeTapeReader(tape, 2)
			player := os.ctx.NewPlayer(reader)
			tapePlayer := &TapePlayer{
				reader: reader,
				player: player,
			}
			os.mu.Lock()
			os.tapePlayers = append(os.tapePlayers, tapePlayer)
			os.mu.Unlock()
			player.Play()
		}
	}
}

func (os *OtoState) StopAllPlayers() {
	os.mu.Lock()
	defer os.mu.Unlock()
	for _, tp := range os.tapePlayers {
		if tp.player.IsPlaying() {
			tp.player.Pause()
		}
	}
	os.tapePlayers = nil
}
