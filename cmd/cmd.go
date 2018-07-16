package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"sort"
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
	Order   string `short:"o" long:"order" description:"Print records ordered by a specific field" choice:"sum" choice:"date" default:"sum"`
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

func since(now, t time.Time) time.Time {
	if t.IsZero() { // Default to start of month
		t = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	}
	return t
}

func until(now, t time.Time) time.Time {
	if t.IsZero() { // Default to current day
		t = now
	}
	return t
}

func (l *List) Execute(args []string) error {
	j, err := journal.FromConfig(l.Config)
	if err != nil {
		return err
	}

	now := time.Now()

	s, err := parseTime(l.Since)
	if err != nil {
		return err
	}
	s = since(now, s)

	u, err := parseTime(l.Until)
	if err != nil {
		return err
	}
	u = until(now, u)

	rs, err := j.Read(l.Args.Account, s, u)
	if err != nil {
		return err
	}

	rgs := j.Group(rs)

	if l.Explain {
		// Sort records in each group
		for _, rg := range rgs {
			sort.Slice(rg.Records, func(i, j int) bool {
				if l.Order == "date" {
					return rg.Records[i].Time.After(rg.Records[j].Time)
				}
				return rg.Records[i].Amount < rg.Records[j].Amount
			})

		}
	} else {
		if l.Order != "sum" {
			return fmt.Errorf("grouped output cannot be sorted by date")
		}
		sort.Slice(rgs, func(i, j int) bool { return rgs[i].Sum() < rgs[j].Sum() })
	}

	if l.Explain {
		writeAll(os.Stdout, rgs)
	} else {
		writeGroups(os.Stdout, rgs, s, u)
	}
	return nil
}

func formatAmount(n int64) string {
	s := strconv.FormatInt(n, 10)
	off := len(s) - 2
	return s[:off] + "," + s[off:]
}

func writeGroups(w io.Writer, rgs []journal.RecordGroup, since, until time.Time) {
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"Group", "Sum", "Records", "From", "To"})
	for _, rg := range rgs {
		var sum int64
		for _, r := range rg.Records {
			sum += r.Amount
		}
		row := []string{
			rg.Name,
			formatAmount(sum),
			strconv.Itoa(len(rg.Records)),
			since.Format("2006-01-02"),
			until.Format("2006-01-02"),
		}
		table.Append(row)
	}
	table.Render()
}

func writeAll(w io.Writer, rgs []journal.RecordGroup) {
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"Group", "Account", "Account name", "Date", "Text", "Amount"})
	table.SetAutoMergeCells(true)
	for _, rg := range rgs {
		for _, r := range rg.Records {
			row := []string{
				rg.Name,
				r.Account.Number,
				r.Account.Name,
				r.Time.Format("2006-01-02"),
				r.Text,
				formatAmount(r.Amount),
			}
			table.Append(row)
		}
	}
	table.Render()
}
