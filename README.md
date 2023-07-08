## Utilities
[![Go Report Card](https://goreportcard.com/badge/github.com/invertedv/utilities)](https://goreportcard.com/report/github.com/invertedv/utilities)
[![godoc](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/mod/github.com/invertedv/utilities?tab=overview)

Everybody has little functions that the find themselves writing ... and re-writing in different packages.
Enough! I got tired of that, so I created a package for it.  

You might find them useful, too.

### plotly

Included are functions that work with the excellent Go [Plotly](github.com/MetalBlueberry/go-plotly) project.
Some of these are convenience functions -- structs to hold common plot parameters such as title.
Others output plots in standard formats such as png and jpeg. These rely on the
[orca](https://github.com/plotly/orca) app.  The steps to use this app on Linux are:

1. Download the latest [release](https://github.com/plotly/orca/releases) of the AppImage.

2. Put it somewhere safe.

3. chmod +x

4. make a symbolic link called "orca" in your path.

   orca requires FUSE.  FUSE installation instructions are [here](https://github.com/AppImage/AppImageKit/wiki/FUSE)

Note that --no-sandbox is added to orca per this [thread](https://github.com/chrismaltby/gb-studio/issues/1102).


