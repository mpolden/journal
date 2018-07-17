package main

import (
	"log"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/mpolden/journal/cmd"
)

func main() {
	p := flags.NewParser(nil, flags.HelpFlag|flags.PassDoubleDash)
	log := log.New(os.Stderr, "journal: ", 0)
	opts := cmd.Options{Log: log, Writer: os.Stdout}

	imp := cmd.Import{Options: opts}
	if _, err := p.AddCommand("import", "Import records", "Imports records into the database.", &imp); err != nil {
		log.Fatal(err)
	}

	export := cmd.Export{Options: opts}
	if _, err := p.AddCommand("export", "Export records", "Export records to CSV.", &export); err != nil {
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
