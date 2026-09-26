// Package coco implements the COCO IS THE BEST demo.
package coco

import (
	"bytes"
	"image"
	"image/color"

	kit "github.com/olivierh59500/democonstructionkit"
	"github.com/olivierh59500/democonstructionkit/composite"
	"github.com/olivierh59500/democonstructionkit/effects"
	"github.com/olivierh59500/democonstructionkit/geometry"
	"github.com/olivierh59500/democonstructionkit/presets"
	"github.com/olivierh59500/democonstructionkit/scrolling"
	"github.com/olivierh59500/democonstructionkit/sound"
	"github.com/olivierh59500/democonstructionkit/sprites"
	"github.com/olivierh59500/democonstructionkit/timeline"
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

// Game state
type Game struct {
	introScroll, mainScroll *scrolling.Scrolling
	// Images
	titleImg   *ebiten.Image
	barsImg    *ebiten.Image
	cocoImg    *ebiten.Image
	dmaLogoImg *ebiten.Image
	fontImg    *ebiten.Image

	// Canvases
	introStrip  *ebiten.Image
	mainCanvas  *ebiten.Image
	titleCanvas *ebiten.Image

	// Audio
	audioContext *audio.Context
	audioPlayer  *audio.Player
	musicStream  *sound.Stream
	audioReady   bool

	// State
	handoff  *timeline.IntroHandoff
	demoTime float64

	// Font data
	fontAtlas *scrolling.Atlas

	// CRT Shader
	crt *effects.CRTOverlay

	// Demo effects
	// Copper bars
	copper *composite.CopperBars

	// 3D Cubes
	cubes     [nbCubes]*effects.SolidCube
	spritePos [nbCubes]float64
	cubeBatch *effects.SolidCubeBatch

	// Shared image formation for the sixteen synchronized logos.
	logoFormation *sprites.Group

	// Rotozoom
	roto        *composite.RotozoomBackground
	rotoProgram *presets.VivaRotozoom

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

func NewGame() *Game {
	g := &Game{
		speedMultiplier: 1.0,
		logicalWidth:    screenWidth,
		logoX:           0.5, // Center the logo (0.5 = centered)
		hold:            0,   // Start immediately
	}

	var err error
	g.handoff, err = timeline.NewIntroHandoff(presets.ImmediateIntroHandoff())
	if err != nil {
		panic(err)
	}

	// Load images and construct the complete shared sprite formation.
	g.loadImages()
	if g.barsImg != nil {
		g.copper, err = composite.NewCopperBars(presets.BilizirCopperBars(g.barsImg, 72, composite.CopperImages, composite.SingleWrapClock))
		if err != nil {
			panic(err)
		}
	}
	g.logoFormation, err = sprites.NewGroup(presets.CocoLogoFormation(g.dmaLogoImg, screenWidth, screenHeight))
	if err != nil {
		panic(err)
	}

	// Create canvases
	g.introStrip = ebiten.NewImage(screenWidth, int(fontHeight*2))
	g.mainCanvas = ebiten.NewImageWithOptions(
		image.Rect(0, 0, screenWidth, screenHeight),
		&ebiten.NewImageOptions{Unmanaged: true},
	)
	g.titleCanvas = ebiten.NewImage(screenWidth, 72)
	g.cubeBatch = effects.NewSolidCubeBatch(nbCubes)

	if g.cocoImg != nil {
		g.rotoProgram, err = presets.NewVivaRotozoom(presets.CocoRotozoom(screenWidth, screenHeight))
		if err != nil {
			panic(err)
		}
		g.roto, err = composite.NewRotozoomBackground(composite.RotozoomBackgroundConfig{
			Image: g.cocoImg, Program: g.rotoProgram,
			SourceQuad: image.Pt(rotoWidth, rotoHeight),
		})
		if err != nil {
			panic(err)
		}
	}

	// Init font
	g.initFontData()

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

	// Bind the same complete text transports used by other productions.
	intro := presets.CocoIntroFeed(g.fontAtlas, introScrollText)
	g.introScroll, err = scrolling.New(scrolling.Config{Feed: &intro})
	if err != nil {
		panic(err)
	}
	main, err := presets.CocoScanlineScroll(g.fontAtlas, demoScrollText)
	if err != nil {
		panic(err)
	}
	g.mainScroll, err = scrolling.New(scrolling.Config{Scanline: &main})
	if err != nil {
		panic(err)
	}

	// Compile CRT shader
	g.crt, err = effects.NewCRTOverlay(presets.DMACRTOverlay())
	if err != nil {
		log.Printf("Failed to compile CRT shader: %v", err)
	}

	return g
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

func (g *Game) Update() error {
	// On Android, NewGame runs while the native library is loaded. Opening the
	// audio device there can block before the Activity has installed its view.
	if !g.audioReady {
		g.audioReady = true
		g.initAudio()
	}

	g.updateControls()

	if !g.handoff.Main() {
		return g.updateIntro()
	}
	g.handoff.Step(false)
	if err := g.updateDemo(); err != nil {
		return err
	}
	return g.mainScroll.Update(kit.Frame{Time: g.demoTime})
}

func (g *Game) updateIntro() error {
	if err := g.introScroll.Update(kit.Frame{}); err != nil {
		return err
	}
	g.handoff.Step(g.introScroll.Finished())
	if g.handoff.JustEntered() {
		g.demoTime = 0
		if g.handoff.CueReady() {
			if g.audioPlayer != nil && !g.audioPlayer.IsPlaying() {
				g.audioPlayer.Play()
			}
			g.handoff.MarkCue()
		}
	}
	return nil
}

func (g *Game) updateDemo() error {
	speed := g.speedMultiplier
	g.demoTime += speed

	// Update copper bars
	if g.copper != nil {
		if err := g.copper.SetSpeed(speed); err != nil {
			return err
		}
		if err := g.copper.Update(kit.Frame{}); err != nil {
			return err
		}
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

	// The group owns the common harmonics, grid positions and sprite poses.
	if err := g.logoFormation.Advance(.02 * speed); err != nil {
		return err
	}

	if g.roto != nil {
		if err := g.rotoProgram.SetSpeedMultiplier(speed); err != nil {
			return err
		}
		if err := g.roto.Update(kit.Frame{}); err != nil {
			return err
		}
	}

	// Update title logo (oscillating movement like viva_tcb)
	if g.hold >= 1 {
		g.hold--
	}
	if g.hold <= 0 {
		g.logoX += 0.0125 * speed // Moves from right to left and back
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(color.Black)

	if !g.handoff.Main() {
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
	if g.crt != nil {
		g.introStrip.Clear()
		g.introScroll.Draw(g.introStrip)
		g.crt.DrawAt(screen, g.introStrip, 0, float64(screenHeight/2-int(fontHeight*2)/2))
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, float64(screenHeight/2-int(fontHeight*2)/2))
	screen.DrawImage(g.introScroll.Image(), op)
}

func (g *Game) drawDemo() {
	g.mainCanvas.Fill(color.RGBA{0x00, 0x00, 0x30, 0xFF})

	// Order of rendering (back to front):
	// 1. Rotozoom background (furthest back)
	if g.roto != nil {
		g.roto.Draw(g.mainCanvas)
	}

	// 2. Scrolling text with distortion
	g.mainScroll.Draw(g.mainCanvas)

	// 3. DMA logo sprites (9 logos grid)
	g.logoFormation.Draw(g.mainCanvas)

	// 4. 3D cubes (on top of logos)
	g.draw3DCubes(g.mainCanvas)

	// 5. Title logo with copper bars on top (always on top)
	g.drawTitleWithCopperbars(g.mainCanvas)

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
	if g.copper != nil {
		g.copper.Draw(g.titleCanvas)
	}

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
