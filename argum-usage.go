package argum

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
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

func (uf *userFields) appendHelpOptions() {
	uf.fields = append(uf.fields, helparg)
	if Version != "" {
		uf.fields = append(uf.fields, versarg)
	}
}

func (uf *userFields) helpRequested(osArgs []string) {
	if contains(osArgs, "--help", "-h") {
		for _, arg := range osArgs {
			if f, ok := uf.lookupCommand(arg); ok {
				f.commandFields.printHelpUsage(os.Stdout, f.name)
				os.Exit(0)
			}
		}

		uf.appendHelpOptions()
		PrintHelp(0)
	}

	if Version != "" {
		if contains(osArgs, "--version") {
			fmt.Println(Version)
			os.Exit(0)
		}
	}
}

//PrintHelp to stdout end exit
func PrintHelp(exitcode int) {
	uf.printHelpUsage(os.Stdout, "")
	os.Exit(exitcode)
}

func (uf *userFields) printHelpUsage(w io.Writer, name string) {
	fmt.Fprintf(w, "usage: %s ", path.Base(os.Args[0]))
	if name != "" {
		fmt.Fprintf(w, "%s ", name)
	}
	uf.usage(w)

	fmt.Fprintln(w)
	uf.help(w)
}

func (uf *userFields) usage(w io.Writer) {
	var options []string
	var shortBool []string
	var usage []string

	if commands := uf.getCommands(); len(commands) > 0 {
		fmt.Fprint(w, "<command> ")
	}

	//print options
	for _, f := range uf.fields {
		if f == helparg || f == versarg || f.pos || f.command {
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

	w.Write([]byte(strings.Join(options, " ")))
}

func (uf *userFields) getCommands() (commands []*field) {
	for _, f := range uf.fields {
		if f == helparg || !f.command {
			continue
		}
		commands = append(commands, f)
	}

	return
}

func (uf *userFields) lookupCommand(name string) (*field, bool) {
	for _, f := range uf.fields {
		if f.command && f.name == name {
			return f, true
		}
	}

	return nil, false
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

func (uf *userFields) help(w io.Writer) {
	if commands := uf.getCommands(); len(commands) > 0 {
		fmt.Fprint(w, "\ncommands:\n")
		for _, f := range commands {
			fmt.Fprintf(w, "  %s\n", f.name)
		}
	}

	if pos := uf.getPositionals(); len(pos) > 0 {
		fmt.Fprintln(w, "\npositional:")
		for _, f := range pos {
			f.writePositional(w, "  ")
			f.writeHelp(w)
			fmt.Fprintln(w, "")
		}
	}

	if opt := uf.getOptions(); len(opt) > 0 {
		fmt.Fprintln(w, "\noptions:")
		for _, f := range opt {
			f.writeOption(w)
			f.writeHelp(w)
			fmt.Fprintln(w, "")
		}
	}
}

func (uf *userFields) getPositionals() (fields []*field) {
	for _, f := range uf.fields {
		if f.pos {
			fields = append(fields, f)
		}
	}
	return
}

func (uf *userFields) getOptions() (fields []*field) {
	for _, f := range uf.fields {
		if !f.pos && !f.command {
			fields = append(fields, f)
		}
	}
	return
}

func (f *field) writePositional(w io.Writer, prefix string) {
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
