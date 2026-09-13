package coco

import (
	"io"
	"testing"
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

func TestYMPlayerSeekUsesPCMByteOffsets(t *testing.T) {
	player, err := NewYMPlayer(musicData, sampleRate, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := player.Close(); err != nil {
			t.Errorf("Close() failed: %v", err)
		}
	})

	oneSecond := int64(sampleRate * 4)
	got, err := player.Seek(oneSecond, io.SeekStart)
	if err != nil {
		t.Fatal(err)
	}
	if got != oneSecond {
		t.Fatalf("Seek(one second) = %d, want %d", got, oneSecond)
	}

	got, err = player.Seek(0, io.SeekEnd)
	if err != nil {
		t.Fatal(err)
	}
	if got != player.totalBytes {
		t.Fatalf("Seek(end) = %d, want %d", got, player.totalBytes)
	}
}

func TestYMPlayerVolumeIsClamped(t *testing.T) {
	player := &YMPlayer{}

	player.SetVolume(-1)
	if got := player.GetVolume(); got != 0 {
		t.Fatalf("volume below range = %v, want 0", got)
	}

	player.SetVolume(2)
	if got := player.GetVolume(); got != 1 {
		t.Fatalf("volume above range = %v, want 1", got)
	}
}
