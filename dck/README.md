# DCK version

This directory contains the construction-kit version of go-cocoisthebest. The original Go sources are preserved at their original paths (revision `0a9b678b13af04bf248b4a8f23bb79496dd36c53`), with small asset accessors so both versions use the same embedded resources.

Run the original with `go run ./cmd/cocoisthebest` and this version with `go run ./dck/cmd/cocoisthebest` from the repository root.

The choreography and assets remain in this repository. Reusable rendering and
effects come from the published `github.com/olivierh59500/democonstructionkit`
module pinned in `go.mod`. Music is opened with `sound.Open`; DCK selects the decoder from the asset and
provides the configured stereo PCM format. The demo keeps its playback level and loop settings.

The intro completion and immediate main-music cue use DCK's
`timeline.IntroHandoff` without a fade, preserving the same entry tick.
The opt-in `coco_intro_sourcecheck` captures the intro bitmap before CRT in
both packages; those surfaces are byte-identical at ticks 0, 1, 60, 240 and
241. The final intro output still differs in the CRT pass at sampled ticks,
while complete main-scene frames at 600, 1,200, 2,400 and 4,800 match the
preserved Go original. The remaining CRT output difference is tracked in the
DCK fidelity sweep rather than hidden by a tolerance. Set
`COCO_INTRO_SOURCE_CAPTURES` to separate output directories when running the
tagged root and `dck` tests.
The `coco_crt_same_sourcecheck` GPU test feeds one texture to the preserved
shader and DCK's shared `CRTOverlay`; their pixels match exactly at ticks 0
and 240. Giving the DCK intro strip its own unmanaged GPU surface reduces
complete-frame differences at those ticks from 13,827 to 6,269 and from
22,379 to 11,396 respectively, without changing the decoded source pixels.
The remaining difference is still visible in the fidelity report. The updated
DCK APK was installed on Pixel 10a; its main-stage 744-interval sample had
p95 16.731 ms, maximum 16.898 ms and none above 20 ms.

The title's 36 copper bars use `composite.CopperBars` with cached DrawImage
strips and fractional phase clocks. The speed control changes the shared
component's clock without resetting either phase. A 60-second comparison
matched all 3,600 decoded frames, including the intro and animated title.

The main background uses `composite.RotozoomBackground` with the
`presets.CocoRotozoom` harmonic program. The source-sized quad, repeated tile,
half-bright tint, and live speed control stay configurable in DCK. The original
production still keeps its own renderer; this version only supplies its image
and screen dimensions.

The top band uses `composite.CopperTitleBand` in retained-surface mode. It owns
the bounded copper/title surface, both copper phases and the title wave clock.
The published preset keeps the original starting phase, fractional wrap and
live speed control; another image, palette or path can be configured without
rewriting the scene.

The twelve orange cubes use `effects.SolidCubeTrain`, which owns their
independent phases, sinusoidal path, per-index rotations and one bounded draw
batch. `presets.CocoCubeTrain` supplies Coco's original values; count, cube
materials, curve rates, spacing and a custom path can be changed independently.
