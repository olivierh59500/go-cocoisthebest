# DCK version

This directory contains the construction-kit version of go-cocoisthebest. The original Go sources are preserved at their original paths (revision `0a9b678b13af04bf248b4a8f23bb79496dd36c53`), with small asset accessors so both versions use the same embedded resources.

Run the original with `go run ./cmd/cocoisthebest` and this version with `go run ./dck/cmd/cocoisthebest` from the repository root.

The choreography and assets remain in this repository. Reusable rendering and
effects come from the published `github.com/olivierh59500/democonstructionkit`
module pinned in `go.mod`. Music is opened with `sound.Open`; DCK selects the decoder from the asset and
provides the configured stereo PCM format. The demo keeps its playback level and loop settings.

The intro completion and immediate main-music cue use DCK's
`timeline.IntroHandoff` without a fade, preserving the same entry tick.

The title's 36 copper bars use `composite.CopperBars` with cached DrawImage
strips and fractional phase clocks. The speed control changes the shared
component's clock without resetting either phase. A 60-second comparison
matched all 3,600 decoded frames, including the intro and animated title.
