// Package coco implements the COCO IS THE BEST demo.
package coco

import (
	"bytes"
	"image"
	"image/color"

	kit "github.com/olivierh59500/democonstructionkit"
	"github.com/olivierh59500/democonstructionkit/composite"
	"github.com/olivierh59500/democonstructionkit/effects"
	"github.com/olivierh59500/democonstructionkit/motion"
	"github.com/olivierh59500/democonstructionkit/presets"
	"github.com/olivierh59500/democonstructionkit/scrolling"
	"github.com/olivierh59500/democonstructionkit/sound"
	"github.com/olivierh59500/democonstructionkit/sprites"
	"github.com/olivierh59500/democonstructionkit/timeline"
	originalassets "github.com/olivierh59500/go-cocoisthebest"

	_ "image/png"
	"log"

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
	introStrip *ebiten.Image
	mainCanvas *ebiten.Image

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

	// 3D cube procession
	cubeTrain *effects.SolidCubeTrain

	// Shared image formation for the sixteen synchronized logos.
	logoFormation *sprites.Group

	// Rotozoom
	roto        *composite.RotozoomBackground
	rotoProgram *presets.VivaRotozoom

	// The title band composes copper bars and a moving logo in one surface.
	titleLayer  *composite.SurfaceLayer
	titleMotion *motion.WaveClock
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
	g.cubeTrain, err = effects.NewSolidCubeTrain(presets.CocoCubeTrain(screenWidth, screenHeight, 40, nbCubes))
	if err != nil {
		panic(err)
	}
	g.titleMotion, err = motion.NewWaveClock(presets.CocoTitleMotion(screenWidth))
	if err != nil {
		panic(err)
	}
	if g.titleImg != nil {
		var sources []kit.Effect
		if g.copper != nil {
			sources = []kit.Effect{g.copper}
		}
		g.titleLayer, err = composite.NewSurfaceLayer(composite.SurfaceLayerConfig{
			Width: screenWidth, Height: 72, Background: color.Black,
			Sources: sources,
			Passes: []composite.SurfaceImagePass{{
				Image: g.titleImg, X: g.titleMotion.At(0),
				ScaleY: 72.0 / float64(g.titleImg.Bounds().Dy()),
			}},
			Outputs: []composite.SurfaceOutput{{}},
		})
		if err != nil {
			panic(err)
		}
	}

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
	}
	if g.titleLayer != nil {
		if err := g.titleLayer.Update(kit.Frame{}); err != nil {
			return err
		}
	} else if g.copper != nil {
		if err := g.copper.Update(kit.Frame{}); err != nil {
			return err
		}
	}

	if err := g.cubeTrain.SetSpeed(speed); err != nil {
		return err
	}
	if err := g.cubeTrain.Update(kit.Frame{}); err != nil {
		return err
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

	if err := g.titleMotion.SetStep(.0125 * speed); err != nil {
		return err
	}
	g.titleMotion.Step()
	if g.titleLayer != nil {
		if err := g.titleLayer.SetPassPosition(0, g.titleMotion.At(0), 0); err != nil {
			return err
		}
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

	// 4. The batched 3D cube procession sits above the logos.
	g.cubeTrain.Draw(g.mainCanvas)

	// 5. The composed title band always stays on top.
	if g.titleLayer != nil {
		g.titleLayer.Draw(g.mainCanvas)
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
