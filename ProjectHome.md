This is a command line tool written in Go which:

  * Resizes images using [ImageMagick](ImageMagick.md)
  * Organizes the image files in folders named after their exif time range
  * Optionally uploads the images via FTP

Refer to the  [Summary page](WikiSummary.md) for detail about the installation.

## How to use the tool ##

This tool was originally intended as an image organizer and web publisher, the first draft was written in Python and then, when Go came along, it became a learning exercise to get in touch with the Go language. Being a learning tool there is no claim to perfection, hopefully it might still be useful to someone.

### Applied to Piwigo ###

Whilst one of the many photo sharing websites Piwigo is one of the best at present.
Refer to its website to gather your own impression:

[Piwigo home page](http://piwigo.org/)

The default settings apply to Piwigo gallery structure but is should be fairly easy to extend or adjust it to any other website.

### The core features of the tool are: ###

  1. Cross-platform: It runs -once compiled- on any operating system supported by Go, Windows and Linux are currently supported.
  1. It is open-source, like any related piece of software it uses.
  1. It uses ImageMagick which is a great cross-platform image conversion tool.
  1. Extracts photo information via an open-source exif library.
  1. Once compiled for an operating system it does not require a runtime or specific tools other than ImageMagick, or your own image processing tool if you prefer.

### A command line example ###

```go

./goconvert -f imagefolderpath -c mycollectionname
```

### Compiled versions: ###
  * [Windows executable](http://goconvert.googlecode.com/files/goconvert.exe)