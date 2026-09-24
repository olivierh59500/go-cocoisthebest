package coco

import (
	"io"
	"math"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/democonstructionkit/composite"
	"github.com/olivierh59500/democonstructionkit/effects"
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
	for i := range game.cubes {
		var err error
		game.cubes[i], err = effects.NewSolidCube(presets.CocoCube(40))
		if err != nil {
			t.Fatal(err)
		}
		defer game.cubes[i].Close()
	}
	game.logoFormation, err = sprites.NewGroup(presets.CocoLogoFormation(nil, screenWidth, screenHeight))
	if err != nil {
		t.Fatal(err)
	}
	if err := game.updateDemo(); err != nil {
		t.Fatal(err)
	}
	copperA, copperB := game.copper.Phases()

	checks := map[string]struct {
		got, want float64
	}{
		"demo time":      {got: game.demoTime, want: 2},
		"copper forward": {got: copperA, want: 6},
		"copper reverse": {got: copperB, want: 1014},
		"sprite phase":   {got: game.logoFormation.Phase(), want: 0.04},
		"rotozoom x":     {got: game.posXi, want: 0.016},
		"rotozoom z":     {got: game.posZi, want: 0.006},
		"rotozoom angle": {got: game.posRi, want: 0.01},
		"title phase":    {got: game.logoX, want: 0.025},
		"cube position":  {got: game.spritePos[0], want: 0.08},
		"cube rotation":  {got: game.cubes[0].Rotation.X, want: 0.04},
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
