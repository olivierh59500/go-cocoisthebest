// Package coco implements the COCO IS THE BEST demo.
package coco

import (
	"bytes"
	"image"
	"image/color"

	"github.com/olivierh59500/democonstructionkit/composite"
	"github.com/olivierh59500/democonstructionkit/effects"
	"github.com/olivierh59500/democonstructionkit/geometry"
	"github.com/olivierh59500/democonstructionkit/presets"
	"github.com/olivierh59500/democonstructionkit/scrolling"
	"github.com/olivierh59500/democonstructionkit/sound"
	originalassets "github.com/olivierh59500/go-cocoisthebest"

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

type gameState uint8

const (
	gameStateIntro gameState = iota
	gameStateDemo
)

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
	musicStream  *sound.Stream
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
	fontAtlas *scrolling.Atlas

	// CRT Shader
	crtShader *ebiten.Shader

	// Demo effects
	// Copper bars
	cnt       float64
	cnt2      float64
	copperSin []int

	// 3D Cubes
	cubes     [nbCubes]*effects.SolidCube
	spritePos [nbCubes]float64
	cubeBatch *effects.SolidCubeBatch

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
	g.cubeBatch = effects.NewSolidCubeBatch(nbCubes)
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
		var err error
		g.cubes[i], err = effects.NewSolidCube(presets.CocoCube(40))
		if err != nil {
			panic(err)
		}
		// Set initial position offset for each cube
		g.spritePos[i] = float64(0.15) * float64(i+1)
		// Set different initial rotations
		g.cubes[i].Rotation = geometry.Vec3{X: float64(i) * .3, Y: float64(i) * .2, Z: float64(i) * .1}
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
	g.copperSin = presets.BilizirCopperOffsets()
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
	g.musicStream, err = sound.Open("music.ym", musicData, sound.Options{SampleRate: sampleRate, Loop: true, PCMFormat: sound.PCM16, Gain: 0.7})
	if err != nil {
		log.Printf("Failed to open music: %v", err)
		return
	}

	g.audioPlayer, err = g.audioContext.NewPlayer(g.musicStream)
	if err != nil {
		log.Printf("Failed to create audio player: %v", err)
		g.musicStream.Close()
		g.musicStream = nil
		return
	}

	// Music will start when transitioning from intro to demo phase
}

func (g *Game) initFontData() {
	var err error
	g.fontAtlas, err = presets.FontAtlas("go-cocoisthebest", g.fontImg)
	if err != nil {
		panic(err)
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
	curves, err := presets.RibbonCurves(1)
	if err != nil {
		panic(err)
	}
	copy(g.curves[:], curves[:8])
}

func (g *Game) precalcPosition() {
	count := 0
	g.position = []int{}

	for _, r := range g.scrollTextRunes {
		if _, letter, ok := g.fontAtlas.ExactGlyph(r); ok {
			count += int(float64(int(letter.Advance)) * 3.0)
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

	var err error
	g.frontMainWave, err = composite.JoinDeltaCurves(g.curves[:], frontMainWaveTable)
	if err != nil {
		panic(err)
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
	return composite.CumulativeAt(arr, index, decal)
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
			if _, letter, ok := g.fontAtlas.ExactGlyph(char); ok {
				g.introX += int(float64(int(letter.Advance)) * 2.0)
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
	if glyphImage, _, ok := g.fontAtlas.ExactGlyph(char); ok {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(2.0, 2.0)
		op.GeoM.Translate(float64(screenWidth+g.introX), 0)
		g.surfScroll1.DrawImage(glyphImage, op)
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
			if glyphImage, letter, ok := g.fontAtlas.ExactGlyph(r); ok {
				glyphs[i] = scrolling.Glyph{Image: glyphImage, Advance: float64(int(letter.Advance))}
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
	g.cubeBatch.Reset()
	// Draw each cube at its position
	for i := 0; i < nbCubes; i++ {
		// Calculate position
		xPos := float64((screenWidth-40)/2) + (float64((screenWidth-40)/2) * math.Sin(g.spritePos[i]))
		yPos := float64(screenHeight)/2 + (84 * math.Cos(g.spritePos[i]*2.5)) // Centered vertically

		// Draw the 3D cube
		g.cubeBatch.Add(g.cubes[i], xPos, yPos)
	}
	g.cubeBatch.Draw(dst)
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
