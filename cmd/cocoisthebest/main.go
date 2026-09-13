package main

import (
	"log"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/go-cocoisthebest"
)

func main() {
	ebiten.SetWindowSize(coco.LogicalScreenWidth, coco.LogicalScreenHeight)
	ebiten.SetWindowTitle("COCO IS THE BEST - DMA 2025")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(coco.NewGame()); err != nil {
		log.Fatal(err)
	}
}
