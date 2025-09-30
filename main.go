package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"io"
	"log"
	"math"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"github.com/olivierh59500/ym-player/pkg/stsound"
)

const (
	screenWidth  = 800
	screenHeight = 600

	// Constants for effects
	nbCubes      = 12
	scrollSpeed  = 4.0
	fontHeight   = 36
	scrollFontCharSize = 32

	// Canvas sizes
	canvasWidth  = screenWidth * 8
	canvasHeight = screenHeight * 8

	sampleRate = 44100
)

// Embedded assets
//
//go:embed assets/title.png
var titleImgData []byte

//go:embed assets/bars.png
var barsImgData []byte

//go:embed assets/coco.png
var cocoImgData []byte

//go:embed assets/small-dma-jelly.png
var dmaLogoImgData []byte

//go:embed assets/font.png
var fontImgData []byte

//go:embed assets/mindbomb.ym
var musicData []byte

// Wave types for distortion
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

// YMPlayer wraps the YM player for Ebiten audio
type YMPlayer struct {
	player       *stsound.StSound
	sampleRate   int
	buffer       []int16
	mutex        sync.Mutex
	position     int64
	totalSamples int64
	loop         bool
	volume       float64
}

// NewYMPlayer creates a new YM player instance
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
		player:       player,
		sampleRate:   sampleRate,
		buffer:       make([]int16, 4096),
		totalSamples: totalSamples,
		loop:         loop,
		volume:       0.7,
	}, nil
}

func (y *YMPlayer) Read(p []byte) (n int, err error) {
	y.mutex.Lock()
	defer y.mutex.Unlock()

	samplesNeeded := len(p) / 4
	outBuffer := make([]int16, samplesNeeded*2)

	processed := 0
	for processed < samplesNeeded {
		chunkSize := samplesNeeded - processed
		if chunkSize > len(y.buffer) {
			chunkSize = len(y.buffer)
		}

		if !y.player.Compute(y.buffer[:chunkSize], chunkSize) {
			if !y.loop {
				for i := processed * 2; i < len(outBuffer); i++ {
					outBuffer[i] = 0
				}
				err = io.EOF
				break
			}
		}

		for i := 0; i < chunkSize; i++ {
			sample := int16(float64(y.buffer[i]) * y.volume)
			outBuffer[(processed+i)*2] = sample
			outBuffer[(processed+i)*2+1] = sample
		}

		processed += chunkSize
		y.position += int64(chunkSize)
	}

	buf := make([]byte, 0, len(outBuffer)*2)
	for _, sample := range outBuffer {
		buf = append(buf, byte(sample), byte(sample>>8))
	}

	copy(p, buf)
	n = len(buf)
	if n > len(p) {
		n = len(p)
	}

	return n, err
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
		newPos = y.totalSamples + offset
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}

	if newPos < 0 {
		newPos = 0
	}
	if newPos > y.totalSamples {
		newPos = y.totalSamples
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

func (y *YMPlayer) SetVolume(vol float64) {
	y.mutex.Lock()
	defer y.mutex.Unlock()
	y.volume = vol
}

// Letter for font rendering
type Letter struct {
	x, y  int
	width int
}

// Cube3D represents a 3D cube
type Cube3D struct {
	angleX float64
	angleY float64
	angleZ float64
	size   float64
}

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
	perspective := 200.0
	factor := perspective / (perspective + z)
	return x * factor, y * factor
}

