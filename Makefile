include $(GOROOT)/src/Make.inc

PREREQ+=ftp4go
PREREQ+=exif
PREREQ+=goconf
PREREQ+=termios
PREREQ+=getpass

TARG=goconvert
GOFMT=gofmt -s -spaces=true -tabindent=false -tabwidth=4

GOFILES=\
  goconvert.go\
  settings.go\
  imgconvert.go\

ifeq ($(GOOS),windows)
# GOFILES+=os_win32.go
else
# GOFILES+=os_posix.go
endif

include $(GOROOT)/src/Make.cmd

format:
	${GOFMT} -w ${GOFILES}
	${GOFMT} -w goconvert_test.go
	${GOFMT} -w examples/hello.go

ftp4go:
	gomake -C ftp4go install

exif: exif/exif.go
	gomake -C exif install

goconf:
	gomake -C goconf install

termios:
	gomake -C termios install

getpass:
	gomake -C getpass install
	
clean: cleandeps

cleandeps:
	gomake -C ftp clean
	gomake -C exif clean
	gomake -C goconf clean
	gomake -C termios clean
	gomake -C getpass clean