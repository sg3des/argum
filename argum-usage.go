package argum

import (
	"bytes"
	"fmt"
	"io"
	"os"
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

func (s *structure) appendHelpOptions() {
	s.fields = append(s.fields, helparg)
	if Version != "" {
		s.fields = append(s.fields, versarg)
	}
}

func (s *structure) writeUsageHelp(w io.Writer) {
	s.writeUsage(os.Stdout)
	s.writeHelp(os.Stdout)
}

func (s *structure) writeUsage(w io.Writer) {
	usage := []string{"usage:", name}

	cs, sb, other := s.splitFieldsUsage()

	for _, f := range cs {
		usage = append(usage, f.usagePos())
	}

	if len(sb) > 0 {
		var shortbooleans string
		for _, f := range sb {
			shortbooleans += f.short[1:]
		}
		usage = append(usage, "[-"+shortbooleans+"]")
	}

	for _, f := range other {
		if f.pos {
			usage = append(usage, f.usagePos())
		} else {
			usage = append(usage, f.usageOpt())
		}
	}

	fmt.Fprintln(w, strings.Join(usage, " "))
}

func (s *structure) writeHelp(w io.Writer) {
	sel, cs, pos, opt := s.splitFieldsHelp()

	for _, f := range sel {
		f.writeHelpString(w, "")
	}

	if len(cs) > 0 {
		fmt.Fprintln(w, "\noptional commands:")
		for _, f := range cs {
			f.writeHelpString(w, "  ")
		}
	}

	if len(pos) > 0 {
		fmt.Fprintln(w, "\npositional:")
		for _, f := range pos {
			f.writeHelpString(w, "  ")
		}
	}

	if len(opt) > 0 {
		fmt.Fprintln(w, "\noptions:")
		for _, f := range opt {
			f.writeHelpString(w, "  ")
		}
	}
}

func (s *structure) splitFieldsUsage() (commands, shortbooleans, other []*field) {
	for _, f := range s.fields {
		switch {
		case f.cmd:
			commands = append(commands, f)
		case f.shortboolean:
			shortbooleans = append(shortbooleans, f)
		default:
			other = append(other, f)
		}
	}
	return
}

func (s *structure) splitFieldsHelp() (sel, cs, pos, opt []*field) {
	for _, f := range s.fields {
		switch {
		case f.sel:
			sel = append(sel, f)
		case f.cmd:
			cs = append(cs, f)
		case f.pos:
			pos = append(pos, f)
		default:
			opt = append(opt, f)
		}
	}
	return
}

func (f *field) usageOpt() string {
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

func (f *field) usagePos() string {
	var name string

	if len(f.variants) > 0 {
		name = strings.Join(f.variants, "|")
	} else {
		switch f.v.Kind() {
		case reflect.Slice:
			// name = fmt.Sprintf("<%s...>", f.field.Name)
			name = fmt.Sprintf("<%s...>", f.name)
		default:
			// name = fmt.Sprintf("<%s>", f.field.Name)
			name = fmt.Sprintf("<%s>", f.name)
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

	if len(f.variants) > 0 {
		return "[" + strings.Join(f.variants, "|") + "]"
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

func (f *field) writeHelpString(w io.Writer, prefix string) {
	switch {
	case f.cmd:
		f.writePositional(w, prefix)
		f.writeHelp(w)
		fmt.Fprintln(w)

		for _, subf := range f.s.fields {
			subf.writeHelpString(w, strings.Repeat(" ", len(prefix)+2))
		}
	case f.pos:
		f.writePositional(w, prefix)
		f.writeHelp(w)
		fmt.Fprintln(w)
	default:
		f.writeOption(w, prefix)
		f.writeHelp(w)
		fmt.Fprintln(w)
	}
}

func (f *field) writePositional(w io.Writer, prefix string) {
	fmt.Fprintf(w, "%s%s", prefix, f.name)

	if len(f.name) >= leftColLength {
		w.Write(newline)
	} else {
		w.Write(bytes.Repeat([]byte{' '}, leftColLength-len(f.name)-len(prefix)))
	}
}

func (f *field) writeOption(w io.Writer, prefix string) {
	fmt.Fprintf(w, "%s", prefix)

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

	w.Write([]byte(left))
	if len(left) >= leftColLength {
		w.Write(newline)
	} else {

		w.Write(bytes.Repeat([]byte{' '}, leftColLength-len(left)-len(prefix)))
	}
}

func (f *field) writeHelp(w io.Writer) {
	//write help
	n := writeWordWrap(w, f.help)

	//write default
	if f.def != "" {
		var def string
		if f.cmd {
			def = " [default]"
		} else {
			def = " [default: " + f.def + "]"
		}
		if n+len(def) > rightColLength {
			w.Write(newline)
		}
		n = writeWordWrap(w, def)
	}

	if len(f.variants) > 0 {
		variants := " [" + strings.Join(f.variants, "|") + "]"
		if n+len(variants) > rightColLength {
			w.Write(newline)
		}
		writeWordWrap(w, variants)
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
