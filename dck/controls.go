package coco

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type control int

const (
	controlVolumeUp control = iota
	controlVolumeDown
	controlSpeedUp
	controlSpeedDown
	controlCount
)

type controlButton struct {
	bounds image.Rectangle
	label  string
}

func makeControlLayout(logicalWidth int) [controlCount]controlButton {
	const (
		size = 84
		gap  = 24
	)

	var buttons [controlCount]controlButton
	sideWidth := (logicalWidth - screenWidth) / 2
	if sideWidth < size+16 {
		return buttons
	}

	leftX := (sideWidth - size) / 2
	rightX := logicalWidth - sideWidth + leftX
	topY := (screenHeight - size*2 - gap) / 2
	bottomY := topY + size + gap
	buttons[controlVolumeUp] = controlButton{bounds: image.Rect(leftX, topY, leftX+size, topY+size), label: "V+"}
	buttons[controlVolumeDown] = controlButton{bounds: image.Rect(leftX, bottomY, leftX+size, bottomY+size), label: "V-"}
	buttons[controlSpeedUp] = controlButton{bounds: image.Rect(rightX, topY, rightX+size, topY+size), label: "S+"}
	buttons[controlSpeedDown] = controlButton{bounds: image.Rect(rightX, bottomY, rightX+size, bottomY+size), label: "S-"}
	return buttons
}

func (g *Game) updateControls() {
	buttons := makeControlLayout(g.logicalWidth)
	var pressed [controlCount]bool

	for _, touchID := range ebiten.AppendTouchIDs(g.touchIDs[:0]) {
		x, y := ebiten.TouchPosition(touchID)
		for action, button := range buttons {
			pressed[action] = pressed[action] || image.Pt(x, y).In(button.bounds)
		}
	}

	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		for action, button := range buttons {
			pressed[action] = pressed[action] || image.Pt(x, y).In(button.bounds)
		}
	}

	if g.musicStream != nil {
		volumeDelta := 0.0
		if ebiten.IsKeyPressed(ebiten.KeyUp) || pressed[controlVolumeUp] {
			volumeDelta += 0.01
		}
		if ebiten.IsKeyPressed(ebiten.KeyDown) || pressed[controlVolumeDown] {
			volumeDelta -= 0.01
		}
		if volumeDelta != 0 {
			g.musicStream.SetVolume(g.musicStream.Volume() + volumeDelta)
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) ||
		inpututil.IsKeyJustPressed(ebiten.KeyNumpadAdd) ||
		pressed[controlSpeedUp] && !g.controlPressed[controlSpeedUp] {
		g.speedMultiplier = min(g.speedMultiplier+0.1, 2)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) ||
		inpututil.IsKeyJustPressed(ebiten.KeyNumpadSubtract) ||
		pressed[controlSpeedDown] && !g.controlPressed[controlSpeedDown] {
		g.speedMultiplier = max(g.speedMultiplier-0.1, 0.5)
	}

	g.controlPressed = pressed
}

func (g *Game) drawControls(screen *ebiten.Image) {
	buttons := makeControlLayout(screen.Bounds().Dx())
	for action, button := range buttons {
		if button.bounds.Empty() {
			continue
		}

		fill := color.RGBA{R: 24, G: 18, B: 8, A: 220}
		border := color.RGBA{R: 200, G: 110, B: 16, A: 255}
		if g.controlPressed[action] {
			fill = color.RGBA{R: 150, G: 70, B: 4, A: 240}
			border = color.RGBA{R: 255, G: 190, B: 70, A: 255}
		}

		bounds := button.bounds
		vector.FillRect(screen, float32(bounds.Min.X), float32(bounds.Min.Y), float32(bounds.Dx()), float32(bounds.Dy()), fill, false)
		vector.StrokeRect(screen, float32(bounds.Min.X), float32(bounds.Min.Y), float32(bounds.Dx()), float32(bounds.Dy()), 2, border, false)
		g.drawControlLabel(screen, button.label, bounds)
	}
}

func (g *Game) drawControlLabel(screen *ebiten.Image, label string, bounds image.Rectangle) {
	if g.fontImg == nil {
		return
	}

	const scale = 0.6
	width := 0.0
	for _, char := range label {
		if letter, ok := g.letterData[char]; ok {
			width += float64(letter.width) * scale
		}
	}

	x := float64(bounds.Min.X) + (float64(bounds.Dx())-width)/2
	y := float64(bounds.Min.Y) + (float64(bounds.Dy())-fontHeight*scale)/2
	for _, char := range label {
		letter, ok := g.letterData[char]
		if !ok {
			continue
		}
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Scale(scale, scale)
		op.GeoM.Translate(x, y)
		op.ColorScale.Scale(1, 0.75, 0.35, 1)
		screen.DrawImage(letter.glyph, op)
		x += float64(letter.width) * scale
	}
}
