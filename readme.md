# webvids

Terminal app to create optimized videos for the web.

![TUI example gif](assets/example.gif)

## Usage

Choose an input file and set your options.

A folder named `optimized` will be created in the working directory containing the optimized videos.

Use both videos in HTML with `source` tags:

```html
<video>
	<source type="video/mp4" src="output.mp4" />
	<source type="video/webm" src="output.webm" />
</video>
```

## Todo

- update option
- pass file directly as argument
