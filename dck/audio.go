package coco

import (
	"fmt"
	"io"
	"sync"

	"github.com/olivierh59500/ym-player/pkg/stsound"
)

// YMPlayer adapts the mono YM synthesizer to Ebitengine's stereo PCM stream.
type YMPlayer struct {
	player     *stsound.StSound
	sampleRate int
	buffer     []int16
	mutex      sync.Mutex
	position   int64 // byte offset in the stereo 16-bit PCM stream
	totalBytes int64
	loop       bool
	volume     float64
}

// NewYMPlayer creates a new YM player instance.
func NewYMPlayer(data []byte, sampleRate int, loop bool) (*YMPlayer, error) {
	player := stsound.CreateWithRate(sampleRate)

	if err := player.LoadMemory(data); err != nil {
		player.Destroy()
		return nil, fmt.Errorf("failed to load YM data: %w", err)
	}

	player.SetLoopMode(loop)
	info := player.GetInfo()
	totalSamples := int64(info.MusicTimeInMs) * int64(sampleRate) / 1000

	return &YMPlayer{
		player:     player,
		sampleRate: sampleRate,
		buffer:     make([]int16, 4096),
		totalBytes: totalSamples * 4,
		loop:       loop,
		volume:     0.7,
	}, nil
}

func (y *YMPlayer) Read(p []byte) (n int, err error) {
	y.mutex.Lock()
	defer y.mutex.Unlock()

	if y.player == nil {
		return 0, io.ErrClosedPipe
	}

	samplesNeeded := len(p) / 4
	processed := 0
	for processed < samplesNeeded {
		chunkSize := min(samplesNeeded-processed, len(y.buffer))

		if !y.player.Compute(y.buffer[:chunkSize], chunkSize) && !y.loop {
			clear(p[processed*4 : samplesNeeded*4])
			return samplesNeeded * 4, io.EOF
		}

		for i := 0; i < chunkSize; i++ {
			sample := int16(float64(y.buffer[i]) * y.volume)
			offset := (processed + i) * 4
			p[offset] = byte(sample)
			p[offset+1] = byte(sample >> 8)
			p[offset+2] = byte(sample)
			p[offset+3] = byte(sample >> 8)
		}

		processed += chunkSize
		y.position += int64(chunkSize * 4)
	}

	return samplesNeeded * 4, nil
}

func (y *YMPlayer) Seek(offset int64, whence int) (int64, error) {
	y.mutex.Lock()
	defer y.mutex.Unlock()

	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = y.position + offset
	case io.SeekEnd:
		newPos = y.totalBytes + offset
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}

	newPos = max(0, min(newPos, y.totalBytes))
	newPos = newPos / 4 * 4

	if y.player == nil {
		return 0, io.ErrClosedPipe
	}
	if !y.player.IsSeekable() {
		if newPos != 0 {
			return y.position, fmt.Errorf("YM stream is not seekable")
		}
		y.player.Restart()
	} else {
		timeInMs := (newPos / 4) * 1000 / int64(y.sampleRate)
		y.player.Seek(uint32(timeInMs))
	}

	y.position = newPos
	return newPos, nil
}

func (y *YMPlayer) Close() error {
	y.mutex.Lock()
	defer y.mutex.Unlock()

	if y.player != nil {
		y.player.Destroy()
		y.player = nil
	}
	return nil
}

func (y *YMPlayer) GetVolume() float64 {
	y.mutex.Lock()
	defer y.mutex.Unlock()
	return y.volume
}

func (y *YMPlayer) SetVolume(volume float64) {
	y.mutex.Lock()
	defer y.mutex.Unlock()
	y.volume = max(0, min(volume, 1))
}
