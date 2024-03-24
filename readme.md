# Work in progress

Terminal app to create optimized videos for use on the web.

![TUI example gif](assets/example.gif)

## Use

Choose an input file and set your options.

A folder named `optimized` will be created in the working directory containing the optimized videos.

To use in HTML:

```html
<video>
	<source type="video/mp4" src="output.mp4" />
	<source type="video/webm" src="output.webm" />
</video>
```

## Todo

- goreleaser
- update option
