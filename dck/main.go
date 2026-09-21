// Package coco implements the COCO IS THE BEST demo.
package coco

import originalassets "github.com/olivierh59500/go-cocoisthebest"

import (
	"bytes"

	"github.com/olivierh59500/democonstructionkit/composite"
	"github.com/olivierh59500/democonstructionkit/scrolling"
	"image"
	"image/color"
	_ "image/png"
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	audio "github.com/olivierh59500/democonstructionkit/sound/output"
)

const (
	screenWidth  = 800
	screenHeight = 600

	// Constants for effects
	nbCubes    = 12
	nbDMALogos = 16
	fontHeight = 36

	// The rotozoom quad is deliberately larger than the scene so rotations and
	// oscillations never reveal its edges. Its texture is repeated by the GPU.
	rotoWidth  = screenWidth * 4
	rotoHeight = screenHeight * 4

	sampleRate      = 44100
	scrollPadding   = "     "
	introScrollText = scrollPadding + scrollPadding +
		"IF YOU THINK THIS IS ALL, YOU'RE SO WRONG..." + scrollPadding
	demoScrollText = scrollPadding + scrollPadding +
		"WELCOME TO THE COCO IS THE BEST DEMO! " + scrollPadding +
		"THIS DEMO COMBINES THE BEST EFFECTS FROM VARIOUS ATARI ST DEMOS. " + scrollPadding +
		"GREETINGS TO ALL DEMOSCENE LOVERS! " + scrollPadding + scrollPadding

	// LogicalScreenWidth and LogicalScreenHeight are the dimensions of the
	// original 4:3 scene. Wider displays add centered side areas instead of
	// stretching the artwork.
	LogicalScreenWidth  = screenWidth
	LogicalScreenHeight = screenHeight
)

// Embedded assets
var titleImgData = originalassets.
	DCKAssetTitleImgData()

var barsImgData = originalassets.
	DCKAssetBarsImgData()

var cocoImgData = originalassets.
	DCKAssetCocoImgData()

var dmaLogoImgData = originalassets.
	DCKAssetDmaLogoImgData()

var fontImgData = originalassets.
	DCKAssetFontImgData()

var musicData = originalassets.

	// Wave types for distortion
	DCKAssetMusicData()

const (
	cdZero = iota
	cdSlowSin
	cdMedSin
	cdFastSin
	cdSlowDist
	cdMedDist
	cdFastDist
	cdSplitted
)

// Letter for font rendering
type Letter struct {
	x, y  int
	width int
	glyph *ebiten.Image
}

// Cube3D represents a 3D cube
type Cube3D struct {
	angleX float64
	angleY float64
	angleZ float64
	size   float64

	rotated      [8][3]float64
	projected    [8][2]float64
	depths       [6]faceDepth
	faceVertices [4]ebiten.Vertex
	edgeVertices [4]ebiten.Vertex
	visibleFaces int
}

type faceDepth struct {
	index int
	depth float64
}

type gameState uint8

const (
	gameStateIntro gameState = iota
	gameStateDemo
)

var (
	cubeVertices = [8][3]float64{
		{-1, -1, -1},
		{1, -1, -1},
		{1, 1, -1},
		{-1, 1, -1},
		{-1, -1, 1},
		{1, -1, 1},
		{1, 1, 1},
		{-1, 1, 1},
	}
	cubeFaces = [6][4]int{
		{0, 3, 2, 1},
		{4, 5, 6, 7},
		{0, 1, 5, 4},
		{2, 3, 7, 6},
		{0, 4, 7, 3},
		{1, 2, 6, 5},
	}
	cubeFaceColors = [6]color.RGBA{
		{R: 255, G: 140, A: 255},
		{R: 255, G: 165, B: 50, A: 255},
		{R: 255, G: 180, B: 80, A: 255},
		{R: 255, G: 120, A: 255},
		{R: 255, G: 150, B: 30, A: 255},
		{R: 255, G: 200, B: 100, A: 255},
	}
	cubeFaceIndices = [6]uint16{0, 1, 2, 0, 2, 3}
)

func NewCube3D(size float64) *Cube3D {
	return &Cube3D{
		size: size,
	}
}

func (c *Cube3D) Rotate(dx, dy, dz float64) {
	c.angleX += dx
	c.angleY += dy
	c.angleZ += dz
}

// project3D projects 3D coordinates to 2D
func project3D(x, y, z float64) (float64, float64) {
	const perspective = 200.0
	factor := perspective / (perspective + z)
	return x * factor, y * factor
}

func (c *Cube3D) updateGeometry(centerX, centerY float64) {
	cosX, sinX := math.Cos(c.angleX), math.Sin(c.angleX)
	cosY, sinY := math.Cos(c.angleY), math.Sin(c.angleY)
	cosZ, sinZ := math.Cos(c.angleZ), math.Sin(c.angleZ)
	halfSize := c.size / 2

	for i, vertex := range cubeVertices {
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
		x2D, y2D := project3D(x, y, z)
		c.projected[i] = [2]float64{centerX + x2D, centerY + y2D}
	}

	c.visibleFaces = 0
	for i, face := range cubeFaces {
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
		c.depths[c.visibleFaces] = faceDepth{index: i, depth: centerZ}
		c.visibleFaces++
	}

	// Larger Z is farther from the camera and must be painted first.
	for i := 1; i < c.visibleFaces; i++ {
		for j := i; j > 0 && c.depths[j-1].depth < c.depths[j].depth; j-- {
			c.depths[j-1], c.depths[j] = c.depths[j], c.depths[j-1]
		}
	}
}

