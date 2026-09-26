package coco

import (
	"image"
	"io"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	kit "github.com/olivierh59500/democonstructionkit"
	"github.com/olivierh59500/democonstructionkit/composite"
	"github.com/olivierh59500/democonstructionkit/effects"
	"github.com/olivierh59500/democonstructionkit/motion"
	"github.com/olivierh59500/democonstructionkit/presets"
	"github.com/olivierh59500/democonstructionkit/sound"
	"github.com/olivierh59500/democonstructionkit/sprites"
)

func TestLayout(t *testing.T) {
	tests := []struct {
		name                        string
		outsideWidth, outsideHeight int
		wantWidth, wantHeight       int
	}{
		{name: "original aspect ratio", outsideWidth: 800, outsideHeight: 600, wantWidth: 800, wantHeight: 600},
		{name: "Pixel 10a landscape", outsideWidth: 2424, outsideHeight: 1080, wantWidth: 1346, wantHeight: 600},
		{name: "narrow display", outsideWidth: 600, outsideHeight: 800, wantWidth: 800, wantHeight: 600},
		{name: "height unavailable", outsideWidth: 0, outsideHeight: 0, wantWidth: 800, wantHeight: 600},
	}

	game := &Game{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotWidth, gotHeight := game.Layout(test.outsideWidth, test.outsideHeight)
			if gotWidth != test.wantWidth || gotHeight != test.wantHeight {
				t.Fatalf("Layout(%d, %d) = (%d, %d), want (%d, %d)",
					test.outsideWidth, test.outsideHeight,
					gotWidth, gotHeight, test.wantWidth, test.wantHeight)
			}
		})
	}
}

func TestMusicStreamSeekUsesPCMByteOffsets(t *testing.T) {
	player, err := sound.Open("music.ym", musicData, sound.Options{SampleRate: sampleRate, Loop: true, PCMFormat: sound.PCM16, Gain: 0.7})
	if err != nil {
		t.Fatal(err)
	}
	defer player.Close()
	// Preserve the exact byte position, including the middle of a stereo frame.
	const tail = 97
	target := int64(sampleRate*4 + 3)
	sequential := make([]byte, target+tail)
	if _, err := io.ReadFull(player, sequential); err != nil {
		t.Fatal(err)
	}
	if got, err := player.Seek(target, io.SeekStart); err != nil || got != target {
		t.Fatalf("Seek = %d, %v", got, err)
	}
	after := make([]byte, tail)
	if _, err := io.ReadFull(player, after); err != nil {
		t.Fatal(err)
	}
	for i, v := range after {
		if v != sequential[int(target)+i] {
			t.Fatalf("seek did not reproduce PCM at byte %d", i)
		}
	}
}

func TestMusicStreamVolumeIsClamped(t *testing.T) {
	player, err := sound.Open("music.ym", musicData, sound.Options{SampleRate: sampleRate, Loop: true, PCMFormat: sound.PCM16, Gain: 0.7})
	if err != nil {
		t.Fatal(err)
	}
	defer player.Close()

	player.SetVolume(-1)
	if got := player.Volume(); got != 0 {
		t.Fatalf("volume below range = %v, want 0", got)
	}

	player.SetVolume(2)
	if got := player.Volume(); got != 1 {
		t.Fatalf("volume above range = %v, want 1", got)
	}
}

func TestCubeBackFaceCulling(t *testing.T) {
	cube, err := effects.NewSolidCube(presets.CocoCube(40))
	if err != nil {
		panic(err)
	}
	defer cube.Close()
	vertices, _ := cube.Geometry(400, 300)
	if got, want := len(vertices)/20, 1; got != want {
		t.Fatalf("axis-aligned visible faces = %d, want %d", got, want)
	}

	cube.Rotate(0.4, 0.6, 0.2)
	vertices, _ = cube.Geometry(400, 300)
	if got, want := len(vertices)/20, 3; got != want {
		t.Fatalf("rotated visible faces = %d, want %d", got, want)
	}
}

func TestControlLayoutUsesOnlySideAreas(t *testing.T) {
	buttons := makeControlLayout(1346)
	sceneLeft := (1346 - screenWidth) / 2
	sceneRight := sceneLeft + screenWidth

	for action, button := range buttons {
		if button.bounds.Empty() {
			t.Fatalf("control %d has empty bounds", action)
		}
		if action <= int(controlVolumeDown) && button.bounds.Max.X > sceneLeft {
			t.Fatalf("left control %d overlaps scene: %v", action, button.bounds)
		}
		if action >= int(controlSpeedUp) && button.bounds.Min.X < sceneRight {
			t.Fatalf("right control %d overlaps scene: %v", action, button.bounds)
		}
	}

	for action, button := range makeControlLayout(screenWidth) {
		if !button.bounds.Empty() {
			t.Fatalf("control %d should be hidden at the original aspect ratio", action)
		}
	}
}