// Draw draws the 3D cube at the specified position
func (c *Cube3D) Draw(screen *ebiten.Image, centerX, centerY float64) {
	// Define cube vertices in 3D space
	vertices := [][3]float64{
		{-c.size / 2, -c.size / 2, -c.size / 2}, // 0
		{c.size / 2, -c.size / 2, -c.size / 2},  // 1
		{c.size / 2, c.size / 2, -c.size / 2},   // 2
		{-c.size / 2, c.size / 2, -c.size / 2},  // 3
		{-c.size / 2, -c.size / 2, c.size / 2},  // 4
		{c.size / 2, -c.size / 2, c.size / 2},   // 5
		{c.size / 2, c.size / 2, c.size / 2},    // 6
		{-c.size / 2, c.size / 2, c.size / 2},   // 7
	}

	// Define cube faces (indices into vertices array)
	faces := [][4]int{
		{0, 1, 2, 3}, // Back
		{4, 5, 6, 7}, // Front
		{0, 1, 5, 4}, // Bottom
		{2, 3, 7, 6}, // Top
		{0, 3, 7, 4}, // Left
		{1, 2, 6, 5}, // Right
	}

	// Define face colors (pink/magenta tones)
	faceColors := []color.Color{
		color.RGBA{255, 80, 160, 255},  // Hot pink
		color.RGBA{255, 120, 200, 255}, // Light pink
		color.RGBA{200, 60, 140, 255},  // Dark pink
		color.RGBA{255, 100, 180, 255}, // Medium pink
		color.RGBA{220, 80, 160, 255},  // Rose
		color.RGBA{255, 140, 200, 255}, // Pale pink
	}

	// Rotate vertices
	rotated := make([][3]float64, len(vertices))
	for i, v := range vertices {
		x, y, z := v[0], v[1], v[2]

		// Rotate around X axis
		cosX, sinX := math.Cos(c.angleX), math.Sin(c.angleX)
		y1 := y*cosX - z*sinX
		z1 := y*sinX + z*cosX
		y, z = y1, z1

		// Rotate around Y axis
		cosY, sinY := math.Cos(c.angleY), math.Sin(c.angleY)
		x1 := x*cosY + z*sinY
		z2 := -x*sinY + z*cosY
		x, z = x1, z2

		// Rotate around Z axis
		cosZ, sinZ := math.Cos(c.angleZ), math.Sin(c.angleZ)
		x2 := x*cosZ - y*sinZ
		y2 := x*sinZ + y*cosZ
		x, y = x2, y2

		rotated[i] = [3]float64{x, y, z}
	}

	// Calculate face depths for sorting
	type faceDepth struct {
		index int
		depth float64
	}
	depths := make([]faceDepth, len(faces))

	for i, face := range faces {
		// Calculate center of face
		centerZ := 0.0
		for _, vi := range face {
			centerZ += rotated[vi][2]
		}
		depths[i] = faceDepth{i, centerZ / 4}
	}

	// Sort faces by depth (back to front)
	for i := 0; i < len(depths)-1; i++ {
		for j := i + 1; j < len(depths); j++ {
			if depths[i].depth > depths[j].depth {
				depths[i], depths[j] = depths[j], depths[i]
			}
		}
	}

	// Draw faces
	for _, fd := range depths {
		face := faces[fd.index]
		faceColor := faceColors[fd.index]

		// Project vertices to 2D
		points := make([]float64, 0, 8)
		for _, vi := range face {
			v := rotated[vi]
			x2d, y2d := project3D(v[0], v[1], v[2])
			points = append(points, centerX+x2d, centerY+y2d)
		}

		// Draw filled polygon
		drawPolygon(screen, points, faceColor)

		// Draw edges with darker color for better visibility
		edgeColor := color.RGBA{
			uint8(faceColor.(color.RGBA).R * 3 / 4),
			uint8(faceColor.(color.RGBA).G * 3 / 4),
			uint8(faceColor.(color.RGBA).B * 3 / 4),
			255,
		}
		for i := 0; i < 4; i++ {
			j := (i + 1) % 4
			vector.StrokeLine(screen,
				float32(points[i*2]), float32(points[i*2+1]),
				float32(points[j*2]), float32(points[j*2+1]),
				1, edgeColor, false)
		}
	}
}

