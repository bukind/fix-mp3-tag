package main

import (
	"flag"
	"fmt"
	"github.com/bogem/id3v2"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"os"
	"strings"
)

// The list of supported tags, get more of them from
// https://pkg.go.dev/github.com/bogem/id3v2/v2#pkg-variables.
var (
	verbose = flag.Int("v", 0, "Increase verbosity")
	doWrite = flag.Bool("w", false, "Write converted frames back")

	supportedTags = []string{
		"Artist",
		"Content type",
		"Title",
		"Content group description",
		"Band",
		"Album",
	}
)

// check if the argument is UTF-8
func isUtf(s string) bool {
	_, _, err := encoding.UTF8Validator.Transform([]byte(s), []byte(s), true)
	return err == nil
}

// check if the argument is Russian cyrillic utf-8 chars
func isCyr(s string) bool {
	for _, c := range s {
		switch {
		case 0 <= c && c <= 0x7f:
			// ascii
		case 0x410 <= c && c <= 0x44f:
			// basic russian
		case c == 0x401 || c == 0x451:
			// yo
		default:
			return false
		}
	}
	return true
}

// the interface similar to that of encoding.Decoder and encoding.Encoder
type StringTrans interface {
	String(src string) (string, error)
}

// Apply a number of transformations to the string.
func decode(src string, tlist ...StringTrans) (string, error) {
	for _, f := range tlist {
		dst, err := f.String(src)
		if err != nil && len(src) > 4 {
			// Also try transforming with one byte at the end stripped.
			src2 := src[0 : len(src)-1]
			dst, err = f.String(src2)
			if err != nil {
				if *verbose > 1 {
					fmt.Printf("  failed!\n")
				}
				return "", err
			}
		}
		if *verbose > 1 {
			fmt.Printf("  converted %s => %s\n", dump(src), dump(dst))
		}
		src = dst
	}
	if isUtf(src) && isCyr(src) {
		return src, nil
	}
	if *verbose > 1 {
		fmt.Printf("  failed (bad result)!\n")
	}
	return "", fmt.Errorf("bad result of conversion")
}

// Show the string with both symbol and hex representation.
func dump(in string) string {
	return fmt.Sprintf("%q [% x]", in, []byte(in))
}

// Extract supported frames into a map.
func extractFrames(tag *id3v2.Tag) (map[string]id3v2.TextFrame, error) {
	out := make(map[string]id3v2.TextFrame)
	for _, t := range supportedTags {
		tf := tag.GetTextFrame(tag.CommonID(t))
		if tf.Text != "" {
			if *verbose > 1 {
				fmt.Printf(" tag %s found, encoding %v, text: %s\n", t, tf.Encoding, dump(tf.Text))
			}
			out[t] = tf
		}
	}
	return out, nil
}

// Attempt to convert frames to utf8.
// Only those that can be converted are returned.
func convertFrames(frames map[string]id3v2.TextFrame) map[string]id3v2.TextFrame {
	out := make(map[string]id3v2.TextFrame)
	for key, tf := range frames {
		if !tf.Encoding.Equals(id3v2.EncodingISO) {
			if *verbose > 1 {
				fmt.Printf(" frame %v encoding is not ISO, skipping\n", tf)
			}
			continue
		}

		win := charmap.Windows1251.NewDecoder()
		enc := charmap.Windows1251.NewEncoder()
		iso := charmap.ISO8859_1.NewEncoder()

		combinations := []struct {
			name  string
			tlist []StringTrans
		}{
			{"win", []StringTrans{win}},
			{"enc-iso-win", []StringTrans{enc, iso, win}},
			{"iso-win", []StringTrans{iso, win}},
			{"iso", []StringTrans{iso}}, // for incorrect encoding field.
		}

		value := strings.TrimSpace(tf.Text)
		if isUtf(value) && isCyr(value) {
			// already normal tag
			if *verbose > 1 {
				fmt.Printf(" frame %v is already normal\n", tf)
			}
			continue
		}

		var newvals []string
		for _, cmb := range combinations {
			if *verbose > 1 {
				fmt.Printf(" attempting %s...\n", cmb.name)
			}
			val, err := decode(value, cmb.tlist...)
			if err != nil {
				continue
			}
			if *verbose > 1 {
				fmt.Printf(" frame %v converted to %q\n", tf, val)
			}
			newvals = append(newvals, val)
		}
		switch len(newvals) {
		case 0:
			if *verbose > 1 {
				fmt.Printf(" could not convert frame %s\n", key)
			}
		case 1:
			out[key] = id3v2.TextFrame{
				Encoding: id3v2.EncodingUTF8,
				Text:     newvals[0],
			}
		case 2:
			if *verbose > 1 {
				fmt.Printf(" ambiguous conversion for frame %s -- possible results: %v", key, newvals)
			}
		}
	}
	return out
}

// Save frames back into mp3.
func saveFrames(tag *id3v2.Tag, frames map[string]id3v2.TextFrame) error {
	for key, tf := range frames {
		tag.AddTextFrame(tag.CommonID(key), tf.Encoding, tf.Text)
	}
	return tag.Save()
}

func processFile(path string) error {
	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err != nil {
		return err
	}
	defer tag.Close()
	if *verbose > 0 {
		fmt.Printf("processing file %q...\n", path)
	}

	frames, err := extractFrames(tag)
	if err != nil {
		return err
	}
	if *verbose > 0 {
		fmt.Printf(" frames found: %v\n", frames)
	}

	frames = convertFrames(frames)
	if len(frames) == 0 {
		if *verbose > 0 {
			fmt.Printf(" no broken frames found, nothing to write back\n")
		}
		return nil
	}
	if *verbose > 0 {
		fmt.Printf(" frames to write: %v\n", frames)
	}
	if *doWrite {
		if err := saveFrames(tag, frames); err != nil {
			fmt.Printf("failed %q: %s\n", path, err.Error())
		}
	}
	return nil
}

func main() {
	flag.Parse()
	if !*doWrite && *verbose <= 0 {
		// In a dry-run mode we'd like to see at least some output.
		*verbose = 1
	}

	if len(flag.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "please specify at least one mp3")
		os.Exit(1)
	}

	for _, image := range flag.Args() {
		if err := processFile(image); err != nil {
			fmt.Fprintln(os.Stderr, "%s: failed: %v", image, err)
		}
	}
}
