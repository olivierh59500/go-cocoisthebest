// Package mobile exposes the game to ebitenmobile.
package mobile

import (
	enginemobile "github.com/hajimehoshi/ebiten/v2/mobile"

	"github.com/olivierh59500/go-cocoisthebest"
)

func init() {
	enginemobile.SetGame(coco.NewGame())
}

// Dummy forces gomobile to include this package in the Android binding.
func Dummy() {}
