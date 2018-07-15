package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/mpolden/journal/journal"
	"github.com/mpolden/journal/record"
	"github.com/mpolden/journal/record/komplett"
	"github.com/mpolden/journal/record/norwegian"
)

type globalOpts struct {
	Config string `short:"f" long:"config" description:"Config file" value-name:"FILE" default:"~/.journalrc"`
}

type Import struct {
	globalOpts
	Log    *log.Logger
	Reader string `short:"r" long:"reader" description:"Name of reader to use when importing data" choice:"csv" choice:"komplett" choice:"norwegian" default:"csv"`
	Dryrun bool   `short:"n" long:"dry-run" description:"Only print what would happen"`
	Args   struct {
		Account string `description:"Account number" positional-arg-name:"account-number"`
		File    string `description:"File containing records to import" positional-arg-name:"import-file"`
	} `positional-args:"yes" required:"yes"`
}

func (i *Import) Execute(args []string) error {
	j, err := journal.FromConfig(i.Config)
	if err != nil {
		return err
	}

	rs, err := i.readRecords()
	if err != nil {
		return err
	}

	return j.Write(i.Args.Account, rs)
}

func (i *Import) readRecords() ([]record.Record, error) {
	f, err := os.Open(i.Args.File)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var r record.Reader
	switch i.Reader {
	case "csv":
		r = record.NewReader(f)
	case "komplett":
		r = komplett.NewReader(f)
	case "norwegian":
		r = norwegian.NewReader(f)
	default:
		return nil, fmt.Errorf("invalid reader: %q", i.Reader)
	}
	return r.Read()
}
