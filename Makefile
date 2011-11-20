include $(GOROOT)/src/Make.inc

PREREQ+=ftp4go
PREREQ+=exif4go
PREREQ+=goconf

TARG=goconvert

#GOPATH=goconvert.googlecode.com/hg

GOFILES=\
  goconvert.go\
  settings.go\
  imgconvert.go\
  webserver.go\
  webgui.go\
  consolegui.go\
  
include $(GOROOT)/src/Make.cmd

exif4go:
	goinstall exif4go.googlecode.com/hg/exif4go

ftp4go:
	goinstall ftp4go.googlecode.com/hg/ftp4go

goconf:
	goinstall goconf.googlecode.com/hg