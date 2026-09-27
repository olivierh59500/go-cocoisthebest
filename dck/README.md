# DCK version

This directory contains the construction-kit version of go-cocoisthebest. The original Go sources are preserved at their original paths (revision `0a9b678b13af04bf248b4a8f23bb79496dd36c53`), with small asset accessors so both versions use the same embedded resources.

Run the original with `go run ./cmd/cocoisthebest` and this version with `go run ./dck/cmd/cocoisthebest` from the repository root.

The choreography and assets remain in this repository. Reusable rendering and
effects come from the published `github.com/olivierh59500/democonstructionkit`
module pinned in `go.mod`. Music is opened with `sound.Open`; DCK selects the decoder from the asset and
provides the configured stereo PCM format. The demo keeps its playback level and loop settings.

The intro completion and immediate main-music cue use DCK's
`timeline.IntroHandoff` without a fade, preserving the same entry tick.
The intro CRT uses the editable `presets.CocoCRTOverlay()` recipe. Its
`SourceOrigin` of four pixels reproduces the historical texture coordinate;
the shared effect owns the bounded source copy, so this port no longer keeps
an intro strip or draws that strip locally. The opt-in `coco_intro_sourcecheck`
confirms that the raw scrolling bitmap remains identical in both packages at
ticks 0, 1, 60, 240 and 241. Complete 800×600 original-versus-DCK captures
are now pixel-identical at those ticks and at 600, 1,200, 2,400 and 4,800.
Set `COCO_INTRO_SOURCE_CAPTURES` to separate output directories when running
the tagged root and `dck` tests.
`NewGameWithOptions(GameOptions{NormalizeIntroCRT: true})` selects DCK's
atlas-independent CRT policy for a new visual variation. The ordinary
`NewGame()` keeps Coco's exact historical sampling. The normalized variant
uses no source offset; its five sampled outputs are unchanged by this update.
The opt-in `coco_intro_sourcecheck` accepts `COCO_NORMALIZE_CRT=1` for
inspecting that variant.
On desktop, run `go run ./dck/cmd/cocoisthebest -normalize-crt` to try it.

The updated DCK APK was installed on a Pixel 10a. Across 744 distinct frame
intervals spanning the intro and main handoff, p95 was 16.741 ms, the maximum
was 16.941 ms, and none exceeded 20 ms. One process snapshot showed 215,793
KiB PSS and 116,028 KiB graphics memory; thermal status was 0. This short
sample does not measure peak memory or long-run battery use.

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
