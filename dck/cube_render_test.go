//go:build dck_cube_rendercheck

package coco

import (
	"bytes"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/democonstructionkit/effects"
	capture "github.com/olivierh59500/democonstructionkit/fidelity/ebiten"
	"github.com/olivierh59500/democonstructionkit/geometry"
	"github.com/olivierh59500/democonstructionkit/presets"
	"github.com/olivierh59500/democonstructionkit/render"
	"image/color"
	"math"
	"os"
	"testing"
)

var solidCubeCheckFrames = []int{0, 1, 17, 90, 301, 1023, 1024, 1025, 2047, 2048}

type solidCubeRenderCheck struct {
	cubes                   [12]*effects.SolidCube
	old                     [12]legacyCocoCube
	batch                   *effects.SolidCubeBatch
	actual, expected, white *ebiten.Image
	vertices                []ebiten.Vertex
	indices                 []uint16
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
	for i := range c.cubes {
		dx, dy, dz := .02*(1+float64(i)*.1), .03*(1+float64(i)*.15), .01*(1+float64(i)*.05)
		c.cubes[i].Rotate(dx, dy, dz)
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
	c.batch.Reset()
	c.vertices = c.vertices[:0]
	c.indices = c.indices[:0]
	for i := range c.cubes {
		phase := float64(i+1)*.15 + float64(c.frame)*.04
		x, y := 380+380*math.Sin(phase), 300+84*math.Cos(phase*2.5)
		if !c.batch.Add(c.cubes[i], x, y) {
			panic("cube batch overflow")
		}
		c.old[i].Draw(c.expected, c.white, x, y)
	}

	c.batch.Draw(c.actual)
	c.actual.ReadPixels(c.pixelsA)
	c.expected.ReadPixels(c.pixelsB)
	if !bytes.Equal(c.pixelsA, c.pixelsB) {
		c.err = fmt.Errorf("cube batch pixels differ at frame %d", c.frame)
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
		c = &solidCubeRenderCheck{batch: effects.NewSolidCubeBatch(12), actual: render.NewSurface(800, 600), expected: render.NewSurface(800, 600), white: ebiten.NewImage(3, 3), pixelsA: make([]byte, 800*600*4), pixelsB: make([]byte, 800*600*4), vertices: make([]ebiten.Vertex, 0, 1440), indices: make([]uint16, 0, 2160)}
		c.white.Fill(color.White)
		for i := range c.cubes {
			c.cubes[i], err = effects.NewSolidCube(presets.CocoCube(40))
			if err != nil {
				return nil, err
			}
			c.old[i] = legacyCocoCube{size: 40, angleX: float64(i) * .3, angleY: float64(i) * .2, angleZ: float64(i) * .1}
			c.cubes[i].Rotation = geometry.Vec3{X: c.old[i].angleX, Y: c.old[i].angleY, Z: c.old[i].angleZ}
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
	fmt.Printf("All %d batched cube captures match exactly: %s\n", c.checked, dir)
}
