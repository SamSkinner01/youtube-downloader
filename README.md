# YouTube Downloader

Download YouTube videos as MP3, MP4, or WAV from the command line or a native macOS GUI.

## Requirements

- [ffmpeg](https://ffmpeg.org/) — `brew install ffmpeg`

## Installation

1. Download the latest `YouTube-Downloader-x.x.x.dmg` from [Releases](https://github.com/SamSkinner01/youtube-downloader/releases)
2. Open the DMG and drag **YouTube Downloader** into your Applications folder
3. Run the following command in Terminal to allow the app to open:
   ```sh
   xattr -cr "/Applications/YouTube Downloader.app"
   ```
4. Open the app from your Applications folder

> **Why is this needed?** macOS blocks apps downloaded from the internet that aren't signed with a paid Apple Developer certificate. The `xattr -cr` command removes that restriction for this app.

## GUI Usage

Open **YouTube Downloader** from your Applications folder.

| Field | Description |
|-------|-------------|
| URL | Paste any YouTube link |
| Filename | Custom output filename (leave blank to use the video title) |
| Save to | Output folder (defaults to ~/Downloads) |
| Format | MP3, MP4, or WAV |
| Quality | Best, medium, or worst available stream |

## CLI Usage

```sh
# Download as MP3 (default)
youtube-downloader "https://youtube.com/watch?v=..."

# Download as MP4
youtube-downloader -f mp4 "https://youtube.com/watch?v=..."

# Download as WAV
youtube-downloader -f wav "https://youtube.com/watch?v=..."

# Custom output filename
youtube-downloader -o my-song "https://youtube.com/watch?v=..."

# List available streams
youtube-downloader -formats "https://youtube.com/watch?v=..."
```

### CLI Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-f` | `mp3` | Output format: `mp3`, `mp4`, `wav` |
| `-q` | `best` | Quality: `best`, `medium`, `worst` |
| `-o` | video title | Output file path |
| `-formats` | — | List all available streams and exit |

## Building from Source

```sh
# CLI
go build ./cmd/cli/

# GUI
go build ./cmd/gui/

# macOS installer (DMG)
cd scripts && ./build-installer.sh
```

## A note on quality

YouTube's audio-only streams max out at ~128–160 kbps regardless of format. Sites claiming to offer 320 kbps MP3 are re-encoding the same source to a higher bitrate number — the actual audio quality is identical.

| Format | Quality |
|--------|---------|
| MP3 | Lossy. Re-encoded from YouTube's already-compressed stream |
| MP4 | Video copied losslessly. Audio re-encoded to AAC |
| WAV | No further loss — raw PCM decode of the source stream |
