# fix-mp3-tag

## About

A lot of Cyrillic mp3's are broken by f%&amp;king windows encoding.
This binary will try to fix this.


## Dependency

You need id3info and id3tag applications, usually they are part of id3lib package for most Linux distribution.
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
  git clone https://github.com/bukind/fix-mp3-tag.git src/bukind/fix-mp3-tag
  ```

* install the binary:

  ```
  go install github.com/bukind/fix-mp3-tag
  ```

  The binary will be created as $GOPATH/bin/fix-mp3-tag.

## Usage

```
$GOPATH/bin/fix-mp3-tag <mp3file>...
```

The program will try to decode the id3 tags of the mp3 using the
combination of the cp1251 and iso8859-1 encodings.

## License

GPL-3
