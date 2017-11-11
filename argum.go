package argum

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	//Version is global variable contains version of main application
	Version string
	name    string
	s       *structure
)

//MustParse parse os.Args for struct and fatal if it has error
func MustParse(i interface{}) {
	if err := Parse(i); err != nil {
		fmt.Println(err)
		PrintHelp(1)
	}
}

//Parse os.Args for incomimng struct and return error
func Parse(i interface{}) error {
	if Version != "" && contains(os.Args, "--version") {
		fmt.Println(Version)
		os.Exit(0)
	}

	name = filepath.Base(os.Args[0])
	if filepath.Ext(name) == ".test" {
		return nil
	}

	var err error
	s, err = prepareStructure(i)
	if err != nil {
		return fmt.Errorf("failed prepare structure, %s", err)
	}

	s.appendHelpOptions()

	if contains(os.Args[1:], "--help", "-h") {
		// INFO: temporary hidden help for specify command, as now output all help information
		// for _, f := range s.fields {
		// 	if f.command && contains(os.Args[1:], f.name) {
		// 		f.s.writeUsageHelp(os.Stdout)
		// 		os.Exit(0)
		// 	}
		// }

		s.writeUsageHelp(os.Stdout)
		os.Exit(0)
	}

	osArgs := prepareArgs(os.Args[1:])

	_, err = s.parseArgs(osArgs)
	return err
}

func prepareArgs(osArgs []string) (newArgs []string) {
	for _, arg := range osArgs {
		if matchEscape(arg) {
			newArgs = append(newArgs, trim(arg))
			continue
		}

		if strings.Contains(arg, "=") {
			ss := strings.SplitN(arg, "=", 2)

			newArgs = append(newArgs, ss[0])
			newArgs = append(newArgs, splitArgs(ss[1])...)
			continue
		}

		if matchShort(arg) && len(arg) > 2 {
			vals := splitShortArgs(arg[2:])
			arg = arg[:2]

			newArgs = append(newArgs, arg)
			newArgs = append(newArgs, vals...)
			continue
		}

		newArgs = append(newArgs, arg)
	}

	return
}

func splitArgs(s string) []string {
	var vals []string

	if matchEscape(s) {
		vals = []string{s}
	} else {
		vals = strings.Split(s, ",")
	}
	return vals
}

func splitShortArgs(s string) []string {
	if _, err := strconv.Atoi(s); err == nil {
		return []string{s}
	}

	if _, err := strconv.ParseFloat(s, 64); err == nil {
		return []string{s}
	}

	if _, err := time.ParseDuration(s); err == nil {
		return []string{s}
	}

	if _, err := strconv.ParseBool(s); err == nil {
		return []string{s}
	}

	var args []string
	for _, b := range s {
		args = append(args, "-"+string(b))
	}

	return args
}

//PrintHelp to stdout end exit
func PrintHelp(exitcode int) {
	s.writeUsageHelp(os.Stdout)
	os.Exit(exitcode)
}

func trim(s string) string {
	if len(s) > 1 {
		if s[0] == '"' && s[len(s)-1] == '"' {
			s = strings.Trim(s, "\"")
		}
		if s[0] == '\'' && s[len(s)-1] == '\'' {
			s = strings.Trim(s, "'")
		}
		if s[0] == '`' && s[len(s)-1] == '`' {
			s = strings.Trim(s, "`")
		}
	}

	return s
}

func contains(strslice []string, ss ...string) bool {
	for _, str := range strslice {
		for _, s := range ss {
			if str == s {
				return true
			}
		}
	}
	return false
}

func matchShort(arg string) bool {
	return len(arg) > 1 && arg[0] == '-' && arg[1] != '-'
}

func matchLong(arg string) bool {
	return len(arg) > 2 && arg[0:2] == "--" && arg[2] != '-'
}

func matchEscape(arg string) bool {
	return len(arg) > 1 && arg[0] == '"' && arg[len(arg)-1] == '"'
}
