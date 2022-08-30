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

var (
	verbose   = flag.Int("v", 0, "Increase verbosity")
	doWrite   = flag.Bool("w", false, "Write converted frames back")
	threshold = flag.Float64("t", 1, "Conversion threshold.  If some fields cannot be converted, try lower values, e.g. 0.8")
)

// The function counts the ratio of the correct UTF8 Cyrillic characters to the string length, in range [0..1].
// For empty string it returns 1.
// If the input is not UTF8, it returns 0.
func countCyr(s string) float64 {
	// Check that the input is UTF8.
	if _, _, err := encoding.UTF8Validator.Transform([]byte(s), []byte(s), true); err != nil {
		return 0
	}
	bad := 0
	total := 0
	for _, c := range s {
		total++
		switch {
		case 0 <= c && c <= 0x7f:
			// ascii
		case 0x410 <= c && c <= 0x44f:
			// basic russian
		case c == 0x401 || c == 0x451:
			// yo
		default:
			bad++
		}
	}
	if total == 0 {
		return 1
	}
	return float64(total-bad) / float64(total)
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
					fmt.Printf("  failed: %v\n", err)
				}
				return "", err
			}
		}
		if *verbose > 1 {
			fmt.Printf("  converted %s => %s\n", dump(src), dump(dst))
		}
		src = dst
	}
	return src, nil
}

// Show the string with both symbol and hex representation.
func dump(in string) string {
	return fmt.Sprintf("%q [% x]", in, []byte(in))
}

// Extract potential frames to convert into a map.
func extractFrames(tag *id3v2.Tag) (map[string]id3v2.TextFrame, error) {
	out := make(map[string]id3v2.TextFrame)
	// Get all frames
	for key, framers := range tag.AllFrames() {
		for _, frame := range framers {
			tf, ok := frame.(id3v2.TextFrame)
			if !ok {
				// This is not a text frame.
				// Since a single key cannot have different types of framers, we can break here.
				break
			}
			if tf.Text == "" {
				continue
			}
			// Check that we only have a single text frame.
			if len(framers) > 1 && *verbose > 0 {
				fmt.Printf(" Warning: the text tag %q has %d frames\n", key, len(framers))
				// We are going to use this frame anyway.
			}
			if !tf.Encoding.Equals(id3v2.EncodingISO) {
				// We don't have to convert non-ISO frames.
				if *verbose > 1 {
					fmt.Printf(" frame %q encoding is not ISO, skipping\n", key)
				}
				continue
			}
			if countCyr(strings.TrimSpace(tf.Text)) >= 1 {
				// If the result is already correct, skip it as well.
				if *verbose > 1 {
					fmt.Printf(" frame %q => %v is already correct\n", key, tf)
				}
				continue
			}
			if *verbose > 1 {
				fmt.Printf(" frame %q found, encoding %v, text: %s\n", key, tf.Encoding, dump(tf.Text))
			}
			out[key] = tf
			break
		}
	}
	return out, nil
}

// Attempt to convert frames to utf8.
// Only those that can be converted are returned.
func convertFrames(frames map[string]id3v2.TextFrame) map[string]id3v2.TextFrame {
	out := make(map[string]id3v2.TextFrame)

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

	for key, tf := range frames {
		if *verbose > 1 {
			fmt.Printf(" ------------------\n processing frame %q...\n", key)
		}
		value := strings.TrimSpace(tf.Text)
		best := 0.0
		var newvals []string
		for _, cmb := range combinations {
			if *verbose > 1 {
				fmt.Printf(" attempting %s...\n", cmb.name)
			}
			val, err := decode(value, cmb.tlist...)
			if err != nil {
				continue
			}
			goodness := countCyr(val)
			if goodness > best {
				best = goodness
			}
			if goodness < *threshold {
				if *verbose > 1 {
					fmt.Printf("  failed (bad result %f)!\n", goodness)
				}
				continue
			}
			if *verbose > 1 {
				fmt.Printf(" frame %q converted to %q, goodness %f\n", key, val, goodness)
			}
			newvals = append(newvals, val)
		}
		switch len(newvals) {
		case 0:
			fmt.Printf(" Warning: could not convert frame %s, best result is %f\n", key, best)
		case 1:
			out[key] = id3v2.TextFrame{
				Encoding: id3v2.EncodingUTF8,
				Text:     newvals[0],
			}
		default:
			fmt.Printf(" Warning: ambiguous conversion for frame %s -- got %d possible results, best is %f\n", key, len(newvals), best)
		}
	}
	return out
}

// Save frames back into mp3.
func saveFrames(tag *id3v2.Tag, frames map[string]id3v2.TextFrame) error {
	for key, tf := range frames {
		tag.AddTextFrame(key, tf.Encoding, tf.Text)
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
		fmt.Printf(" %d frames to convert found\n", len(frames))
	}

	frames = convertFrames(frames)
	if len(frames) == 0 {
		if *verbose > 0 {
			fmt.Printf(" cannot convert any frames, nothing to write back\n")
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
	if *threshold < 0.1 || *threshold > 1 {
		fmt.Fprintf(os.Stderr, "Invalid value of threshold (%f), must be in range [0.1, 1]\n", *threshold)
		os.Exit(1)
	}

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
