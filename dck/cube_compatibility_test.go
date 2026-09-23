package coco

import (
	"image/color"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/democonstructionkit/effects"
	"github.com/olivierh59500/democonstructionkit/geometry"
	"github.com/olivierh59500/democonstructionkit/presets"
)

// Collect the frozen renderer's submissions without using DCK geometry helpers.
func (c *legacyCocoCube) submittedGeometry(x, y float64) ([]ebiten.Vertex, []uint16) {
	c.updateGeometry(x, y)
	vertices := make([]ebiten.Vertex, 0, 120)
	indices := make([]uint16, 0, 180)
	appendQuad := func(v [4]ebiten.Vertex) {
		base := uint16(len(vertices))
		vertices = append(vertices, v[:]...)
		indices = append(indices, base, base+1, base+2, base, base+2, base+3)
	}
	for _, depth := range c.depths[:c.visibleFaces] {
		face := legacyCocoFaces[depth.index]
		clr := legacyCocoColors[depth.index]
		for i, vi := range face {
			p := c.projected[vi]
			legacyCocoVertex(&c.faceVertices[i], p[0], p[1], clr)
		}
		appendQuad(c.faceVertices)
		edge := color.RGBA{R: clr.R * 3 / 4, G: clr.G * 3 / 4, B: clr.B * 3 / 4, A: 255}
		for i := 0; i < 4; i++ {
			from, to := c.projected[face[i]], c.projected[face[(i+1)%4]]
			dx, dy := to[0]-from[0], to[1]-from[1]
			length := math.Hypot(dx, dy)
			if length == 0 {
				continue
			}
			ox, oy := -dy/length/2, dx/length/2
			legacyCocoVertex(&c.edgeVertices[0], from[0]+ox, from[1]+oy, edge)
			legacyCocoVertex(&c.edgeVertices[1], to[0]+ox, to[1]+oy, edge)
			legacyCocoVertex(&c.edgeVertices[2], from[0]-ox, from[1]-oy, edge)
			legacyCocoVertex(&c.edgeVertices[3], to[0]-ox, to[1]-oy, edge)
			appendQuad(c.edgeVertices)
		}
	}
	return vertices, indices
}
func TestSharedCocoCubeMatchesOriginalGeometry(t *testing.T) {
	cube, err := effects.NewSolidCube(presets.CocoCube(40))
	if err != nil {
		t.Fatal(err)
	}
	defer cube.Close()
	for instance := 0; instance < 12; instance++ {
		old := legacyCocoCube{size: 40, angleX: float64(instance) * .3, angleY: float64(instance) * .2, angleZ: float64(instance) * .1}
		cube.Rotation = geometry.Vec3{X: old.angleX, Y: old.angleY, Z: old.angleZ}
		for frame := 0; frame < 1200; frame++ {
			x, y := 380+380*math.Sin(float64(frame)*.04), 300+84*math.Cos(float64(frame)*.1)
			wantV, wantI := old.submittedGeometry(x, y)
			gotV, gotI := cube.Geometry(x, y)
			if len(gotV) != len(wantV) || len(gotI) != len(wantI) {
				t.Fatalf("instance %d frame %d count mismatch", instance, frame)
			}
			for i := range gotV {
				if gotV[i] != wantV[i] {
					t.Fatalf("instance %d frame %d vertex %d got=%+v want=%+v", instance, frame, i, gotV[i], wantV[i])
				}
			}
			for i := range gotI {
				if gotI[i] != wantI[i] {
					t.Fatalf("index %d changed", i)
				}
			}
			dx, dy, dz := .02*(1+float64(instance)*.1), .03*(1+float64(instance)*.15), .01*(1+float64(instance)*.05)
			old.Rotate(dx, dy, dz)
			cube.Rotate(dx, dy, dz)
		}
	}
}
