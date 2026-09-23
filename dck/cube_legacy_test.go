package coco

import (
	"github.com/hajimehoshi/ebiten/v2"
	"image/color"
	"math"
)

// Frozen pre-extraction renderer, retained only as an independent fidelity oracle.
// legacyCocoCube represents a 3D cube
type legacyCocoCube struct {
	angleX float64
	angleY float64
	angleZ float64
	size   float64

	rotated      [8][3]float64
	projected    [8][2]float64
	depths       [6]legacyCocoFaceDepth
	faceVertices [4]ebiten.Vertex
	edgeVertices [4]ebiten.Vertex
	visibleFaces int
}

type legacyCocoFaceDepth struct {
	index int
	depth float64
}

var (
	legacyCocoVertices = [8][3]float64{
		{-1, -1, -1},
		{1, -1, -1},
		{1, 1, -1},
		{-1, 1, -1},
		{-1, -1, 1},
		{1, -1, 1},
		{1, 1, 1},
		{-1, 1, 1},
	}
	legacyCocoFaces = [6][4]int{
		{0, 3, 2, 1},
		{4, 5, 6, 7},
		{0, 1, 5, 4},
		{2, 3, 7, 6},
		{0, 4, 7, 3},
		{1, 2, 6, 5},
	}
	legacyCocoColors = [6]color.RGBA{
		{R: 255, G: 140, A: 255},
		{R: 255, G: 165, B: 50, A: 255},
		{R: 255, G: 180, B: 80, A: 255},
		{R: 255, G: 120, A: 255},
		{R: 255, G: 150, B: 30, A: 255},
		{R: 255, G: 200, B: 100, A: 255},
	}
	legacyCocoIndices = [6]uint16{0, 1, 2, 0, 2, 3}
)

func NewlegacyCocoCube(size float64) *legacyCocoCube {
	return &legacyCocoCube{
		size: size,
	}
}

func (c *legacyCocoCube) Rotate(dx, dy, dz float64) {
	c.angleX += dx
	c.angleY += dy
	c.angleZ += dz
}

// legacyCocoProject projects 3D coordinates to 2D
func legacyCocoProject(x, y, z float64) (float64, float64) {
	const perspective = 200.0
	factor := perspective / (perspective + z)
	return x * factor, y * factor
}

func (c *legacyCocoCube) updateGeometry(centerX, centerY float64) {
	cosX, sinX := math.Cos(c.angleX), math.Sin(c.angleX)
	cosY, sinY := math.Cos(c.angleY), math.Sin(c.angleY)
	cosZ, sinZ := math.Cos(c.angleZ), math.Sin(c.angleZ)
	halfSize := c.size / 2

	for i, vertex := range legacyCocoVertices {
		x := vertex[0] * halfSize
		y := vertex[1] * halfSize
		z := vertex[2] * halfSize

		y1 := y*cosX - z*sinX
		z1 := y*sinX + z*cosX
		y, z = y1, z1

		x1 := x*cosY + z*sinY
		z2 := -x*sinY + z*cosY
		x, z = x1, z2

		x2 := x*cosZ - y*sinZ
		y2 := x*sinZ + y*cosZ
		x, y = x2, y2

		c.rotated[i] = [3]float64{x, y, z}
		x2D, y2D := legacyCocoProject(x, y, z)
		c.projected[i] = [2]float64{centerX + x2D, centerY + y2D}
	}

	c.visibleFaces = 0
	for i, face := range legacyCocoFaces {
		a := c.rotated[face[0]]
		b := c.rotated[face[1]]
		d := c.rotated[face[2]]
		abX, abY, abZ := b[0]-a[0], b[1]-a[1], b[2]-a[2]
		adX, adY, adZ := d[0]-a[0], d[1]-a[1], d[2]-a[2]
		normalX := abY*adZ - abZ*adY
		normalY := abZ*adX - abX*adZ
		normalZ := abX*adY - abY*adX

		centerX, centerY := 0.0, 0.0
		centerZ := 0.0
		for _, vi := range face {
			centerX += c.rotated[vi][0]
			centerY += c.rotated[vi][1]
			centerZ += c.rotated[vi][2]
		}
		centerX /= 4
		centerY /= 4
		centerZ /= 4

		// The camera is at (0, 0, -200). Discard faces whose outward
		// normal points away from it.
		facing := normalX*-centerX + normalY*-centerY + normalZ*(-200-centerZ)
		if facing <= 0 {
			continue
		}
		c.depths[c.visibleFaces] = legacyCocoFaceDepth{index: i, depth: centerZ}
		c.visibleFaces++
	}

	// Larger Z is farther from the camera and must be painted first.
	for i := 1; i < c.visibleFaces; i++ {
		for j := i; j > 0 && c.depths[j-1].depth < c.depths[j].depth; j-- {
			c.depths[j-1], c.depths[j] = c.depths[j], c.depths[j-1]
		}
	}
}

func legacyCocoVertex(vertex *ebiten.Vertex, x, y float64, clr color.RGBA) {
	vertex.DstX = float32(x)
	vertex.DstY = float32(y)
	vertex.SrcX = 1
	vertex.SrcY = 1
	vertex.ColorR = float32(clr.R) / 0xff
	vertex.ColorG = float32(clr.G) / 0xff
	vertex.ColorB = float32(clr.B) / 0xff
	vertex.ColorA = float32(clr.A) / 0xff
}

func (c *legacyCocoCube) drawEdge(screen, fillImage *ebiten.Image, from, to [2]float64, clr color.RGBA) {
	dx, dy := to[0]-from[0], to[1]-from[1]
	length := math.Hypot(dx, dy)
	if length == 0 {
		return
	}
	offsetX := -dy / length / 2
	offsetY := dx / length / 2
	legacyCocoVertex(&c.edgeVertices[0], from[0]+offsetX, from[1]+offsetY, clr)
	legacyCocoVertex(&c.edgeVertices[1], to[0]+offsetX, to[1]+offsetY, clr)
	legacyCocoVertex(&c.edgeVertices[2], from[0]-offsetX, from[1]-offsetY, clr)
	legacyCocoVertex(&c.edgeVertices[3], to[0]-offsetX, to[1]-offsetY, clr)
	screen.DrawTriangles(c.edgeVertices[:], legacyCocoIndices[:], fillImage, nil)
}

// Draw draws the 3D cube at the specified position.
func (c *legacyCocoCube) Draw(screen, fillImage *ebiten.Image, centerX, centerY float64) {
	c.updateGeometry(centerX, centerY)

	for _, legacyCocoFaceDepth := range c.depths[:c.visibleFaces] {
		face := legacyCocoFaces[legacyCocoFaceDepth.index]
		faceColor := legacyCocoColors[legacyCocoFaceDepth.index]

		for i, vertexIndex := range face {
			point := c.projected[vertexIndex]
			legacyCocoVertex(&c.faceVertices[i], point[0], point[1], faceColor)
		}
		screen.DrawTriangles(c.faceVertices[:], legacyCocoIndices[:], fillImage, nil)

		// Draw edges with darker color for better visibility
		edgeColor := color.RGBA{
			R: faceColor.R * 3 / 4,
			G: faceColor.G * 3 / 4,
			B: faceColor.B * 3 / 4,
			A: 255,
		}
		for i := 0; i < 4; i++ {
			j := (i + 1) % 4
			from := c.projected[face[i]]
			to := c.projected[face[j]]
			c.drawEdge(screen, fillImage, from, to, edgeColor)
		}
	}
}
