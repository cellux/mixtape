package main

import (
	"github.com/ebitengine/oto/v3"
	"sync"
)

type TapePlayer struct {
	reader *TapeReader
	player *oto.Player
	owner  Screen
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

func (os *OtoState) GetTapePlayers(owner Screen) []*TapePlayer {
	os.mu.Lock()
	result := make([]*TapePlayer, 0, len(os.tapePlayers))
	for _, tp := range os.tapePlayers {
		if tp.owner == owner {
			result = append(result, tp)
		}
	}
	os.mu.Unlock()
	return result
}

func (os *OtoState) PlayTape(x any, owner Screen) {
	if streamable, ok := x.(Streamable); ok {
		stream := streamable.Stream()
		if stream.nframes > 0 {
			tape := stream.Take(nil, stream.nframes)
			reader := MakeTapeReader(tape, 2)
			player := os.ctx.NewPlayer(reader)
			tapePlayer := &TapePlayer{
				reader: reader,
				player: player,
				owner:  owner,
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