// drawPolygon draws a filled polygon
func drawPolygon(screen *ebiten.Image, points []float64, fillColor color.Color) {
	if len(points) < 6 {
		return
	}

	// Draw as a filled rectangle using vector
	if len(points) >= 8 {
		// Draw filled quadrilateral as two triangles
		// Triangle 1: points 0, 1, 2
		drawTriangle(screen,
			float32(points[0]), float32(points[1]),
			float32(points[2]), float32(points[3]),
			float32(points[4]), float32(points[5]),
			fillColor)

		// Triangle 2: points 0, 2, 3
		drawTriangle(screen,
			float32(points[0]), float32(points[1]),
			float32(points[4]), float32(points[5]),
			float32(points[6]), float32(points[7]),
			fillColor)
	}
}

// drawTriangle draws a filled triangle
func drawTriangle(screen *ebiten.Image, x1, y1, x2, y2, x3, y3 float32, clr color.Color) {
	// Sort vertices by Y coordinate
	if y1 > y2 {
		x1, y1, x2, y2 = x2, y2, x1, y1
	}
	if y1 > y3 {
		x1, y1, x3, y3 = x3, y3, x1, y1
	}
	if y2 > y3 {
		x2, y2, x3, y3 = x3, y3, x2, y2
	}

	// Draw horizontal lines to fill the triangle
	for y := y1; y <= y3; y++ {
		var xStart, xEnd float32

		if y < y2 {
			// Upper part of triangle
			if y2-y1 > 0 {
				t := (y - y1) / (y2 - y1)
				x12 := x1 + (x2-x1)*t
				t13 := (y - y1) / (y3 - y1)
				x13 := x1 + (x3-x1)*t13
				xStart, xEnd = x12, x13
			}
		} else {
			// Lower part of triangle
			if y3-y2 > 0 && y3-y1 > 0 {
				t := (y - y2) / (y3 - y2)
				x23 := x2 + (x3-x2)*t
				t13 := (y - y1) / (y3 - y1)
				x13 := x1 + (x3-x1)*t13
				xStart, xEnd = x23, x13
			}
		}

		if xStart > xEnd {
			xStart, xEnd = xEnd, xStart
		}

		vector.StrokeLine(screen, xStart, y, xEnd, y, 1, clr, false)
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
	// Images
	titleImg    *ebiten.Image
	barsImg     *ebiten.Image
	cocoImg     *ebiten.Image
	dmaLogoImg  *ebiten.Image
	fontImg     *ebiten.Image

	// Canvases
	introCanvas *ebiten.Image
	mainCanvas  *ebiten.Image
	cocoCanvas  *ebiten.Image
	scrollSurf  *ebiten.Image
	titleCanvas *ebiten.Image

	// Audio
	audioContext *audio.Context
	audioPlayer  *audio.Player
	ymPlayer     *YMPlayer

	// State
	state          string // "intro" or "demo"
	introComplete  bool
	iteration      int

	// Intro scrolling
	introX         int
	introLetter    int
	introTile      int
	introSpeed     int
	introText      string
	surfScroll1    *ebiten.Image
	surfScroll2    *ebiten.Image

	// Font data
	letterData     map[rune]*Letter

	// CRT Shader
	crtShader      *ebiten.Shader

	// Demo effects
	// Copper bars
	cnt            int
	cnt2           int
	copperSin      []int

	// 3D Cubes
	cubes          []*Cube3D
	spritePos      []float64

	// DMA logo sprites (9 logos in 3x3 grid)
	dmaSprites     [9]DMASprite
	ctrSprite      float64

	// Scrolling text (megatwist style)
	frontWavePos   int
	letterNum      int
	letterDecal    int
	curves         [][]int
	frontMainWave  []int
	position       []int
	scrollText     string
	scrollTextRunes []rune

	// Rotozoom
	posXi          float64
	posZi          float64
	posRi          float64

	// Title logo animation
	logoX          float64
	hold           int
	rasterY1       float64
	rasterY2       float64

	// Speed control
	speedMultiplier float64

	// VBL counter
	vbl            int
}

type DMASprite struct {
	x, y float64
}

func NewGame() *Game {
	g := &Game{
		state:           "intro",
		introX:          -1,
		introLetter:     -1,
		introTile:       -1,
		introSpeed:      8,
		letterData:      make(map[rune]*Letter),
		spritePos:       make([]float64, nbCubes),
		speedMultiplier: 1.0,
		logoX:           0.5, // Center the logo (0.5 = centered)
		hold:            0, // Start immediately
	}

	// Init intro text
	spc := "     "
	g.introText = spc + spc + "IF YOU THINK THIS IS ALL, YOU'RE SO WRONG..." + spc

	// Init demo scroll text
	g.scrollText = spc + spc + "WELCOME TO THE COCO IS THE BEST DEMO! " + spc +
		"THIS DEMO COMBINES THE BEST EFFECTS FROM VARIOUS ATARI ST DEMOS. " + spc +
		"GREETINGS TO ALL DEMOSCENE LOVERS! " + spc + spc
	g.scrollTextRunes = []rune(g.scrollText)

	// Load images
	g.loadImages()

	// Create canvases
	g.introCanvas = ebiten.NewImage(screenWidth, screenHeight)
	g.mainCanvas = ebiten.NewImage(screenWidth, screenHeight)
	g.cocoCanvas = ebiten.NewImage(canvasWidth, canvasHeight)
	g.surfScroll1 = ebiten.NewImage(screenWidth+96, int(fontHeight*2))
	g.surfScroll2 = ebiten.NewImage(screenWidth+96, int(fontHeight*2))
	g.scrollSurf = ebiten.NewImage(int(float64(screenWidth)*2.0), int(fontHeight*3))
	g.titleCanvas = ebiten.NewImage(screenWidth, 72)

	// Create rotozoom canvas with tiled Coco image
	if g.cocoImg != nil {
		cocoW := g.cocoImg.Bounds().Dx()
		cocoH := g.cocoImg.Bounds().Dy()
		for y := 0; y < canvasHeight; y += cocoH {
			for x := 0; x < canvasWidth; x += cocoW {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(x), float64(y))
				g.cocoCanvas.DrawImage(g.cocoImg, op)
			}
		}
	}

	// Init font
	g.initFontData()

	// Init 3D cubes
	g.cubes = make([]*Cube3D, nbCubes)
	for i := 0; i < nbCubes; i++ {
		g.cubes[i] = NewCube3D(40.0) // Size of cube
		// Set initial position offset for each cube
		g.spritePos[i] = float64(0.15) * float64(i+1)
		// Set different initial rotations
		g.cubes[i].angleX = float64(i) * 0.3
		g.cubes[i].angleY = float64(i) * 0.2
		g.cubes[i].angleZ = float64(i) * 0.1
	}

	// Init wave curves for scrolling
	g.curves = make([][]int, 8)
	g.createCurves()
	g.precalcPosition()
	g.precalcMainWave()

	// Init audio
	g.initAudio()

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
		g.letterData[d.char] = &Letter{x: d.x, y: d.y, width: d.width}
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
	if i > 0 && i <= len(g.position) {
		return g.getSum(g.position, i-1, 0)
	}
	return 0
}

