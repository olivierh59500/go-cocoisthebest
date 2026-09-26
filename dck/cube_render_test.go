//go:build dck_cube_rendercheck

package coco

import (
	"bytes"
	"fmt"
	"image/color"
	"math"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	kit "github.com/olivierh59500/democonstructionkit"
	"github.com/olivierh59500/democonstructionkit/effects"
	capture "github.com/olivierh59500/democonstructionkit/fidelity/ebiten"
	"github.com/olivierh59500/democonstructionkit/presets"
	"github.com/olivierh59500/democonstructionkit/render"
)

var solidCubeCheckFrames = []int{0, 1, 17, 90, 301, 1023, 1024, 1025, 2047, 2048}

type solidCubeRenderCheck struct {
	train                   *effects.SolidCubeTrain
	old                     [12]legacyCocoCube
	phases                  [12]float64
	actual, expected, white *ebiten.Image
	pixelsA, pixelsB        []byte
	frame, checked          int
	err                     error
}

func (c *solidCubeRenderCheck) Layout(int, int) (int, int) { return 800, 600 }
func (c *solidCubeRenderCheck) Update() error {
	if c.err != nil {
		return c.err
	}
	c.frame++
	if err := c.train.Update(kit.Frame{}); err != nil {
		return err
	}
	for i := range c.old {
		c.phases[i] += .04
		dx, dy, dz := .02*(1+float64(i)*.1), .03*(1+float64(i)*.15), .01*(1+float64(i)*.05)
		c.old[i].Rotate(dx, dy, dz)
	}
	return nil
}
func (c *solidCubeRenderCheck) Draw(dst *ebiten.Image) {
	if c.err != nil || c.checked >= len(solidCubeCheckFrames) || c.frame != solidCubeCheckFrames[c.checked] {
		return
	}
	c.actual.Clear()
	c.expected.Clear()
	for i := range c.old {
		phase := c.phases[i]
		x, y := 380+380*math.Sin(phase), 300+84*math.Cos(phase*2.5)
		c.old[i].Draw(c.expected, c.white, x, y)
	}

	c.train.Draw(c.actual)
	c.actual.ReadPixels(c.pixelsA)
	c.expected.ReadPixels(c.pixelsB)
	if !bytes.Equal(c.pixelsA, c.pixelsB) {
		c.err = fmt.Errorf("shared cube train pixels differ at frame %d", c.frame)
	}
	dst.DrawImage(c.actual, nil)
	c.checked++
}
func TestMain(m *testing.M) {
	if code := m.Run(); code != 0 {
		os.Exit(code)
	}
	dir, err := os.MkdirTemp("", "coco-cubes-")
	if err != nil {
		panic(err)
	}
	var c *solidCubeRenderCheck
	err = capture.Run(capture.Config{Directory: dir, Frames: solidCubeCheckFrames, Width: 800, Height: 600}, func() (ebiten.Game, error) {
		train, err := effects.NewSolidCubeTrain(presets.CocoCubeTrain(800, 600, 40, 12))
		if err != nil {
			return nil, err
		}
		c = &solidCubeRenderCheck{train: train, actual: render.NewSurface(800, 600), expected: render.NewSurface(800, 600), white: ebiten.NewImage(3, 3), pixelsA: make([]byte, 800*600*4), pixelsB: make([]byte, 800*600*4)}
		c.white.Fill(color.White)
		for i := range c.old {
			c.old[i] = legacyCocoCube{size: 40, angleX: float64(i) * .3, angleY: float64(i) * .2, angleZ: float64(i) * .1}
			c.phases[i] = .15 * float64(i+1)
		}
		return c, nil
	})
	if err == nil {
		err = c.err
		if err == nil && c.checked != len(solidCubeCheckFrames) {
			err = fmt.Errorf("only %d captures checked", c.checked)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("All %d shared cube train captures match exactly: %s\n", c.checked, dir)
}
