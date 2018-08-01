package main

import (
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/mpolden/journal/cmd"
)

func main() {
	p := flags.NewParser(nil, flags.HelpFlag|flags.PassDoubleDash)
	log := cmd.NewLogger(os.Stderr)
	isPipe, err := isPipe(os.Stdout)
	if err != nil {
		log.Fatal(err)
	}
	opts := cmd.Options{Log: log, Writer: os.Stdout, IsPipe: isPipe}

	imp := cmd.Import{Options: opts}
	if _, err := p.AddCommand("import", "Import records", "Imports records into the database.", &imp); err != nil {
		log.Fatal(err)
	}

	export := cmd.Export{Options: opts}
	if _, err := p.AddCommand("export", "Export records", "Export records to CSV.", &export); err != nil {
		log.Fatal(err)
	}

	acct := cmd.Accounts{Options: opts}
	if _, err := p.AddCommand("acct", "List accounts", "Display accounts in database", &acct); err != nil {
		log.Fatal(err)
	}

	list := cmd.List{Options: opts}
	if _, err := p.AddCommand("ls", "List records", "Display records in database", &list); err != nil {
		log.Fatal(err)
	}

	if _, err := p.Parse(); err != nil {
		log.Fatal(err)
	}
}

func isPipe(f *os.File) (bool, error) {
	stat, err := f.Stat()
	if err != nil {
		return false, err
	}
	return stat.Mode()&os.ModeCharDevice == 0, nil
}
