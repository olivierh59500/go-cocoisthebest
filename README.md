# 🎨 COCO IS THE BEST 🎨

**A loving tribute to the golden age of Atari ST demoscene, rebuilt in pure Go!**

Welcome to the most rad demoscene experience you'll have in a terminal-spawned window! This demo combines the best visual effects from classic Atari ST demos, all rendered with the power of modern Go and a whole lot of sine waves.

![DMA 2025](https://img.shields.io/badge/DMA-2025-orange) ![Made with Go](https://img.shields.io/badge/Made%20with-Go-00ADD8) ![Demoscene](https://img.shields.io/badge/Demoscene-Forever-ff69b4)

## 🚀 Quick Start

```bash
# Clone this masterpiece
git clone https://github.com/olivierh59500/go-cocoisthebest.git
cd go-cocoisthebest

# Run the demo
go run ./cmd/cocoisthebest

# Or build it
go build -o cocoisthebest ./cmd/cocoisthebest
./cocoisthebest
```

### Android (Pixel en USB)

Avec le SDK Android, Java 17 et un unique appareil ADB autorisé :

```bash
./scripts/run-android.sh
```

Le script génère l’AAR arm64, construit l’APK de débogage, l’installe puis
lance `com.olivierh.cocoisthebest/.MainActivity`.

## 🎭 The Effects

This demo packs **six classic demoscene effects** that'll make your eyes dance and your heart sing:

### 🌊 Copper Bars

Remember those mesmerizing horizontal color bars sweeping across the screen? We've got 'em! These beauties use dual sine wave tables to create that hypnotic oscillating motion. Watch as they weave behind the title logo, just like the good old days.

**Technical sauce**: 1024-entry sine lookup table, dual-phase animation, 36 bars × 2 pixels = 72 pixels of pure copper goodness

### 🎲 3D Rotating Cubes

Twelve orange-hued cubes spinning through 3D space with that classic filled-polygon look. Each cube rotates on all three axes at slightly different speeds, creating a mesmerizing asynchronous ballet of geometry.

**Technical sauce**: Hand-rolled 3D rotation matrices, perspective projection with z-buffering, painter's algorithm for face sorting, triangle rasterization

### 🌀 Rotozoom

A tiled "COCO" texture that rotates, zooms, and oscillates across the screen. This classic effect creates mind-bending patterns as it scales from tiny to massive while spinning like a vinyl record on a turntable.

**Technical sauce**: Combined rotation and zoom matrices, sinusoidal center-point oscillation, texture wrapping, 8× upscaled canvas for smooth animation

### 📜 Mega-Twist Scroller

This ain't your grandma's horizontal scroller! The text waves and distorts vertically using eight different curve types (slow sin, medium sin, fast sin, slow distorted, medium distorted, fast distorted, and even a "splitted" effect). It's like the text is made of jelly riding a rollercoaster.

**Technical sauce**: 8 precalculated wave curve types, delta-encoded position tables, per-scanline rendering with bounce effect, seamless text wrapping

### 👾 DMA Logo Sprites

Sixteen semi-transparent DMA logos arranged in a 4×4 grid, all synchronized to move together in beautiful Lissajous-curve patterns. They float like ethereal jellyfish across your screen.

**Technical sauce**: Synchronized sprite motion using dual sine/cosine calculations, 4×4 grid layout, alpha blending

### 📺 CRT Shader

The entire intro is rendered through an authentic CRT shader featuring:
- **Barrel distortion** - That classic curved-screen look
- **Scanlines** - Those horizontal lines that made everything look cooler
- **RGB separation** - Chromatic aberration for that slightly-out-of-tune color TV vibe
- **Vignette** - Darkened edges just like real CRTs

**Technical sauce**: Kagi shader language, multi-pass image effects, real-time fragment processing

## 🎵 The Music

Authentic **Atari ST YM music** plays throughout the demo! The YM format was the sound of the demoscene, created by the legendary YM2149 sound chip. We're using a real YM player that generates samples on the fly.

**Track**: "Mindbomb" (embedded in the binary)

## 🎮 Controls

- **↑/↓** - Adjust music volume (because sometimes you want it LOUDER)
- **+/-** - Speed multiplier (0.5× to 2.0×) - Make the demo dance faster or go full slow-mo
- **Android** - Use the `V+`/`V-` and `S+`/`S-` buttons in the side areas; multitouch is supported
- **Just watch** - Sometimes the best interaction is appreciation

## 🏗️ Technical Details

- **Language**: Go 1.25+
- **Engine**: Ebiten v2 (a dead-simple 2D game library)
- **Architecture**: Single-file monolith (~1,350 lines of pure demo code)
- **Audio**: Real YM2149 emulation via ym-player
- **Graphics**: All effects rendered in software, no GPU shaders (except the CRT effect)
- **Philosophy**: If it can be done with a sine table, it will be done with a sine table

## 🌟 The Demoscene Spirit

This demo is a love letter to the demoscene - that beautiful intersection of programming, art, and music where people pushed computers to do things they were never meant to do, just because they could.

Back in the day, Atari ST demos like:
- **Mindbomb** by TLB
- **Union Demo** by The Union
- **Cuddly Demos** by TCB
- ...

...showed us that with clever coding, limited hardware could produce unlimited creativity. This project brings that spirit into the modern age with Go.

## 🙏 Greetings

Greetings to all demosceners, past and present! To the cracktro makers, the demoparty organizers, the chiptune composers, and everyone who ever thought "I wonder if I can make the screen do *that*..."

**Special shoutouts**: The Union and all the groups who made the Atari ST demoscene legendary.

## 📝 License

GPLv3 License - Because demos should be free, just like they always were.

## 🤝 Contributing

Found a bug? Want to add another effect? Pull requests welcome! Let's keep the demoscene alive, one commit at a time.

---

**Made with ❤️ and an unreasonable number of sine waves**

*"If you think this is all, you're so wrong..."*

## Optional DCK version

The original implementation remains at its original paths. Run it with `go run ./cmd/cocoisthebest`.

The construction-kit version is in [dck/](dck/README.md). Run `go run ./dck/cmd/cocoisthebest` from this directory. Both versions share the original assets.
