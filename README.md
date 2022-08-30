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

If some tags cannot be converted there will be a warning in the output.
Typically it can be either because the conversion could not find any
suitable result, or because there are too many suitable results.
The warning would contain a goodness value of the best result in range [0..1].
If the value is closer to 1, you can try to rerun the program with lower
threshold to see if it works.  It is recommended to do that in a
dry-run mode first, to see what would be the result:

```
$GOPATH/bin/fix-mp3-tag -t=0.8 <mp3file>...
```

There is also a verbosity flag `-v` to see some debugging messages.
Use larger values to have more detailed output, e.g. `-v=2`.

## License

GPL-3