func (g *Game) getLetter(pos int) rune {
	if len(g.scrollTextRunes) == 0 {
		return ' '
	}
	return g.scrollTextRunes[pos%len(g.scrollTextRunes)]
}

func (g *Game) getIntroLetter(pos int) rune {
	runes := []rune(g.introText)
	if len(runes) == 0 {
		return ' '
	}
	return runes[pos%len(runes)]
}

func (g *Game) Update() error {
	// Volume control
	if g.ymPlayer != nil {
		if ebiten.IsKeyPressed(ebiten.KeyUp) {
			vol := g.ymPlayer.GetVolume() + 0.01
			if vol > 1.0 {
				vol = 1.0
			}
			g.ymPlayer.SetVolume(vol)
		}
		if ebiten.IsKeyPressed(ebiten.KeyDown) {
			vol := g.ymPlayer.GetVolume() - 0.01
			if vol < 0 {
				vol = 0
			}
			g.ymPlayer.SetVolume(vol)
		}
	}

	// Speed control
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) {
		g.speedMultiplier += 0.1
		if g.speedMultiplier > 2.0 {
			g.speedMultiplier = 2.0
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) {
		g.speedMultiplier -= 0.1
		if g.speedMultiplier < 0.5 {
			g.speedMultiplier = 0.5
		}
	}

	if g.state == "intro" {
		g.updateIntro()
	} else {
		g.updateDemo()
	}

	g.vbl++
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
		runes := []rune(g.introText)
		if g.introLetter >= len(runes) {
			g.introComplete = true
			g.state = "demo"
			g.iteration = 0
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
	srcRect := image.Rect(g.introSpeed, 0, g.surfScroll1.Bounds().Dx(), int(fontHeight*2))
	g.surfScroll2.DrawImage(g.surfScroll1.SubImage(srcRect).(*ebiten.Image), nil)

	g.surfScroll1.Clear()
	g.surfScroll1.DrawImage(g.surfScroll2, nil)

	// Draw new letter
	char := g.getIntroLetter(g.introTile)
	if letter, ok := g.letterData[char]; ok {
		srcRect := image.Rect(letter.x, letter.y, letter.x+letter.width, letter.y+fontHeight)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2.0, 2.0)
		op.GeoM.Translate(float64(screenWidth+g.introX), 0)
		g.surfScroll1.DrawImage(g.fontImg.SubImage(srcRect).(*ebiten.Image), op)
	}
}