func setColoredVertex(vertex *ebiten.Vertex, x, y float64, clr color.RGBA) {
	vertex.DstX = float32(x)
	vertex.DstY = float32(y)
	vertex.SrcX = 1
	vertex.SrcY = 1
	vertex.ColorR = float32(clr.R) / 0xff
	vertex.ColorG = float32(clr.G) / 0xff
	vertex.ColorB = float32(clr.B) / 0xff
	vertex.ColorA = float32(clr.A) / 0xff
}

func (c *Cube3D) drawEdge(screen, fillImage *ebiten.Image, from, to [2]float64, clr color.RGBA) {
	dx, dy := to[0]-from[0], to[1]-from[1]
	length := math.Hypot(dx, dy)
	if length == 0 {
		return
	}
	offsetX := -dy / length / 2
	offsetY := dx / length / 2
	setColoredVertex(&c.edgeVertices[0], from[0]+offsetX, from[1]+offsetY, clr)
	setColoredVertex(&c.edgeVertices[1], to[0]+offsetX, to[1]+offsetY, clr)
	setColoredVertex(&c.edgeVertices[2], from[0]-offsetX, from[1]-offsetY, clr)
	setColoredVertex(&c.edgeVertices[3], to[0]-offsetX, to[1]-offsetY, clr)
	screen.DrawTriangles(c.edgeVertices[:], cubeFaceIndices[:], fillImage, nil)
}

