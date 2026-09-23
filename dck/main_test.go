package coco

import (
	"io"
	"math"
	"testing"

	"github.com/olivierh59500/democonstructionkit/sound"
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

func TestScrollMeshTopology(t *testing.T) {
	game := &Game{}
	game.initScrollMesh()

	const lineCount = screenHeight - 72
	if got, want := len(game.scrollVertices), lineCount*4; got != want {
		t.Fatalf("vertex count = %d, want %d", got, want)
	}
	if got, want := len(game.scrollIndices), lineCount*6; got != want {
		t.Fatalf("index count = %d, want %d", got, want)
	}
	for _, index := range game.scrollIndices {
		if int(index) >= len(game.scrollVertices) {
			t.Fatalf("index %d exceeds vertex count %d", index, len(game.scrollVertices))
		}
	}
}

func TestCubeBackFaceCulling(t *testing.T) {
	cube := NewCube3D(40)
	cube.updateGeometry(400, 300)
	if got, want := cube.visibleFaces, 1; got != want {
		t.Fatalf("axis-aligned visible faces = %d, want %d", got, want)
	}

	cube.Rotate(0.4, 0.6, 0.2)
	cube.updateGeometry(400, 300)
	if got, want := cube.visibleFaces, 3; got != want {
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
	game.updateDemo()

	checks := map[string]struct {
		got, want float64
	}{
		"demo time":      {got: game.demoTime, want: 2},
		"copper forward": {got: game.cnt, want: 6},
		"copper reverse": {got: game.cnt2, want: 1014},
		"sprite phase":   {got: game.ctrSprite, want: 0.04},
		"rotozoom x":     {got: game.posXi, want: 0.016},
		"rotozoom z":     {got: game.posZi, want: 0.006},
		"rotozoom angle": {got: game.posRi, want: 0.01},
		"title phase":    {got: game.logoX, want: 0.025},
		"cube position":  {got: game.spritePos[0], want: 0.08},
		"cube rotation":  {got: game.cubes[0].angleX, want: 0.04},
	}
	for name, check := range checks {
		if math.Abs(check.got-check.want) > 1e-12 {
			t.Errorf("%s = %v, want %v", name, check.got, check.want)
		}
	}
}

func TestScrollLetterWrapsAcrossMultipleMessageLoops(t *testing.T) {
	game := &Game{
		position:        []int{10, 20, 30},
		scrollTextRunes: []rune("ABC"),
	}

	if got, want := game.getPosition(4), 40; got != want {
		t.Fatalf("position after wrap = %d, want %d", got, want)
	}

	game.advanceScrollLetter(95)
	if got, want := game.letterNum, 9; got != want {
		t.Fatalf("letter after three loops = %d, want %d", got, want)
	}
	if got, want := game.getLetter(game.letterNum), 'A'; got != want {
		t.Fatalf("wrapped letter = %q, want %q", got, want)
	}

	// Some curve sections move backwards; the active letter must follow them.
	game.advanceScrollLetter(5)
	if got, want := game.letterNum, 0; got != want {
		t.Fatalf("letter after backwards movement = %d, want %d", got, want)
	}
}

func TestScrollerFollowsRealWaveAcrossThreeFullMessages(t *testing.T) {
	game := &Game{
		letterData:      make(map[rune]Letter),
		scrollTextRunes: []rune(demoScrollText),
	}
	game.initFontData()
	game.createCurves()
	game.precalcPosition()
	game.precalcMainWave()

	targetLetter := len(game.scrollTextRunes) * 3
	for frontWavePos := 0; frontWavePos < 100_000_000 && game.letterNum < targetLetter; frontWavePos += 600 {
		decalX := game.scrollOffset(frontWavePos)
		game.advanceScrollLetter(decalX)
		if start, end := game.getPosition(game.letterNum), game.getPosition(game.letterNum+1); decalX < start || decalX >= end {
			t.Fatalf("offset %d is outside active letter %d interval [%d, %d)", decalX, game.letterNum, start, end)
		}
	}
	if game.letterNum < targetLetter {
		t.Fatalf("scroller reached only letter %d, want at least %d", game.letterNum, targetLetter)
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
	cube := NewCube3D(40)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		cube.Rotate(0.02, 0.03, 0.01)
		cube.updateGeometry(400, 300)
	}
}
