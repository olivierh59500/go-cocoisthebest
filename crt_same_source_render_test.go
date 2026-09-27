//go:build coco_crt_same_sourcecheck

package coco

// This opt-in GPU check gives the preserved and shared CRT shaders one source
// texture and matching output surfaces, isolating shader math from allocation.

import (
	"fmt"
	"image"
	"image/color"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/democonstructionkit/effects"
	capture "github.com/olivierh59500/democonstructionkit/fidelity/ebiten"
	"github.com/olivierh59500/democonstructionkit/presets"
)

type crtSameSourceCheck struct {
	*Game
	shared *effects.CRTOverlay
	tick   int
	counts map[int]int
}

func (c *crtSameSourceCheck) Update() error { c.tick++; return c.Game.Update() }
func (c *crtSameSourceCheck) Draw(dst *ebiten.Image) {
	c.Game.Draw(dst)
	if c.tick != 0 && c.tick != 240 {
		return
	}
	options := &ebiten.NewImageOptions{Unmanaged: true}
	reference := ebiten.NewImageWithOptions(image.Rect(0, 0, screenWidth, screenHeight), options)
	mutualized := ebiten.NewImageWithOptions(image.Rect(0, 0, screenWidth, screenHeight), options)
	reference.Fill(color.Black)
	mutualized.Fill(color.Black)
	y := float64(screenHeight/2 - int(fontHeight*2)/2)
	var originalOptions ebiten.DrawRectShaderOptions
	originalOptions.Images[0] = c.introStrip
	originalOptions.GeoM.Translate(0, y)
	reference.DrawRectShader(screenWidth, int(fontHeight*2), c.crtShader, &originalOptions)
	c.shared.DrawAt(mutualized, c.introStrip, 0, y)
	first, second := make([]byte, screenWidth*screenHeight*4), make([]byte, screenWidth*screenHeight*4)
	reference.ReadPixels(first)
	mutualized.ReadPixels(second)
	different := 0
	for i := 0; i < len(first); i += 4 {
		if first[i] != second[i] || first[i+1] != second[i+1] || first[i+2] != second[i+2] || first[i+3] != second[i+3] {
			different++
		}
	}
	c.counts[c.tick] = different
	reference.Deallocate()
	mutualized.Deallocate()
}

func TestMain(m *testing.M) {
	if code := m.Run(); code != 0 {
		os.Exit(code)
	}
	directory, err := os.MkdirTemp("", "coco-crt-same-source-")
	if err != nil {
		panic(err)
	}
	var check *crtSameSourceCheck
	err = capture.Run(capture.Config{Directory: directory, Frames: []int{0, 240}, Width: screenWidth, Height: screenHeight}, func() (ebiten.Game, error) {
		game := NewGame()
		game.audioReady = true
		shared, err := effects.NewCRTOverlay(presets.DMACRTOverlay())
		if err != nil {
			return nil, err
		}
		check = &crtSameSourceCheck{Game: game, shared: shared, counts: map[int]int{}}
		return check, nil
	})
	if check != nil {
		check.shared.Close()
		fmt.Printf("CRT same-source pixel differences: tick 0=%d, tick 240=%d\n", check.counts[0], check.counts[240])
		if err == nil && (check.counts[0] != 0 || check.counts[240] != 0) {
			err = fmt.Errorf("shared CRT output differs from the preserved shader for one source image")
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