// Draw draws the 3D cube at the specified position.
func (c *Cube3D) Draw(screen, fillImage *ebiten.Image, centerX, centerY float64) {
	c.updateGeometry(centerX, centerY)

	for _, faceDepth := range c.depths[:c.visibleFaces] {
		face := cubeFaces[faceDepth.index]
		faceColor := cubeFaceColors[faceDepth.index]

		for i, vertexIndex := range face {
			point := c.projected[vertexIndex]
			setColoredVertex(&c.faceVertices[i], point[0], point[1], faceColor)
		}
		screen.DrawTriangles(c.faceVertices[:], cubeFaceIndices[:], fillImage, nil)

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

// CRT Shader
const crtShaderSrc = `
package main

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	var uv vec2
	uv = texCoord

	// Barrel distortion
	var dc vec2
	dc = uv - 0.5
	dc = dc * (1.0 + dot(dc, dc) * 0.15)
	uv = dc + 0.5

	if uv.x < 0.0 || uv.x > 1.0 || uv.y < 0.0 || uv.y > 1.0 {
		return vec4(0.0, 0.0, 0.0, 1.0)
	}

	var col vec4
	col = imageSrc0At(uv)

	// Scanlines
	var scanline float
	scanline = sin(uv.y * 800.0) * 0.04
	col.rgb = col.rgb - scanline

	// RGB shift
	var rShift float
	var bShift float
	rShift = imageSrc0At(uv + vec2(0.002, 0.0)).r
	bShift = imageSrc0At(uv - vec2(0.002, 0.0)).b
	col.r = rShift
	col.b = bShift

	// Vignette
	var vignette float
	vignette = 1.0 - dot(dc, dc) * 0.5
	col.rgb = col.rgb * vignette

	return col * color
}
`

// Game state
type Game struct {
	scrollRenderer *scrolling.Scrolling
	stripBatch     *composite.QuadBatch
	// Images
	titleImg   *ebiten.Image
	barsImg    *ebiten.Image
	barStrips  [10]*ebiten.Image
	cocoImg    *ebiten.Image
	dmaLogoImg *ebiten.Image
	fontImg    *ebiten.Image

	// Canvases
	introStrip  *ebiten.Image
	mainCanvas  *ebiten.Image
	scrollSurf  *ebiten.Image
	titleCanvas *ebiten.Image

	// Audio
	audioContext *audio.Context
	audioPlayer  *audio.Player
	ymPlayer     *YMPlayer
	audioReady   bool

	// State
	state    gameState
	demoTime float64

	// Intro scrolling
	introX            int
	introLetter       int
	introTile         int
	introSpeed        int
	introTextRunes    []rune
	surfScroll1       *ebiten.Image
	surfScroll2       *ebiten.Image
	introScrollSource *ebiten.Image

	// Font data
	letterData map[rune]Letter

	// CRT Shader
	crtShader *ebiten.Shader

	// Demo effects
	// Copper bars
	cnt       float64
	cnt2      float64
	copperSin []int

	// 3D Cubes
	cubes         [nbCubes]Cube3D
	spritePos     [nbCubes]float64
	cubeFillImage *ebiten.Image

	// DMA logo sprites (16 logos in 4x4 grid)
	dmaSprites [nbDMALogos]DMASprite
	ctrSprite  float64

	// Scrolling text (megatwist style)
	frontWavePos    int
	letterNum       int
	letterDecal     int
	curves          [8][]int
	frontMainWave   []int
	position        []int
	scrollTextRunes []rune
	displayedLetter int
	scrollVertices  []ebiten.Vertex
	scrollIndices   []uint16

	// Rotozoom
	posXi        float64
	posZi        float64
	posRi        float64
	rotoVertices [4]ebiten.Vertex

	// Title logo animation
	logoX float64
	hold  int
	// Speed control
	speedMultiplier float64

	// Responsive layout and input state
	logicalWidth   int
	touchIDs       [8]ebiten.TouchID
	controlPressed [controlCount]bool
}

type DMASprite struct {
	x, y float64
}

func NewGame() *Game {
	g := &Game{
		state:           gameStateIntro,
		introX:          -1,
		introLetter:     -1,
		introTile:       -1,
		introSpeed:      8,
		letterData:      make(map[rune]Letter),
		speedMultiplier: 1.0,
		displayedLetter: -1,
		logicalWidth:    screenWidth,
		logoX:           0.5, // Center the logo (0.5 = centered)
		hold:            0,   // Start immediately
	}

	g.introTextRunes = []rune(introScrollText)
	g.scrollTextRunes = []rune(demoScrollText)

	// Load images
	g.loadImages()

	// Create canvases
	g.introStrip = ebiten.NewImage(screenWidth, int(fontHeight*2))
	g.mainCanvas = ebiten.NewImageWithOptions(
		image.Rect(0, 0, screenWidth, screenHeight),
		&ebiten.NewImageOptions{Unmanaged: true},
	)
	g.surfScroll1 = ebiten.NewImage(screenWidth+96, int(fontHeight*2))
	g.surfScroll2 = ebiten.NewImage(screenWidth+96, int(fontHeight*2))
	g.introScrollSource = g.surfScroll1.SubImage(
		image.Rect(g.introSpeed, 0, g.surfScroll1.Bounds().Dx(), int(fontHeight*2)),
	).(*ebiten.Image)
	g.scrollSurf = ebiten.NewImageWithOptions(
		image.Rect(0, 0, screenWidth*2, fontHeight*3),
		&ebiten.NewImageOptions{Unmanaged: true},
	)
	g.titleCanvas = ebiten.NewImage(screenWidth, 72)
	g.cubeFillImage = ebiten.NewImage(3, 3)
	g.cubeFillImage.Fill(color.White)
	g.initScrollMesh()

	for i := range g.rotoVertices {
		g.rotoVertices[i].ColorR = 0.5
		g.rotoVertices[i].ColorG = 0.5
		g.rotoVertices[i].ColorB = 0.5
		g.rotoVertices[i].ColorA = 1
	}
	g.rotoVertices[1].SrcX = rotoWidth
	g.rotoVertices[2].SrcY = rotoHeight
	g.rotoVertices[3].SrcX = rotoWidth
	g.rotoVertices[3].SrcY = rotoHeight

	// Init font
	g.initFontData()
	g.initBarStrips()

	// Init 3D cubes
	for i := 0; i < nbCubes; i++ {
		g.cubes[i].size = 40
		// Set initial position offset for each cube
		g.spritePos[i] = float64(0.15) * float64(i+1)
		// Set different initial rotations
		g.cubes[i].angleX = float64(i) * 0.3
		g.cubes[i].angleY = float64(i) * 0.2
		g.cubes[i].angleZ = float64(i) * 0.1
	}

	// Init wave curves for scrolling
	g.createCurves()
	g.precalcPosition()
	g.precalcMainWave()

	// Init copper bars sine table
	g.initCopperSin()

	// Compile CRT shader
	var err error
	g.crtShader, err = ebiten.NewShader([]byte(crtShaderSrc))
	if err != nil {
		log.Printf("Failed to compile CRT shader: %v", err)
	}

	return g
}

// initCopperSin initializes the sine table for copper bars animation
func (g *Game) initCopperSin() {
	g.copperSin = []int{
		264, 264, 268, 272, 276, 280, 280, 284, 288, 292, 296, 296, 300, 304, 308, 312, 312, 316, 320, 324, 328, 328, 332, 336, 340, 340, 344, 348, 352, 352, 356, 360, 364, 364, 368, 372, 376, 376, 380, 384, 388, 388, 392, 396, 396, 400, 404, 404, 408, 412, 412, 416, 420, 420, 424, 428, 428, 432, 436, 436, 440, 440, 444, 448, 448, 452, 452, 456, 456, 460, 460, 464, 464, 468, 472, 472, 472, 476, 476, 480, 480, 484, 484, 488, 488, 488, 492, 492, 496, 496, 496, 500, 500, 500, 504, 504, 504, 508, 508, 508, 512, 512, 512, 512, 516, 516, 516, 516, 520, 520, 520, 520, 520, 520, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 520, 520, 520, 520, 520, 520, 516, 516, 516, 516, 512, 512, 512, 512, 508, 508, 508, 508, 504, 504, 504, 500, 500, 500, 496, 496, 492, 492, 492, 488, 488, 484, 484, 480, 480, 480, 476, 476, 472, 472, 468, 468, 464, 464, 460, 456, 456, 452, 452, 448, 448, 444, 444, 440, 436, 436, 432, 428, 428, 424, 424, 420, 416, 416, 412, 408, 408, 404, 400, 400, 396, 392, 388, 388, 384, 380, 380, 376, 372, 368, 368, 364, 360, 356, 356, 352, 348, 344, 344, 340, 336, 332, 328, 328, 324, 320, 316, 316, 312, 308, 304, 300, 300, 296, 292, 288, 284, 284, 280, 276, 272, 268, 264, 264, 264, 260, 256, 252, 252, 248, 244, 240, 236, 236, 232, 228, 224, 220, 220, 216, 212, 208, 204, 204, 200, 196, 192, 192, 188, 184, 180, 176, 176, 172, 168, 164, 164, 160, 156, 152, 152, 148, 144, 144, 140, 136, 132, 132, 128, 124, 124, 120, 116, 116, 112, 108, 108, 104, 100, 100, 96, 96, 92, 88, 88, 84, 84, 80, 76, 76, 72, 72, 68, 68, 64, 64, 60, 60, 56, 56, 52, 52, 48, 48, 44, 44, 40, 40, 40, 36, 36, 32, 32, 32, 28, 28, 28, 24, 24, 24, 20, 20, 20, 16, 16, 16, 16, 12, 12, 12, 12, 12, 8, 8, 8, 8, 8, 8, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 8, 8, 8, 8, 8, 8, 12, 12, 12, 12, 12, 16, 16, 16, 20, 20, 20, 20, 24, 24, 24, 28, 28, 28, 32, 32, 36, 36, 36, 40, 40, 44, 44, 44, 48, 48, 52, 52, 56, 56, 60, 60, 64, 64, 68, 68, 72, 72, 76, 80, 80, 84, 84, 88, 92, 92, 96, 96, 100, 104, 104, 108, 112, 112, 116, 120, 120, 124, 128, 128, 132, 136, 136, 140, 144, 148, 148, 152, 156, 156, 160, 164, 168, 168, 172, 176, 180, 180, 184, 188, 192, 196, 196, 200, 204, 208, 212, 212, 216, 220, 224, 224, 228, 232, 236, 240, 244, 244, 248, 252, 256, 260, 260, 264, 264, 268, 272, 276, 280, 280, 284, 288, 292, 296, 296, 300, 304, 308, 312, 312, 316, 320, 324, 328, 328, 332, 336, 340, 340, 344, 348, 352, 352, 356, 360, 364, 364, 368, 372, 376, 376, 380, 384, 388, 388, 392, 396, 396, 400, 404, 404, 408, 412, 412, 416, 420, 420, 424, 428, 428, 432, 436, 436, 440, 440, 444, 448, 448, 452, 452, 456, 456, 460, 460, 464, 464, 468, 472, 472, 472, 476, 476, 480, 480, 484, 484, 488, 488, 488, 492, 492, 496, 496, 496, 500, 500, 500, 504, 504, 504, 508, 508, 508, 512, 512, 512, 512, 516, 516, 516, 516, 520, 520, 520, 520, 520, 520, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 524, 520, 520, 520, 520, 520, 520, 516, 516, 516, 516, 512, 512, 512, 512, 508, 508, 508, 508, 504, 504, 504, 500, 500, 500, 496, 496, 492, 492, 492, 488, 488, 484, 484, 480, 480, 480, 476, 476, 472, 472, 468, 468, 464, 464, 460, 456, 456, 452, 452, 448, 448, 444, 444, 440, 436, 436, 432, 428, 428, 424, 424, 420, 416, 416, 412, 408, 408, 404, 400, 400, 396, 392, 388, 388, 384, 380, 380, 376, 372, 368, 368, 364, 360, 356, 356, 352, 348, 344, 344, 340, 336, 332, 328, 328, 324, 320, 316, 316, 312, 308, 304, 300, 300, 296, 292, 288, 284, 284, 280, 276, 272, 268, 264, 264, 264, 260, 256, 252, 252, 248, 244, 240, 236, 236, 232, 228, 224, 220, 220, 216, 212, 208, 204, 204, 200, 196, 192, 192, 188, 184, 180, 176, 176, 172, 168, 164, 164, 160, 156, 152, 152, 148, 144, 144, 140, 136, 132, 132, 128, 124, 124, 120, 116, 116, 112, 108, 108, 104, 100, 100, 96, 96, 92, 88, 88, 84, 84, 80, 76, 76, 72, 72, 68, 68, 64, 64, 60, 60, 56, 56, 52, 52, 48, 48, 44, 44, 40, 40, 40, 36, 36, 32, 32, 32, 28, 28, 28, 24, 24, 24, 20, 20, 20, 16, 16, 16, 16, 12, 12, 12, 12, 12, 8, 8, 8, 8, 8, 8, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 8, 8, 8, 8, 8, 8, 12, 12, 12, 12, 12, 16, 16, 16, 20, 20, 20, 20, 24, 24, 24, 28, 28, 28, 32, 32, 36, 36, 36, 40, 40, 44, 44, 44, 48, 48, 52, 52, 56, 56, 60, 60, 64, 64, 68, 68, 72, 72, 76, 80, 80, 84, 84, 88, 92, 92, 96, 96, 100, 104, 104, 108, 112, 112, 116, 120, 120, 124, 128, 128, 132, 136, 136, 140, 144, 148, 148, 152, 156, 156, 160, 164, 168, 168, 172, 176, 180, 180, 184, 188, 192, 196, 196, 200, 204, 208, 212, 212, 216, 220, 224, 224, 228, 232, 236, 240, 244, 244, 248, 252, 256, 260, 260,
	}
}

func (g *Game) loadImages() {
	var err error

	img, _, err := image.Decode(bytes.NewReader(titleImgData))
	if err != nil {
		log.Printf("Failed to load title: %v", err)
	} else {
		g.titleImg = ebiten.NewImageFromImage(img)
	}

	img, _, err = image.Decode(bytes.NewReader(barsImgData))
	if err != nil {
		log.Printf("Failed to load bars: %v", err)
	} else {
		g.barsImg = ebiten.NewImageFromImage(img)
	}

	img, _, err = image.Decode(bytes.NewReader(cocoImgData))
	if err != nil {
		log.Printf("Failed to load coco: %v", err)
	} else {
		g.cocoImg = ebiten.NewImageFromImage(img)
	}

	img, _, err = image.Decode(bytes.NewReader(dmaLogoImgData))
	if err != nil {
		log.Printf("Failed to load dma logo: %v", err)
	} else {
		g.dmaLogoImg = ebiten.NewImageFromImage(img)
	}

	img, _, err = image.Decode(bytes.NewReader(fontImgData))
	if err != nil {
		log.Printf("Failed to load font: %v", err)
	} else {
		g.fontImg = ebiten.NewImageFromImage(img)
	}
}

func (g *Game) initAudio() {
	g.audioContext = audio.NewContext(sampleRate)

	var err error
	g.ymPlayer, err = NewYMPlayer(musicData, sampleRate, true)
	if err != nil {
		log.Printf("Failed to create YM player: %v", err)
		return
	}

	g.audioPlayer, err = g.audioContext.NewPlayer(g.ymPlayer)
	if err != nil {
		log.Printf("Failed to create audio player: %v", err)
		g.ymPlayer.Close()
		g.ymPlayer = nil
		return
	}

	// Music will start when transitioning from intro to demo phase
}

func (g *Game) initFontData() {
	data := []struct {
		char  rune
		x, y  int
		width int
	}{
		{' ', 0, 0, 32}, {'!', 48, 0, 16}, {'"', 96, 0, 32},
		{'\'', 336, 0, 16}, {'(', 384, 0, 32}, {')', 432, 0, 32},
		{'+', 48, 36, 48}, {',', 96, 36, 16}, {'-', 144, 36, 32},
		{'.', 192, 36, 16}, {'0', 288, 36, 48}, {'1', 336, 36, 48},
		{'2', 384, 36, 48}, {'3', 432, 36, 48}, {'4', 0, 72, 48},
		{'5', 48, 72, 48}, {'6', 96, 72, 48}, {'7', 144, 72, 48},
		{'8', 192, 72, 48}, {'9', 240, 72, 48}, {':', 288, 72, 16},
		{';', 336, 72, 16}, {'<', 384, 72, 32}, {'=', 432, 72, 32},
		{'>', 0, 108, 32}, {'?', 48, 108, 48}, {'A', 144, 108, 48},
		{'B', 192, 108, 48}, {'C', 240, 108, 48}, {'D', 288, 108, 48},
		{'E', 336, 108, 48}, {'F', 384, 108, 48}, {'G', 432, 108, 48},
		{'H', 0, 144, 48}, {'I', 48, 144, 16}, {'J', 96, 144, 48},
		{'K', 144, 144, 48}, {'L', 192, 144, 48}, {'M', 240, 144, 48},
		{'N', 288, 144, 48}, {'O', 336, 144, 48}, {'P', 384, 144, 48},
		{'Q', 432, 144, 48}, {'R', 0, 180, 48}, {'S', 48, 180, 48},
		{'T', 96, 180, 48}, {'U', 144, 180, 48}, {'V', 192, 180, 48},
		{'W', 240, 180, 48}, {'X', 288, 180, 48}, {'Y', 336, 180, 48},
		{'Z', 384, 180, 48},
	}

	for _, d := range data {
		letter := Letter{x: d.x, y: d.y, width: d.width}
		if g.fontImg != nil {
			letter.glyph = g.fontImg.SubImage(
				image.Rect(d.x, d.y, d.x+d.width, d.y+fontHeight),
			).(*ebiten.Image)
		}
		g.letterData[d.char] = letter
	}
}

func (g *Game) initBarStrips() {
	if g.barsImg == nil || g.barsImg.Bounds().Dy() < len(g.barStrips)*2 {
		return
	}
	width := g.barsImg.Bounds().Dx()
	for i := range g.barStrips {
		y := i * 2
		g.barStrips[i] = g.barsImg.SubImage(image.Rect(0, y, width, y+2)).(*ebiten.Image)
	}
}

func (g *Game) createCurves() {
	for funcType := 0; funcType <= 7; funcType++ {
		var step, progress float64

		switch funcType {
		case cdZero:
			step, progress = 2.25, 0
		case cdSlowSin:
			step, progress = 0.20, 140
		case cdMedSin:
			step, progress = 0.25, 175
		case cdFastSin:
			step, progress = 0.30, 210
		case cdSlowDist:
			step, progress = 0.12, 175
		case cdMedDist:
			step, progress = 0.16, 210
		case cdFastDist:
			step, progress = 0.20, 245
		case cdSplitted:
			step, progress = 0.18, 0
		}

		local := []float64{}
		decal := 0.0
		previous := 0
		maxAngle := 360.0
		if funcType == cdSplitted {
			maxAngle = 720.0
		}

		for i := 0.0; i < maxAngle-step; i += step {
			val := 0.0
			rad := i * math.Pi / 180

			switch funcType {
			case cdZero:
				val = 0
			case cdSlowSin:
				val = 100 * math.Sin(rad)
			case cdMedSin:
				val = 110 * math.Sin(rad)
			case cdFastSin:
				val = 120 * math.Sin(rad)
			case cdSlowDist:
				val = 100*math.Sin(rad) + 25.0*math.Sin(rad*10)
			case cdMedDist:
				val = 110*math.Sin(rad) + 27.5*math.Sin(rad*9)
			case cdFastDist:
				val = 120*math.Sin(rad) + 30.0*math.Sin(rad*8)
			case cdSplitted:
				dir := 1.0
				if len(local)%2 == 1 {
					dir = -1.0
				}
				amp := 12.0
				if i < 160 {
					amp *= i / 160
				} else if (720 - 160) < i {
					amp *= (720 - i) / 160
				}
				val = 90*math.Sin(rad) + dir*amp*math.Sin(rad*3)
			}
			local = append(local, val)
		}

		g.curves[funcType] = make([]int, len(local))
		for i := 0; i < len(local); i++ {
			nitem := -int(math.Floor(local[i] - decal))
			g.curves[funcType][i] = nitem - previous
			previous = nitem
			decal += progress / float64(len(local))
		}
	}
}

func (g *Game) precalcPosition() {
	count := 0
	g.position = []int{}

	for _, r := range g.scrollTextRunes {
		if letter, ok := g.letterData[r]; ok {
			count += int(float64(letter.width) * 3.0)
			g.position = append(g.position, count)
		}
	}
}

func (g *Game) precalcMainWave() {
	frontMainWaveTable := []int{
		cdSlowSin, cdSlowSin, cdSlowDist, cdSlowSin,
		cdSlowSin, cdMedSin, cdFastSin, cdMedSin,
		cdSlowSin, cdMedDist, cdMedSin, cdSlowSin,
		cdSplitted,
	}

	count := 0
	g.frontMainWave = []int{}

	for _, waveType := range frontMainWaveTable {
		wave := g.curves[waveType]
		for _, val := range wave {
			count += val
			g.frontMainWave = append(g.frontMainWave, count)
		}
	}
}

func (g *Game) initScrollMesh() {
	const lineCount = screenHeight - 72
	g.scrollVertices = make([]ebiten.Vertex, lineCount*4)
	g.scrollIndices = make([]uint16, lineCount*6)

	for line := 0; line < lineCount; line++ {
		vertexBase := line * 4
		for i := 0; i < 4; i++ {
			vertex := &g.scrollVertices[vertexBase+i]
			vertex.ColorR = 1
			vertex.ColorG = 1
			vertex.ColorB = 1
			vertex.ColorA = 1
		}

		indexBase := line * 6
		base := uint16(vertexBase)
		g.scrollIndices[indexBase] = base
		g.scrollIndices[indexBase+1] = base + 1
		g.scrollIndices[indexBase+2] = base + 2
		g.scrollIndices[indexBase+3] = base + 1
		g.scrollIndices[indexBase+4] = base + 2
		g.scrollIndices[indexBase+5] = base + 3
	}
}

func (g *Game) getSum(arr []int, index, decal int) int {
	n := len(arr)
	if n == 0 {
		return decal
	}

	maxVal := arr[n-1]
	f := index / n
	m := index % n
	return decal + f*maxVal + arr[m]
}

func (g *Game) getWave(i int) int {
	return g.getSum(g.frontMainWave, i, 0)
}

func (g *Game) getPosition(i int) int {
	if i > 0 {
		return g.getSum(g.position, i-1, 0)
	}
	return 0
}

func (g *Game) advanceScrollLetter(decalX int) {
	if len(g.position) == 0 {
		g.letterNum = 0
		g.letterDecal = 0
		return
	}
	for g.letterNum > 0 && decalX < g.getPosition(g.letterNum) {
		g.letterNum--
	}
	for g.getPosition(g.letterNum+1) <= decalX {
		g.letterNum++
	}
	g.letterDecal = g.getPosition(g.letterNum)
}

func (g *Game) scrollOffset(frontWavePos int) int {
	decalX := g.getWave(frontWavePos)
	for line := 1; line < fontHeight; line++ {
		decalX = min(decalX, g.getWave(frontWavePos+line))
	}
	return max(decalX, 0)
}

func (g *Game) getLetter(pos int) rune {
	if len(g.scrollTextRunes) == 0 {
		return ' '
	}
	return g.scrollTextRunes[pos%len(g.scrollTextRunes)]
}

func (g *Game) getIntroLetter(pos int) rune {
	if len(g.introTextRunes) == 0 {
		return ' '
	}
	return g.introTextRunes[pos%len(g.introTextRunes)]
}

func (g *Game) Update() error {
	// On Android, NewGame runs while the native library is loaded. Opening the
	// audio device there can block before the Activity has installed its view.
	if !g.audioReady {
		g.audioReady = true
		g.initAudio()
	}

	g.updateControls()

	if g.state == gameStateIntro {
		g.updateIntro()
	} else {
		g.updateDemo()
	}

	return nil
}

func (g *Game) updateIntro() {
	if g.introX < 0 {
		if g.introTile > -1 {
			char := g.getIntroLetter(g.introTile)
			if letter, ok := g.letterData[char]; ok {
				g.introX += int(float64(letter.width) * 2.0)
			}
		}
		g.introLetter++
		if g.introLetter >= len(g.introTextRunes) {
			g.state = gameStateDemo
			g.demoTime = 0
			// Start music
			if g.audioPlayer != nil && !g.audioPlayer.IsPlaying() {
				g.audioPlayer.Play()
			}
			return
		}
		g.introTile = g.introLetter
	}
	g.introX -= g.introSpeed

	// Scroll
	g.surfScroll2.Clear()
	g.surfScroll2.DrawImage(g.introScrollSource, nil)

	g.surfScroll1.Clear()
	g.surfScroll1.DrawImage(g.surfScroll2, nil)

	// Draw new letter
	char := g.getIntroLetter(g.introTile)
	if letter, ok := g.letterData[char]; ok {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2.0, 2.0)
		op.GeoM.Translate(float64(screenWidth+g.introX), 0)
		g.surfScroll1.DrawImage(letter.glyph, op)
	}
}

func (g *Game) updateDemo() {
	speed := g.speedMultiplier
	g.demoTime += speed

	// Update copper bars
	g.cnt += 3 * speed
	if g.cnt >= 1024 {
		g.cnt -= 1024
	}
	g.cnt2 -= 5 * speed
	if g.cnt2 < 0 {
		g.cnt2 += 1024
	}

	// Update 3D cubes
	for i := 0; i < nbCubes; i++ {
		g.spritePos[i] += 0.04 * speed
		g.cubes[i].Rotate(
			0.02*speed*(1+float64(i)*0.1),
			0.03*speed*(1+float64(i)*0.15),
			0.01*speed*(1+float64(i)*0.05),
		)
	}

	// Update DMA logo sprites - synchronized movement (all move together)
	g.ctrSprite += 0.02 * speed

	// Base movement for all sprites (synchronized)
	baseX := 100*math.Sin(g.ctrSprite*1.35+1.25) + 100*math.Sin(g.ctrSprite*1.86+0.54)
	baseY := 60*math.Cos(g.ctrSprite*1.72+0.23) + 60*math.Cos(g.ctrSprite*1.63+0.98)

	for i := 0; i < nbDMALogos; i++ {
		// 4x4 grid pattern
		row := i / 4
		col := i % 4

		// Base position centered on screen, avoiding top banner (72px height)
		centerX := float64(screenWidth) / 2
		centerY := 72 + float64(screenHeight-72)/2 // Below banner, centered in remaining space

		// Grid offsets - spread to occupy the screen (4x4 grid)
		offsetX := (float64(col) - 1.5) * 200 // Centered with 4 columns
		offsetY := (float64(row) - 1.5) * 140 // Centered with 4 rows

		// Apply synchronized movement
		g.dmaSprites[i].x = centerX + offsetX + baseX
		g.dmaSprites[i].y = centerY + offsetY + baseY
	}

	// Update rotozoom
	g.posXi += 0.008 * speed
	g.posZi += 0.003 * speed
	g.posRi += 0.005 * speed

	// Update title logo (oscillating movement like viva_tcb)
	if g.hold >= 1 {
		g.hold--
	}
	if g.hold <= 0 {
		g.logoX += 0.0125 * speed // Moves from right to left and back
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)

	if g.state == gameStateIntro {
		g.drawIntro(g.mainCanvas)
	} else {
		g.drawDemo()
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64((screen.Bounds().Dx()-screenWidth)/2), 0)
	screen.DrawImage(g.mainCanvas, op)
	g.drawControls(screen)
}

func (g *Game) drawIntro(screen *ebiten.Image) {
	screen.Fill(color.Black)

	if g.crtShader != nil {
		g.introStrip.Clear()
		g.introStrip.DrawImage(g.surfScroll1, nil)

		op := &ebiten.DrawRectShaderOptions{}
		op.Images[0] = g.introStrip
		op.GeoM.Translate(0, float64(screenHeight/2-int(fontHeight*2)/2))

		screen.DrawRectShader(screenWidth, int(fontHeight*2), g.crtShader, op)
	} else {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(0, float64(screenHeight/2-int(fontHeight*2)/2))
		screen.DrawImage(g.surfScroll1, op)
	}
}

func (g *Game) drawDemo() {
	g.mainCanvas.Fill(color.RGBA{0x00, 0x00, 0x30, 0xFF})

	// Order of rendering (back to front):
	// 1. Rotozoom background (furthest back)
	g.drawRotozoom(g.mainCanvas)

	// 2. Scrolling text with distortion
	g.drawScrollText(g.mainCanvas)

	// 3. DMA logo sprites (9 logos grid)
	g.drawDMALogos(g.mainCanvas)

	// 4. 3D cubes (on top of logos)
	g.draw3DCubes(g.mainCanvas)

	// 5. Title logo with copper bars on top (always on top)
	g.drawTitleWithCopperbars(g.mainCanvas)

}

func (g *Game) drawRotozoom(dst *ebiten.Image) {
	if g.cocoImg == nil {
		return
	}

	zoom := 0.5 + math.Abs(math.Sin(g.posZi)*2.5)
	rot := 360.0 / 4.0 * math.Cos(g.posRi*4-math.Cos(g.posRi-0.01)) * 0.3 * math.Pi / 180

	oscX := (float64(screenWidth) / 4) * math.Cos(g.posXi*4-math.Cos(g.posXi-0.1))
	oscY := (float64(screenHeight) / 2.7) * -math.Sin(g.posXi*2.3-math.Cos(g.posXi-0.1))

	centerX := float64(screenWidth)/2 + oscX
	centerY := float64(screenHeight)/2 + oscY

	cosRot, sinRot := math.Cos(rot), math.Sin(rot)
	setDestination := func(index int, x, y float64) {
		x -= float64(rotoWidth) / 2
		y -= float64(rotoHeight) / 2
		g.rotoVertices[index].DstX = float32((x*cosRot-y*sinRot)*zoom + centerX)
		g.rotoVertices[index].DstY = float32((x*sinRot+y*cosRot)*zoom + centerY)
	}
	setDestination(0, 0, 0)
	setDestination(1, rotoWidth, 0)
	setDestination(2, 0, rotoHeight)
	setDestination(3, rotoWidth, rotoHeight)

	op := &ebiten.DrawTrianglesOptions{Address: ebiten.AddressRepeat}
	dst.DrawTriangles(g.rotoVertices[:], []uint16{0, 1, 2, 1, 2, 3}, g.cocoImg, op)
}

func (g *Game) drawDMALogos(dst *ebiten.Image) {
	if g.dmaLogoImg == nil {
		return
	}

	logoW := float64(g.dmaLogoImg.Bounds().Dx())
	logoH := float64(g.dmaLogoImg.Bounds().Dy())
	scale := 0.5 // Larger logos (increased from 0.35)

	for _, sprite := range g.dmaSprites {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(-logoW/2, -logoH/2)
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(sprite.x, sprite.y)
		op.ColorScale.Scale(1, 1, 1, 0.6) // Semi-transparent
		dst.DrawImage(g.dmaLogoImg, op)
	}
}

func (g *Game) drawScrollText(dst *ebiten.Image) {
	g.frontWavePos = int(g.demoTime * 15)
	decalX := g.scrollOffset(g.frontWavePos)
	g.advanceScrollLetter(decalX)
	if g.displayedLetter != g.letterNum {
		g.displayText(g.letterNum)
		g.displayedLetter = g.letterNum
	}
	bounce := int(math.Floor(18 * math.Abs(math.Sin(g.demoTime*.1))))
	width := g.scrollSurf.Bounds().Dx()
	height := int(fontHeight * 3.0)
	baseY := 72
	lines := screenHeight - 72
	if g.stripBatch == nil {
		g.stripBatch = composite.NewQuadBatch(lines)
		g.stripBatch.AlternateDiagonal = true
		g.stripBatch.Options.Address = ebiten.AddressRepeat
	}
	g.stripBatch.Begin(dst, g.scrollSurf)
	for line := 0; line < lines; line++ {
		sourceLine := line / 3
		raw := g.getWave(g.frontWavePos+sourceLine) - g.letterDecal
		sy := (((sourceLine+bounce)%fontHeight)*3 + line%3) % height
		dx := 0
		sx := raw % width
		if raw < 0 {
			dx = min(-raw, screenWidth)
			sx = 0
		}
		w := screenWidth - dx
		g.stripBatch.Rect(image.Rect(sx, sy, sx+w, sy+1), float32(dx), float32(baseY+line), float32(w), 1)
	}
	g.stripBatch.Flush()
}

func (g *Game) displayText(letterOffset int) {
	g.scrollSurf.Clear()
	if g.scrollRenderer == nil {
		glyphs := make([]scrolling.Glyph, len(g.scrollTextRunes))
		for i, r := range g.scrollTextRunes {
			if letter, ok := g.letterData[r]; ok {
				glyphs[i] = scrolling.Glyph{Image: letter.glyph, Advance: float64(letter.width)}
			} else {
				glyphs[i] = scrolling.Glyph{Advance: 32}
			}
		}
		var err error
		g.scrollRenderer, err = scrolling.New(scrolling.Config{Glyphs: glyphs})
		if err != nil {
			panic(err)
		}
	}
	state := g.scrollRenderer.Window(letterOffset, float64(g.scrollSurf.Bounds().Dx())/3)
	state.ScaleX = 3
	state.ScaleY = 3
	state.X *= 3
	g.scrollRenderer.DrawAt(g.scrollSurf, state)
}

func (g *Game) draw3DCubes(dst *ebiten.Image) {
	// Draw each cube at its position
	for i := 0; i < nbCubes; i++ {
		// Calculate position
		xPos := float64((screenWidth-40)/2) + (float64((screenWidth-40)/2) * math.Sin(g.spritePos[i]))
		yPos := float64(screenHeight)/2 + (84 * math.Cos(g.spritePos[i]*2.5)) // Centered vertically

		// Draw the 3D cube
		g.cubes[i].Draw(dst, g.cubeFillImage, xPos, yPos)
	}
}

func (g *Game) drawTitleWithCopperbars(dst *ebiten.Image) {
	if g.titleImg == nil {
		return
	}

	// Fill title canvas with black (banner background)
	g.titleCanvas.Fill(color.Black)

	// Draw copper bars FIRST (background) - they will show through black/transparent areas of logo
	g.drawCopperBars(g.titleCanvas)

	// Draw title logo on top with oscillating movement
	// Oscillating horizontal movement that goes off-screen
	titleX := 64 + float64(screenWidth)*math.Cos(g.logoX)

	// Scale logo to fill the entire banner height (72px)
	titleH := float64(g.titleImg.Bounds().Dy())
	scaleY := 72.0 / titleH

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1.0, scaleY)
	op.GeoM.Translate(titleX, 0)
	g.titleCanvas.DrawImage(g.titleImg, op)

	// Draw title canvas at top of screen
	dst.DrawImage(g.titleCanvas, nil)
}

