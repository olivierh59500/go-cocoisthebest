package main

import (
	"flag"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/go-cocoisthebest/dck"
)

func main() {
	normalizedCRT := flag.Bool("normalize-crt", false, "use atlas-independent CRT sampling for the intro")
	flag.Parse()
	ebiten.SetWindowSize(coco.LogicalScreenWidth, coco.LogicalScreenHeight)
	ebiten.SetWindowTitle("COCO IS THE BEST - DMA 2025")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(coco.NewGameWithOptions(coco.GameOptions{NormalizeIntroCRT: *normalizedCRT})); err != nil {
		log.Fatal(err)
	}
}
