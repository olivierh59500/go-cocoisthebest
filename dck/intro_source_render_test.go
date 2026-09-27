//go:build coco_intro_sourcecheck

package coco

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	capture "github.com/olivierh59500/democonstructionkit/fidelity/ebiten"
)

type introSourceCapture struct {
	*Game
	tick      int
	directory string
}

func (c *introSourceCapture) Update() error {
	c.tick++
	return c.Game.Update()
}

func (c *introSourceCapture) Draw(dst *ebiten.Image) {
	c.Game.Draw(dst)
	if c.tick != 0 && c.tick != 1 && c.tick != 60 && c.tick != 240 && c.tick != 241 {
		return
	}
	pixels := image.NewRGBA(image.Rect(0, 0, screenWidth, int(fontHeight*2)))
	c.introStrip.ReadPixels(pixels.Pix)
	file, err := os.Create(filepath.Join(c.directory, fmt.Sprintf("intro-%06d.png", c.tick)))
	if err == nil {
		err = png.Encode(file, pixels)
		closeErr := file.Close()
		if err == nil {
			err = closeErr
		}
	}
	if err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	if code := m.Run(); code != 0 {
		os.Exit(code)
	}
	directory := os.Getenv("COCO_INTRO_SOURCE_CAPTURES")
	if directory == "" {
		var err error
		directory, err = os.MkdirTemp("", "coco-intro-source-")
		if err != nil {
			panic(err)
		}
	}
	if err := os.MkdirAll(directory, 0755); err != nil {
		panic(err)
	}
	err := capture.Run(capture.Config{Directory: directory, Frames: []int{0, 1, 60, 240, 241}, Width: screenWidth, Height: screenHeight}, func() (ebiten.Game, error) {
		game := NewGameWithOptions(GameOptions{NormalizeIntroCRT: os.Getenv("COCO_NORMALIZE_CRT") == "1"})
		game.audioReady = true
		return &introSourceCapture{Game: game, directory: directory}, nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
