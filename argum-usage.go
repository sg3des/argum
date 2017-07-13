package argum

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"time"
)

const leftColLength = 26
const rightColLength = 80 - leftColLength

var (
	versarg = &field{long: "--version", help: "display version and exit"}
	helparg = &field{short: "-h", long: "--help", help: "display this help and exit"}
	newline = []byte(fmt.Sprintf("\n%26s", " "))
)

func (uf *userFields) helpRequested(osArgs []string) {
	for _, arg := range osArgs {
		if arg == "-h" || arg == "--help" {
			uf.fields = append(uf.fields, helparg)
			uf.fields = append(uf.fields, versarg)
			PrintHelp(0)
		}
	}

	if Version != "" {
		for _, arg := range osArgs {
			if arg == "--version" {
				fmt.Println(Version)
				os.Exit(0)
			}
		}
	}
}

//PrintHelp to stdout end exit
func PrintHelp(exitcode int) {
	Usage(os.Stdout)
	Help(os.Stdout)
	os.Exit(exitcode)
}

//Usage function print first usage string to io.Writer
func Usage(w io.Writer) {
	uf.usage(w)
}

func (uf *userFields) usage(w io.Writer) {
	options := []string{path.Base(os.Args[0])}

	var shortBool []string
	var usage []string

MODES:
	for _, fields := range uf.getModes() {
		for _, f := range fields {
			if s, ok := f.callMethod("Usage"); ok {
				w.Write([]byte(s.String()))
				continue MODES
			}
		}

		var mode []string
		var req bool
		for _, f := range fields {
			req = f.req
			mode = append(mode, f.name)
		}
		if req {
			options = append(options, strings.Join(mode, "|"))
		} else {
			options = append(options, fmt.Sprintf("[%s]", strings.Join(mode, "|")))
		}
	}

	//print options
	for _, f := range uf.fields {
		if f == helparg || f == versarg || f.pos || f.mode {
			continue
		}

		if f.v.Kind() == reflect.Bool && f.short != "" && !f.req {
			shortBool = append(shortBool, strings.TrimLeft(f.short, "-"))
			continue
		}

		usage = append(usage, f.usage())
	}

	if len(shortBool) != 0 {
		options = append(options, "[-"+strings.Join(shortBool, "")+"]")
	}

	options = append(options, usage...)

	//print pos arguments
	for _, f := range uf.fields {
		if f == helparg || !f.pos {
			continue
		}
		options = append(options, f.usagePositional())
	}

	w.Write([]byte(strings.Join(options, " ") + "\n"))
}

func (uf *userFields) getModes() map[string][]*field {
	var modes = make(map[string][]*field)

	for _, f := range uf.fields {
		if f == helparg || !f.mode {
			continue
		}
		modes[f.modeType] = append(modes[f.modeType], f)
	}

	return modes
}

func (f *field) usage() string {
	name := f.short
	if name == "" {
		name = f.long
	}

	val := f.valueType()
	if val == "" {
		return name
	}

	if f.req {
		return fmt.Sprintf("%s=%s", name, val)
	}

	return fmt.Sprintf("[%s=%s]", name, val)
}

func (f *field) usagePositional() string {
	var name string

	if len(f.opt) > 0 {
		name = strings.Join(f.opt, "|")
	} else {
		switch f.v.Kind() {
		case reflect.Slice:
			name = fmt.Sprintf("<%s...>", strings.ToLower(f.field.Name))
		default:
			name = fmt.Sprintf("<%s>", strings.ToLower(f.field.Name))
		}
	}

	if f.req {
		return fmt.Sprintf("%s", name)
	}

	return fmt.Sprintf("[%s]", name)
}

func (f *field) valueType() string {
	if !f.v.CanSet() {
		return ""
	}

	if len(f.opt) > 0 {
		return "[" + strings.Join(f.opt, "|") + "]"
	}

	switch f.v.Interface().(type) {
	case string:
		return "<s>"
	case int:
		return "<n>"
	case float32:
		return "<float>"
	case float64:
		return "<float>"
	case time.Duration:
		return "<time>"

	case []string:
		return "[s...]"
	case []int:
		return "[n...]"
	case []float32:
		return "[float...]"
	case []float64:
		return "[float...]"
	case []time.Duration:
		return "[time...]"
	}

	return ""
}