func (g *Game) drawCopperBars(dst *ebiten.Image) {
	if g.barsImg == nil {
		return
	}

	if g.barStrips[0] == nil {
		return
	}

	// Draw copper bars filling the banner height (72px)
	cc := 0
	for i := 0; i < 36; i++ { // 36 bars * 2 pixels = 72 pixels height
		// Calculate sine positions for animation
		val2 := (int(g.cnt) + i*7) & 0x3ff
		val := g.copperSin[val2]
		val2 = (int(g.cnt2) + i*10) & 0x3ff
		val += g.copperSin[val2]
		val += 60

		// Position
		xPos := val >> 1
		yPos := i << 1 // i * 2
		height := 72 - yPos

		if height > 0 && yPos < 72 {
			op := &ebiten.DrawImageOptions{}

			// Scale to stretch the 2 pixels
			scaleY := float64(height) / 2.0

			op.GeoM.Scale(1, scaleY)
			op.GeoM.Translate(float64(xPos), float64(yPos))

			dst.DrawImage(g.barStrips[cc/2], op)
		}

		// Cycle through the bars
		cc += 2
		if cc >= 20 {
			cc = 0
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	if outsideHeight <= 0 {
		return screenWidth, screenHeight
	}

	logicalWidth := outsideWidth * screenHeight / outsideHeight
	if logicalWidth < screenWidth {
		logicalWidth = screenWidth
	}
	g.logicalWidth = logicalWidth
	return logicalWidth, screenHeight
}
