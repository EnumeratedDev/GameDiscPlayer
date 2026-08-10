# GameDiscPlayer
## Description
Game Disc Player is a Linux game backup launcher designed for CD, DVD and Blu-ray but works on all kinds of removable storage. It's designed to be plug-and-play so all the user needs to do is run the launcher and press Play

## Compatibility
At this moment GameDiscPlayer has support for the following systems:
- Native Linux games
- Windows games (through [umu-launcher](https://github.com/Open-Wine-Components/umu-launcher) and [Proton-GE](github.com/GloriousEggroll/proton-ge-custom))
- Gameboy ROMs (through [mGBA](https://github.com/mgba-emu/mgba))
- Gameboy Color ROMs (through [mGBA](https://github.com/mgba-emu/mgba))
- Gameboy Advance ROMs (through [mGBA](https://github.com/mgba-emu/mgba))
- PS1 Disc Backups (through [DuckStation](https://github.com/stenzek/duckstation))
- PS2 Disc Backups (through [PCSX2](https://github.com/PCSX2/pcsx2))

## How to use
Read [GUIDE.md](GUIDE.md) for a detailed setup guide

## Licensing
The GameDiscPlayer source-code by itself is is licensed under the **MIT license**, **however**
- GameDiscPlayer uses the [GOTK4](github.com/diamondburned/gotk4) GTK bindings for Go which is licensed under the **MPL-2.0**
- The underlying [GTK library](https://www.gtk.org/) is licensed under the **LGPL-2.1**
