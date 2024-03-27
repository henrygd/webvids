# webvids

Terminal app to create optimized videos for the web.

![TUI example gif](assets/example.gif)

## Installation

You must have `ffmpeg` installed on your system.

If you get an error downloading the archive URL, find the right URL for your architecture on the [releases page](https://github.com/henrygd/webvids/releases).

### One liner

```bash
sudo sh -c 'curl -L "https://github.com/henrygd/webvids/releases/latest/download/webvids_$(uname -s)_$(uname -m | sed 's/x86_64/amd64/' | sed 's/i386/386/' | sed 's/aarch64/arm64/').tar.gz" | tar -xz -O webvids | tee /usr/bin/webvids >/dev/null && chmod +x /usr/bin/webvids'
```

### Manual

```bash
# Download the latest release archive
curl -L -o webvids.tar.gz "https://github.com/henrygd/webvids/releases/latest/download/webvids_$(uname -s)_$(uname -m | sed 's/x86_64/amd64/' | sed 's/i386/386/' | sed 's/aarch64/arm64/').tar.gz"

# Extract the binary from the archive
tar -xzf webvids.tar.gz webvids

# Make the binary executable
chmod +x webvids

# Move the binary to /usr/bin
sudo mv webvids /usr/bin/
```

## Usage

Run the `webvids` command. You can specify an input file as an argument, or use the built in file picker.

A folder named `optimized` will be created in the working directory containing the optimized videos.

Use both videos in HTML with `source` tags:

```html
<video>
	<source type="video/mp4" src="output.mp4" />
	<source type="video/webm" src="output.webm" />
</video>
```

## Updating

```bash
webvids --update
```

## Todo

- check that ffmpeg / required codecs are installed
- allow selection of preset for faster / slower encoding