func (g *Game) updateDemo() {
	g.iteration++

	// Update copper bars
	g.cnt = (g.cnt + 3) & 0x3ff
	g.cnt2 = (g.cnt2 - 5) & 0x3ff

	// Update 3D cubes
	for i := 0; i < nbCubes; i++ {
		g.spritePos[i] += 0.04 * g.speedMultiplier
		g.cubes[i].Rotate(
			0.02*g.speedMultiplier*(1+float64(i)*0.1),
			0.03*g.speedMultiplier*(1+float64(i)*0.15),
			0.01*g.speedMultiplier*(1+float64(i)*0.05),
		)
	}

	// Update DMA logo sprites - synchronized movement (all move together)
	g.ctrSprite += 0.02

	// Base movement for all sprites (synchronized)
	baseX := 100 * math.Sin(g.ctrSprite*1.35+1.25) + 100 * math.Sin(g.ctrSprite*1.86+0.54)
	baseY := 60 * math.Cos(g.ctrSprite*1.72+0.23) + 60 * math.Cos(g.ctrSprite*1.63+0.98)

	for i := 0; i < 9; i++ {
		// 3x3 grid pattern
		row := i / 3
		col := i % 3

		// Base position centered on screen, avoiding top banner (72px height)
		centerX := float64(screenWidth) / 2
		centerY := 72 + float64(screenHeight-72)/2 // Below banner, centered in remaining space

		// Grid offsets - spread to occupy the screen (3x3 grid)
		offsetX := (float64(col) - 1) * 250 // Spread horizontally (increased from 220)
		offsetY := (float64(row) - 1) * 180 // Spread vertically (increased from 160)

		// Apply synchronized movement
		g.dmaSprites[i].x = centerX + offsetX + baseX
		g.dmaSprites[i].y = centerY + offsetY + baseY
	}

	// Update rotozoom
	g.posXi += 0.008
	g.posZi += 0.003
	g.posRi += 0.005

	// Update title logo (oscillating movement like viva_tcb)
	if g.hold >= 1 {
		g.hold--
	}
	if g.hold <= 0 {
		g.logoX += 0.0125 // Moves from right to left and back
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)

	if g.state == "intro" {
		g.drawIntro(screen)
	} else {
		g.drawDemo(screen)
	}
}

