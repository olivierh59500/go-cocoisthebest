// Command video exports the complete game canvas and its own audio.
package main

import (
	"flag"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/democonstructionkit/video"
	demo "github.com/olivierh59500/go-cocoisthebest/dck"
)

func main() {
	config := video.Config{Output: "go-cocoisthebest.mp4", Title: "Coco Is The Best", Width: 800, Height: 600, FPS: 60, TPS: 60, SampleRate: 44100, Duration: 3 * time.Minute}
	config.Flags(flag.CommandLine)
	flag.Parse()
	if err := video.Run(config, func() (ebiten.Game, error) {
		return demo.NewGame(), nil
	}); err != nil {
		log.Fatal(err)
	}
}