//ArgumentHelp print description of available options to io.Writer
func Help(w io.Writer) {
	uf.help(w)
}

func (uf *userFields) help(w io.Writer) {
	var headerPos, headerOpt bool

MODES:
	for name, fields := range uf.getModes() {
		for _, f := range fields {
			if s, ok := f.callMethod("Help"); ok {
				w.Write([]byte(s.String()))
				continue MODES
			}
		}

		name = filepath.Ext(name)[1:]
		var names []string
		var f *field
		for _, f = range fields {
			names = append(names, f.name)
		}
		if len(names) > 1 {
			w.Write([]byte(fmt.Sprintf("\n%s: %v\n", name, strings.Join(names, "|"))))
		} else if len(names) == 1 {
			w.Write([]byte(fmt.Sprintf("\n%s\n", names[0])))
		}
		for _, mf := range f.modeFields.fields {
			mf.writePositional("  ", w)
			mf.writeHelp(w)
			w.Write([]byte("\n"))
		}
	}

	for _, f := range uf.fields {
		if f.pos {
			if !headerPos {
				w.Write([]byte("\npositional:\n"))
				headerPos = true
			}
			f.writePositional("  ", w)
			f.writeHelp(w)
			w.Write([]byte("\n"))
		}

	}

	for _, f := range uf.fields {
		if !f.pos && !f.mode {
			if !headerOpt {
				w.Write([]byte("\noptions:\n"))
				headerOpt = true
			}
			f.writeOption(w)
			f.writeHelp(w)
			w.Write([]byte("\n"))
		}
	}
}

func (f *field) writePositional(prefix string, w io.Writer) {
	fmt.Fprintf(w, "%s%s", prefix, f.name)

	if len(f.name) >= leftColLength {
		w.Write(newline)
	} else {
		w.Write(bytes.Repeat([]byte{' '}, leftColLength-len(f.name)-len(prefix)))
	}
}

func (f *field) writeOption(w io.Writer) {
	var left string
	if f.short != "" {
		left += f.short
	} else {
		left += "   "
	}
	if f.long != "" {
		if f.short != "" {
			left += ","
		}
		left += " " + f.long
	}

	if val := f.valueType(); val != "" {
		left += "=" + val
	}

	w.Write([]byte("  " + left))
	if len(left) >= leftColLength {
		w.Write(newline)
	} else {

		w.Write(bytes.Repeat([]byte{' '}, leftColLength-len(left)-2))
	}
}

func (f *field) writeHelp(w io.Writer) {
	//write help
	n := writeWordWrap(w, f.help)

	//write default
	if f.def != "" {
		def := " [default: " + f.def + "]"
		if n+len(def) > rightColLength {
			w.Write(newline)
		}
		n = writeWordWrap(w, def)
	}

	if len(f.opt) > 0 {
		opt := " [" + strings.Join(f.opt, "|") + "]"
		if n+len(opt) > rightColLength {
			w.Write(newline)
		}
		writeWordWrap(w, opt)
	}
}

//write text by words and return length of last line
func writeWordWrap(w io.Writer, text string) (n int) {
	var rightWords []string
	var rightLen int
	words := strings.Split(text, " ")
	for _, s := range words {

		if rightLen+len(s) > rightColLength {
			line := []byte(strings.Join(rightWords, " "))

			w.Write(line)
			w.Write(newline)
			rightWords = nil
			rightLen = 0
			n = len(line)
		}

		if len(s) > rightColLength {
			line := []byte(s)
			w.Write(line)
			w.Write(newline)
			n = len(line)
			continue
		}

		rightWords = append(rightWords, s)
		rightLen += len(s) + 1
	}

	if len(rightWords) > 0 {
		line := []byte(strings.Join(rightWords, " "))
		w.Write(line)
		n = len(line)
	}

	return
}
