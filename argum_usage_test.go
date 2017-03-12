package argum

import (
	"bytes"
	"net/mail"
	"os"
	"testing"
	"time"
)

var testusage struct {
	S        string `argum:"req"`
	String   string
	Name     string `argum:"-n,--name" help:"name of something with multiline long long long long text, and text and some text"`
	Duration time.Duration
	Mail     []*mail.Address `argum:"-m"`

	Pos   string   `argum:"pos,req" help:"positional argument more more more text and text text text"`
	Slice []string `argum:"pos" help:"slice arguments"`
}

func TestUsage(t *testing.T) {
	os.Args = []string{"testing", "-s=s", "pos"}
	err := Parse(&testusage)
	if err != nil {
		t.Error(err)
	}

	w := bytes.NewBuffer([]byte{})
	Usage(w)
	t.Log(w.String())
	if w.Len() == 0 {
		t.Error("failed get usage line")
	}
}

func TestUsageArgumentHelp(t *testing.T) {
	os.Args = []string{"testing", "-s=s", "pos"}
	err := Parse(&testusage)
	if err != nil {
		t.Error(err)
	}

	w := bytes.NewBuffer([]byte{})
	ArgumentHelp(w)
	t.Log(w.String())
	if len(w.String()) == 0 {
		t.Error("failed get help")
	}
}
