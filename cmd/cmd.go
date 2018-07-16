package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/mpolden/journal/journal"
	"github.com/mpolden/journal/record"
	"github.com/mpolden/journal/record/komplett"
	"github.com/mpolden/journal/record/norwegian"
	"github.com/olekukonko/tablewriter"
)

type globalOpts struct {
	Config string `short:"f" long:"config" description:"Config file" value-name:"FILE" default:"~/.journalrc"`
}

type Import struct {
	globalOpts
	Log    *log.Logger
	Reader string `short:"r" long:"reader" description:"Name of reader to use when importing data" choice:"csv" choice:"komplett" choice:"norwegian" default:"csv"`
	Args   struct {
		Account string `description:"Account number" positional-arg-name:"account-number"`
		File    string `description:"File containing records to import" positional-arg-name:"import-file"`
	} `positional-args:"yes" required:"yes"`
}

type List struct {
	globalOpts
	Log     *log.Logger
	Explain bool   `short:"e" long:"explain" description:"Print all records and their group"`
	Since   string `short:"s" long:"since" description:"Print records since this date" value-name:"YYYY-MM-DD"`
	Until   string `short:"u" long:"until" description:"Print records until this date" value-name:"YYYY-MM-DD"`
	Args    struct {
		Account string `description:"Only print records for given account number" positional-arg-name:"account-number"`
	} `positional-args:"yes"`
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

	writes, err := j.Write(i.Args.Account, rs)
	i.Log.Printf("created %d new account(s)", writes.Account)
	i.Log.Printf("imported %d new record(s)", writes.Record)
	return err
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

func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse("2006-01-02", s)
}

func (l *List) Execute(args []string) error {
	j, err := journal.FromConfig(l.Config)
	if err != nil {
		return err
	}

	since, err := parseTime(l.Since)
	if err != nil {
		return err
	}

	until, err := parseTime(l.Until)
	if err != nil {
		return err
	}

	rs, err := j.Read(l.Args.Account, since, until)
	if err != nil {
		return err
	}

	if l.Explain {
		writeAll(os.Stdout, j.Group(rs))
	} else {
		writeGroups(os.Stdout, j.Group(rs), since, until)
	}
	return nil
}

func formatAmount(n int64) string {
	s := strconv.FormatInt(n, 10)
	off := len(s) - 2
	return s[:off] + "," + s[off:]
}

func writeGroups(w io.Writer, rgs []journal.RecordGroup, since, until time.Time) {
	if until.IsZero() {
		until = time.Now()
	}
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"Group", "Sum", "From", "To"})
	for _, rg := range rgs {
		var sum int64
		for _, r := range rg.Records {
			sum += r.Amount
		}
		row := []string{rg.Name, formatAmount(sum), since.Format("2006-01-02"), until.Format("2006-01-02")}
		table.Append(row)
	}
	table.Render()
}

func writeAll(w io.Writer, rgs []journal.RecordGroup) {
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"Account", "Account name", "Date", "Record", "Amount", "Group"})
	for _, rg := range rgs {
		for _, r := range rg.Records {
			row := []string{
				r.Account.Number,
				r.Account.Name,
				r.Time.Format("2006-01-02"),
				r.Text,
				formatAmount(r.Amount),
				rg.Name,
			}
			table.Append(row)
		}
	}
	table.Render()
}
