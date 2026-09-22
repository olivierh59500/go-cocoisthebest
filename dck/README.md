# DCK version

This directory contains the construction-kit version of go-cocoisthebest. The original Go sources are preserved at their original paths (revision `0a9b678b13af04bf248b4a8f23bb79496dd36c53`), with small asset accessors so both versions use the same embedded resources.

Run the original with `go run ./cmd/cocoisthebest` and this version with `go run ./dck/cmd/cocoisthebest` from the repository root.

The choreography and assets remain in this repository. Reusable rendering and
effects come from the published `github.com/olivierh59500/democonstructionkit`
module pinned in `go.mod`. Go downloads the dependencies automatically, including
`github.com/olivierh59500/ym-player v1.0.0` for YM playback. Second Reality retains its original ST3 music synchronization.