func TestUpdateDemoUsesSpeedMultiplier(t *testing.T) {
	game := &Game{speedMultiplier: 2}
	bars := ebiten.NewImage(46, 20)
	defer bars.Deallocate()
	var err error
	game.copper, err = composite.NewCopperBars(presets.BilizirCopperBars(bars, 72, composite.CopperImages, composite.SingleWrapClock))
	if err != nil {
		t.Fatal(err)
	}
	game.titleMotion, err = motion.NewWaveClock(presets.CocoTitleMotion(screenWidth))
	if err != nil {
		t.Fatal(err)
	}
	title := ebiten.NewImage(1, 1)
	defer title.Deallocate()
	game.titleLayer, err = composite.NewSurfaceLayer(composite.SurfaceLayerConfig{
		Width: screenWidth, Height: 72,
		Sources: []kit.Effect{game.copper},
		Passes:  []composite.SurfaceImagePass{{Image: title, X: game.titleMotion.At(0)}},
		Outputs: []composite.SurfaceOutput{{}},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer game.titleLayer.Close()
	game.cubeTrain, err = effects.NewSolidCubeTrain(presets.CocoCubeTrain(screenWidth, screenHeight, 40, nbCubes))
	if err != nil {
		t.Fatal(err)
	}
	defer game.cubeTrain.Close()
	game.logoFormation, err = sprites.NewGroup(presets.CocoLogoFormation(nil, screenWidth, screenHeight))
	if err != nil {
		t.Fatal(err)
	}
	texture := ebiten.NewImage(1, 1)
	defer texture.Deallocate()
	game.rotoProgram, err = presets.NewVivaRotozoom(presets.CocoRotozoom(screenWidth, screenHeight))
	if err != nil {
		t.Fatal(err)
	}
	game.roto, err = composite.NewRotozoomBackground(composite.RotozoomBackgroundConfig{
		Image: texture, Program: game.rotoProgram, SourceQuad: image.Pt(rotoWidth, rotoHeight),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := game.updateDemo(); err != nil {
		t.Fatal(err)
	}
	copperA, copperB := game.copper.Phases()
	rotoPose := game.roto.Repetition()
	cubePosition, cubeRotation, ok := game.cubeTrain.Pose(0)
	if !ok {
		t.Fatal("first cube is missing")
	}

	checks := map[string]struct {
		got, want float64
	}{
		"demo time":         {got: game.demoTime, want: 2},
		"copper forward":    {got: copperA, want: 6},
		"copper reverse":    {got: copperB, want: 1014},
		"sprite phase":      {got: game.logoFormation.Phase(), want: 0.04},
		"rotozoom center x": {got: rotoPose.CenterX, want: 400 + 200*math.Cos(.016*4-math.Cos(.016-.1))},
		"rotozoom center y": {got: rotoPose.CenterY, want: 300 + (600.0/2.7)*-math.Sin(.016*2.3-math.Cos(.016-.1))},
		"rotozoom zoom":     {got: rotoPose.Zoom, want: .5 + math.Abs(math.Sin(.006)*2.5)},
		"rotozoom rotation": {got: rotoPose.Rotation, want: 360.0 / 4.0 * math.Cos(.01*4-math.Cos(.01-.01)) * .3 * math.Pi / 180},
		"title phase":       {got: game.titleMotion.Phase(), want: .525},
		"title position":    {got: game.titleMotion.At(0), want: 64 + 800*math.Cos(.525)},
		"cube x":            {got: cubePosition.X, want: 380 + 380*math.Sin(.15+.08)},
		"cube y":            {got: cubePosition.Y, want: 300 + 84*math.Cos((.15+.08)*2.5)},
		"cube rotation":     {got: cubeRotation.X, want: 0.04},
	}
	for name, check := range checks {
		if math.Abs(check.got-check.want) > 1e-12 {
			t.Errorf("%s = %v, want %v", name, check.got, check.want)
		}
	}
}

func BenchmarkMusicStreamRead(b *testing.B) {
	player, err := sound.Open("music.ym", musicData, sound.Options{SampleRate: sampleRate, Loop: true, PCMFormat: sound.PCM16, Gain: 0.7})
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		_ = player.Close()
	})
	buffer := make([]byte, 4096)

	b.ReportAllocs()
	b.SetBytes(int64(len(buffer)))
	b.ResetTimer()
	for range b.N {
		if _, err := player.Read(buffer); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCubeGeometry(b *testing.B) {
	cube, err := effects.NewSolidCube(presets.CocoCube(40))
	if err != nil {
		panic(err)
	}
	defer cube.Close()

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		cube.Rotate(0.02, 0.03, 0.01)
		cube.Geometry(400, 300)
	}
}