func (g *Game) drawIntro(screen *ebiten.Image) {
	g.introCanvas.Fill(color.Black)

	if g.crtShader != nil {
		tmpImg := ebiten.NewImage(screenWidth, int(fontHeight*2))
		tmpImg.Clear()
		tmpImg.DrawImage(g.surfScroll1, nil)

		op := &ebiten.DrawRectShaderOptions{}
		op.Images[0] = tmpImg
		op.GeoM.Translate(0, float64(screenHeight/2-int(fontHeight*2)/2))

		screen.DrawRectShader(screenWidth, int(fontHeight*2), g.crtShader, op)
	} else {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(0, float64(screenHeight/2-int(fontHeight*2)/2))
		screen.DrawImage(g.surfScroll1, op)
	}
}

func (g *Game) drawDemo(screen *ebiten.Image) {
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

	screen.DrawImage(g.mainCanvas, nil)
}

func (g *Game) drawRotozoom(dst *ebiten.Image) {
	zoom := 0.5 + math.Abs(math.Sin(g.posZi)*2.5)
	rot := 360.0 / 4.0 * math.Cos(g.posRi*4-math.Cos(g.posRi-0.01)) * 0.3 * math.Pi / 180

	oscX := (float64(screenWidth) / 4) * math.Cos(g.posXi*4-math.Cos(g.posXi-0.1))
	oscY := (float64(screenHeight) / 2.7) * -math.Sin(g.posXi*2.3-math.Cos(g.posXi-0.1))

	centerX := float64(screenWidth)/2 + oscX
	centerY := float64(screenHeight)/2 + oscY

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-float64(canvasWidth)/2, -float64(canvasHeight)/2)
	op.GeoM.Rotate(rot)
	op.GeoM.Scale(zoom, zoom)
	op.GeoM.Translate(centerX, centerY)
	op.ColorScale.Scale(0.5, 0.5, 0.5, 1.0) // Darken background
	dst.DrawImage(g.cocoCanvas, op)
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
	// Update wave position
	g.frontWavePos = g.iteration * 10

	// Calculate horizontal offset
	decalX := 999999999
	for ligne := 0; ligne < fontHeight; ligne++ {
		c := g.getWave(g.frontWavePos + ligne)
		if c < decalX {
			decalX = c
		}
	}

	if decalX < 0 {
		decalX = 0
	}

	// Calculate first visible letter
	i := 0
	dir := 0
	if decalX > g.letterDecal {
		dir = 1
	} else if decalX < g.letterDecal {
		dir = -1
	}

	for decalX < g.getPosition(g.letterNum+i) || g.getPosition(g.letterNum+i+1) <= decalX {
		i += dir
		if g.letterNum+i < 0 || g.letterNum+i >= len(g.position) {
			break
		}
	}
	g.letterNum += i
	if g.letterNum < 0 {
		g.letterNum = 0
	} else if g.letterNum >= len(g.position) {
		g.letterNum = len(g.position) - 1
	}
	g.letterDecal = g.getPosition(g.letterNum)

	// Render text to scroll surface
	g.displayText(g.letterNum)

	// Calculate bounce effect
	bounce := int(math.Floor(18.0 * math.Abs(math.Sin(float64(g.iteration)*0.1))))

	scrollWidth := g.scrollSurf.Bounds().Dx()
	scaledFontHeight := int(fontHeight * 3.0)

	// Render each line with distortion - cover full screen height (below banner)
	baseY := 72 // Start just below the banner
	totalLines := screenHeight - 72 // Total lines from banner to bottom
	for ligne := 0; ligne < totalLines; ligne++ {
		sourceFontLine := ligne / 3

		frontWave := g.getWave(g.frontWavePos + sourceFontLine)
		scrollXRaw := frontWave - g.letterDecal

		scaledLine := ((sourceFontLine+bounce)%fontHeight)*3 + (ligne % 3)

		if scaledLine >= scaledFontHeight {
			scaledLine = scaledLine % scaledFontHeight
		}

		if scrollXRaw < 0 {
			visibleWidth := screenWidth + scrollXRaw
			if visibleWidth > 0 {
				srcRect := image.Rect(0, scaledLine, minInt(visibleWidth, scrollWidth), scaledLine+1)
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(-scrollXRaw), float64(baseY+ligne))
				dst.DrawImage(g.scrollSurf.SubImage(srcRect).(*ebiten.Image), op)
			}
			continue
		}

		scrollX := scrollXRaw % scrollWidth
		if scrollX >= scrollWidth-screenWidth {
			width1 := scrollWidth - scrollX
			if width1 > 0 && width1 <= screenWidth {
				srcRect := image.Rect(scrollX, scaledLine, scrollWidth, scaledLine+1)
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(0, float64(baseY+ligne))
				dst.DrawImage(g.scrollSurf.SubImage(srcRect).(*ebiten.Image), op)
			}

			width2 := screenWidth - width1
			if width2 > 0 && width2 <= screenWidth {
				srcRect := image.Rect(0, scaledLine, width2, scaledLine+1)
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Translate(float64(width1), float64(baseY+ligne))
				dst.DrawImage(g.scrollSurf.SubImage(srcRect).(*ebiten.Image), op)
			}
		} else if scrollX+screenWidth <= scrollWidth {
			srcRect := image.Rect(scrollX, scaledLine, scrollX+screenWidth, scaledLine+1)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(0, float64(baseY+ligne))
			dst.DrawImage(g.scrollSurf.SubImage(srcRect).(*ebiten.Image), op)
		}
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (g *Game) displayText(letterOffset int) {
	g.scrollSurf.Clear()

	xPos := 0
	i := 0
	maxWidth := g.scrollSurf.Bounds().Dx() + 200*3

	for xPos < maxWidth {
		char := g.getLetter(i + letterOffset)
		if letter, ok := g.letterData[char]; ok {
			srcRect := image.Rect(letter.x, letter.y, letter.x+letter.width, letter.y+fontHeight)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Scale(3.0, 3.0)
			op.GeoM.Translate(float64(xPos), 0)
			g.scrollSurf.DrawImage(g.fontImg.SubImage(srcRect).(*ebiten.Image), op)
			xPos += int(float64(letter.width) * 3.0)
		}
		i++
	}
}

