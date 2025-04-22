# yt-playlist-downloader

This project is a Python-based tool for downloading YouTube playlists as MP3 files. It allows users to easily retrieve and download all audio tracks from a specified playlist.

## Features

- Download entire YouTube playlists as MP3 files
- Automatically saves files to `/home/vk15/Music/`
- Handles errors gracefully and skips unavailable videos
- Bypasses geographic restrictions

## Installation

To get started, clone the repository and install the required dependencies:

```bash
git clone https://github.com/yourusername/yt-playlist-downloader.git
cd yt-playlist-downloader
pip install -r requirements.txt
```

Ensure that `ffmpeg` is installed on your system. You can install it using:

```bash
sudo apt install ffmpeg
```

## Usage

To download a playlist, run the script and provide the playlist URL when prompted:

```bash
python downloader.py
```

The downloaded MP3 files will be saved in `/home/vk15/Music/<playlist_name>/`.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request for any improvements or bug fixes.

## License

This project is licensed under the MIT License. See the LICENSE file for more details.