# YT Playlist Downloader

A fast, single-binary YouTube playlist downloader with a clean browser-based GUI. Built in Go — no Python environment hassle, no pip, no dependencies to manage. Just run the binary and go.

![screenshot](https://raw.githubusercontent.com/VedadKrajina/yt-playlist-downloader/main/screenshot.png)

## Features

- Download entire playlists **or** single videos
- Choose between **MP4** (video) or **MP3** (audio only, 192kbps)
- Live progress bar and log streamed in real time
- Native folder picker (uses `zenity` / `kdialog` if available)
- Dark UI served locally — opens automatically in your browser
- Single ~9 MB binary, zero runtime deps beyond `yt-dlp` and `ffmpeg`

## Requirements

| Tool | Install (Arch) | Install (Debian/Ubuntu) |
|------|----------------|------------------------|
| `yt-dlp` | `sudo pacman -S yt-dlp` or `pip install yt-dlp` | `pip install yt-dlp` |
| `ffmpeg` | `sudo pacman -S ffmpeg` | `sudo apt install ffmpeg` |

> Go is only needed if you want to **build from source**. The pre-built binary needs nothing extra.

## Download

Grab the latest binary from the [Releases](https://github.com/VedadKrajina/yt-playlist-downloader/releases) page, make it executable, and run it:

```bash
chmod +x ytdl-downloader
./ytdl-downloader
```

Your browser will open automatically with the GUI.

## Build from Source

```bash
git clone https://github.com/VedadKrajina/yt-playlist-downloader.git
cd yt-playlist-downloader/ytdl
go build -o ../ytdl-downloader .
```

Requires [Go 1.21+](https://go.dev/dl/).

## Usage

1. Run `./ytdl-downloader`
2. Paste a YouTube playlist or video URL
3. Select **MP4** or **MP3**
4. Choose a save folder (or type a path)
5. Hit **Download**

Downloaded files are named `01 - Title.mp4` / `01 - Title.mp3` etc., preserving playlist order.

## How It Works

The binary starts a local HTTP server on a random port and opens your default browser. All downloading is handled by `yt-dlp` running as a subprocess; progress is streamed back to the browser via Server-Sent Events (SSE). No data ever leaves your machine.

## License

MIT
