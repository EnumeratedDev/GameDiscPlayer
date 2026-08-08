# GameDiscPlayer Setup Guide
## Prerequisites
To use GameDiscPlayer you must first have a DRM-free copy of a PC game or a ROM
- You can find DRM-free PC games on [GOG](https://www.gog.com) and sometimes on [Steam](https://store.steampowered.com/) (Refer to [this list](https://www.pcgamingwiki.com/wiki/The_big_list_of_DRM-free_games_on_Steam) for DRM-free Steam games)
- You should use your own legally dumped ROMs. GameDiscPlayer does not endorse using pirated ROMs in any way

## Building
1) Install `make`, `go` and `gtk4` (Only `gtk4` is required at runtime)
2) Clone this repository
3) Build the project using `make`. The final binary should be placed in `build/GameDiscPlayer`

## Basic game setup
1) Create a new directory. You can name this anything.
2) Copy the launcher binary inside the newly created directory
3) Create a `metadata.yml` text file that contains the following:
```yml
name: "MY_GAME_BACKUP"
description: |
  Line 1
  Line 2
  Line 3
  Line...
version: "1.0.0"
developer: "developer"
publisher: "publisher"
run_from_disc: true/false
type: linux/windows/gb/gbc/gba
run: "linux_game_bin/windows_game.exe/gameboy_rom.gb"
```
4) Change the values in the metadata to suit your game backup (NOTE: Only enable `run_from_disc` if your game is a ROM or if you know that the game does not try to modify any files in its installation directory)
5) Inside the newly created directory make another one named `files` in all lowercase
6) Copy your game files inside the `files` directory
7) Change the `run` field in the metadata file to point to the game executable/ROM relative to the `files` directory (e.g for `files/my_game.exe` use `run: "my_game.exe"`)
8) Run the launcher binary and enjoy!

## Additional `metadata.yml` fields
- Some Windows games may require additional DLLs to work correctly. You can use winetricks verbs to install such DLLs like this:
```yml
winetricks_verbs:
  - d3dcompiler_47
```

## ISO file creation and burning to disc
The easiest way to create ISOs or burn a game to a disc is to use a graphical program like `K3b` for KDE or `Brasero` for Gnome
