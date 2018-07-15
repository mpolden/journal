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

	imp := cmd.Import{Log: log}
	if _, err := p.AddCommand("import", "Import records", "Imports records into the database.", &imp); err != nil {
		log.Fatal(err)
	}
		log.Fatal(err)
	}

	if _, err := p.Parse(); err != nil {
		log.Fatal(err)
	}
}