func (g *Game) draw3DCubes(dst *ebiten.Image) {
	// Draw each cube at its position
	for i := 0; i < nbCubes; i++ {
		// Calculate position
		xPos := float64((screenWidth-40)/2) + (float64((screenWidth-40)/2) * math.Sin(g.spritePos[i]))
		yPos := float64(screenHeight)/2 + (84 * math.Cos(g.spritePos[i]*2.5)) // Centered vertically

		// Draw the 3D cube
		g.cubes[i].Draw(dst, xPos, yPos)
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

	barsWidth, barsHeight := g.barsImg.Size()
	if barsHeight < 20 {
		return
	}

	// Draw copper bars filling the banner height (72px)
	cc := 0
	for i := 0; i < 36; i++ { // 36 bars * 2 pixels = 72 pixels height
		// Calculate sine positions for animation
		val2 := (g.cnt + i*7) & 0x3ff
		val := g.copperSin[val2]
		val2 = (g.cnt2 + i*10) & 0x3ff
		val += g.copperSin[val2]
		val += 60

		// Position
		xPos := val >> 1
		yPos := i << 1 // i * 2
		height := 72 - yPos

		if height > 0 && yPos < 72 {
			op := &ebiten.DrawImageOptions{}

			// Source rectangle: 2 pixels high from bars
			srcRect := image.Rect(0, cc, barsWidth, cc+2)
			if srcRect.Max.Y > barsHeight {
				srcRect.Max.Y = barsHeight
			}

			// Scale to stretch the 2 pixels
			scaleY := float64(height) / 2.0

			op.GeoM.Scale(1, scaleY)
			op.GeoM.Translate(float64(xPos), float64(yPos))

			dst.DrawImage(g.barsImg.SubImage(srcRect).(*ebiten.Image), op)
		}

		// Cycle through the bars
		cc += 2
		if cc >= 20 {
			cc = 0
		}
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("COCO IS THE BEST - DMA 2025")
	ebiten.SetWindowResizable(true)

	game := NewGame()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
