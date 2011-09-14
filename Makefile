include $(GOROOT)/src/Make.inc

PREREQ+=ftp4go
PREREQ+=exif4go

TARG=goconvert

GOFILES=\
  goconvert.go\
  settings.go\
  imgconvert.go\
  
include $(GOROOT)/src/Make.cmd

exif4go:
	goinstall exif4go.googlecode.com/hg/exif4go

ftp4go:
	goinstall ftp4go.googlecode.com/hg/ftp4go
