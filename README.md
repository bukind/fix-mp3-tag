# fix-mp3-tag

## About

A lot of Cyrillic mp3 tags are broken by f%&amp;king windows encoding.
This binary will try to fix this.


## Dependency

It is written in Go, based on github.com/bogem/id3v2 library.
To install you need a git client, and a go compiler to build.

## Installation

* setup the environment:

  ```
  mkdir -p devel/golang/{bin,src}
  export GOPATH=$PWD/devel/golang
  ```

* bring the package from github:

  ```
  cd $GOPATH
  git clone https://github.com/bukind/fix-mp3-tag.git src/github.com/bukind/fix-mp3-tag
  ```

* install the binary:

  ```
  go install github.com/bukind/fix-mp3-tag
  ```

  The binary will be created as $GOPATH/bin/fix-mp3-tag.

## Usage

By default the program runs in a dry-run mode:

```
$GOPATH/bin/fix-mp3-tag <mp3file>...
```

The program will try to decode the id3 tags of the mp3 using the
combination of the cp1251 and iso8859-1 encodings and print what it is
going to write back.

Then, you can run it to actually write those tags back:

```
$GOPATH/bin/fix-mp3-tag -w <mp3file>...
```

There is a verbosity flag `-v=INT` to see some debugging messages.

## License

GPL-3
