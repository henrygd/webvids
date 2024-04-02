# webvids

Terminal app to create optimized videos for the web.

![TUI example gif](assets/example.gif)

## Installation

Please install / update `ffmpeg` first as the program requires it.

Run the command below or download the correct binary for your system on the [releases page](https://github.com/henrygd/webvids/releases).

### One liner

```bash
curl -sL "https://github.com/henrygd/webvids/releases/latest/download/webvids_$(uname -s)_$(uname -m | sed 's/x86_64/amd64/' | sed 's/i386/386/' | sed 's/aarch64/arm64/').tar.gz" | tar -xz -O webvids | sudo tee /usr/local/bin/webvids >/dev/null && sudo chmod +x /usr/local/bin/webvids && ls /usr/local/bin/webvids
```

### Manual

```bash
# Download the latest release archive
curl -L -o webvids.tar.gz "https://github.com/henrygd/webvids/releases/latest/download/webvids_$(uname -s)_$(uname -m | sed 's/x86_64/amd64/' | sed 's/i386/386/' | sed 's/aarch64/arm64/').tar.gz"

# Extract the binary from the archive
tar -xzf webvids.tar.gz webvids

# Make the binary executable
chmod +x webvids

# Move the binary to /usr/local/bin
sudo mv webvids /usr/local/bin/
```

## Usage

Run the `webvids` command. You may specify an input file or use the built-in file picker.

```bash
webvids input.mp4
```

The video(s) will be written to a folder named `optimized` in the current directory.

Use both videos in HTML with `source` tags:

```html
<video controls>
	<source type="video/mp4" src="output.mp4" />
	<source type="video/webm" src="output.webm" />
</video>
```

## Command line options

webvids can run without interaction by passing in a file and form options:

```bash
webvids --crf 26 --preview=false --strip-audio input.mp4
```

The following options are available:

| Flag              | Description                                     |
| ----------------- | ----------------------------------------------- |
| `--crf`           | Constant rate factor                            |
| `-h`, `--help`    | Show help                                       |
| `--preview`       | Converts only the first 3 seconds               |
| `--skip-av1`      | Skip AV1 conversion                             |
| `--skip-x265`     | Skip x265 conversion                            |
| `-s`, `--speed`   | Priority of conversion speed over quality (0-5) |
| `--strip-audio`   | Remove audio track from output                  |
| `-u`, `--update`  | Update to the latest version                    |
| `-v`, `--version` | Print version and exit                          |

## Uninstall

```bash
sudo rm /usr/local/bin/webvids
```

## Todo

- check that libsvtav1 codec is available
- allow multiple files to be passed in
- add flag for optimized settings for animated videos
