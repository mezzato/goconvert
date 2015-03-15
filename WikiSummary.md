# Introduction #

This command line tool does the following.
  1. resizes all the images in a folder and exports them in a target folder. It also creates thumbnail versions of the images and copies the original files in a subfolder of the target folder.
  1. Optionally uploads the files to an FTP server.
  1. It uses an initial set of questions to determine your settings and saves them to a file for reuse the next time.


---

# How to install and use the tool #

  1. Install ImageMagick to compress the images: http://www.imagemagick.org. Download one of the binary releases for your system.<br>Make sure that the ImageMagick executables are in your system PATH variable in order for the command line to find them.</li></ul>

<b>Note:</b> To add a folder to your system PATH on Windows 7 for instance: right-click on the Computer icon, choose "Properties", then "Advanced System Settings", finally on the tab "Advanced" the "Environment Variables..." button.<br>
<br>
Follow then the steps below according to your operating system.<br>
<br>
<h3>Linux ###
  1. You need to compile the source code, see below

### Windows ###
  1. Download the Windows executable from here: [Windows goconvert executable](http://goconvert.googlecode.com/files/goconvert.exe)
  1. Save the executable to a folder of your choice and add the folder to your PATH variable
  1. Open a command window, for instance via Start -> type "cmd". Type "cd folder\_where\_goconvert\_is". Type goconvert -h for help.


---

# How to compile the tool from the sources #

  1. Install Git and mercurial, msysgit and tortoishg on Windows for instance.
  1. You need to install the Go language in order to compile  the tool.
Refer to [Go installation page](http://golang.org/doc/install.html) for details.

## Linux ##

  * Just install Go

## Windows ##

  * Download the Google Go Windows port from here: [Go Windows port](http://code.google.com/p/gomingw/downloads/list)
  * Install [Cygwin](http://www.cygwin.com/) to make your life easier
  * Set/Create a user environment variable called HOME and set it to your local working folder
  * Install the GNU make compiler: http://gnuwin32.sourceforge.net/packages/make.htm
> Make sure that make is in the PATH variable

  1. Open a command line and type:
```
goinstall goconvert.googlecode.com/hg
```

  1. Type for a test:
```
goconvert -f test -c firsttest
```


---

# Development tips #

  * Install Eclipse
  * Install the [Gocode](https://github.com/nsf/gocode) autocompletion daemon for the Go language
  * Install the Goclipse plug-in for Eclipse: http://code.google.com/p/goclipse/