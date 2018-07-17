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

type Export struct {
	globalOpts
	Log   *log.Logger
	Since string `short:"s" long:"since" description:"Print records since this date" value-name:"YYYY-MM-DD"`
	Until string `short:"u" long:"until" description:"Print records until this date" value-name:"YYYY-MM-DD"`
	Args  struct {
		Account string `description:"Account number" positional-arg-name:"account-number"`
	} `positional-args:"yes"`
}

type List struct {
	globalOpts
	Log     *log.Logger
	Explain bool   `short:"e" long:"explain" description:"Print all records and their group"`
	Since   string `short:"s" long:"since" description:"Print records since this date" value-name:"YYYY-MM-DD"`
	Until   string `short:"u" long:"until" description:"Print records until this date" value-name:"YYYY-MM-DD"`
	OrderBy string `short:"o" long:"order-by" description:"Print records ordered by a specific field" choice:"sum" choice:"date" choice:"name" default:"sum"`
	Args    struct {
		Account string `description:"Only print records for given account number" positional-arg-name:"account-number"`
	} `positional-args:"yes"`
}

func (i *Import) Execute(args []string) error {
	f, err := os.Open(i.Args.File)
	if err != nil {
		return err
	}
	defer f.Close()

	j, err := journal.FromConfig(i.Config)
	if err != nil {
		return err
	}

	rs, err := j.ReadFrom(i.Reader, f)
	if err != nil {
		return err
	}

	writes, err := j.Write(i.Args.Account, rs)
	i.Log.Printf("created %d new account(s)", writes.Account)
	i.Log.Printf("imported %d new record(s)", writes.Record)
	return err
}

func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse("2006-01-02", s)
}

func timeRange(since, until string) (time.Time, time.Time, error) {
	now := time.Now()
	s, err := parseTime(since)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	u, err := parseTime(until)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if s.IsZero() { // Default to start of month
		s = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	}
	if u.IsZero() {
		u = now
	}
	return s, u, nil
}

func (l *List) Execute(args []string) error {
	j, err := journal.FromConfig(l.Config)
	if err != nil {
		return err
	}

	s, u, err := timeRange(l.Since, l.Until)
	if err != nil {
		return err
	}

	rs, err := j.Read(l.Args.Account, s, u)
	if err != nil {
		return err
	}

	rgs := j.Group(rs)

	if err := l.sort(rgs); err != nil {
		return err
	}

	if l.Explain {
		writeAll(os.Stdout, rgs)
	} else {
		writeGroups(os.Stdout, rgs, s, u)
	}
	return nil
}

func (l *List) sort(rgs []journal.RecordGroup) error {
	switch l.OrderBy {
	case "name":
		break // default
	case "sum":
		sort.Slice(rgs, func(i, j int) bool { return rgs[i].Sum() < rgs[j].Sum() })
	default:
		if !l.Explain {
			return fmt.Errorf("grouped output cannot be ordered by date")
		}
	}
	// Sort records in each group
	for _, rg := range rgs {
		sort.Slice(rg.Records, func(i, j int) bool {
			switch l.OrderBy {
			case "name":
				return rg.Records[i].Text < (rg.Records[j].Text)
			case "date":
				return rg.Records[i].Time.After(rg.Records[j].Time)
			}
			return rg.Records[i].Amount < rg.Records[j].Amount
		})
	}
	return nil
}

func writeGroups(w io.Writer, rgs []journal.RecordGroup, since, until time.Time) {
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"Group", "Sum", "Records", "From", "To"})
	table.SetBorder(false)
	for _, rg := range rgs {
		var sum int64
		for _, r := range rg.Records {
			sum += r.Amount
		}
		row := []string{
			rg.Name,
			journal.FormatAmount(sum),
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
	table.SetHeader([]string{"Group", "Account", "Account name", "ID", "Date", "Text", "Amount"})
	table.SetBorder(false)
	for _, rg := range rgs {
		for _, r := range rg.Records {
			row := []string{
				rg.Name,
				r.Account.Number,
				r.Account.Name,
				r.ID(),
				r.Time.Format("2006-01-02"),
				r.Text,
				journal.FormatAmount(r.Amount),
			}
			table.Append(row)
		}
	}
	table.Render()
}

func (e *Export) Execute(args []string) error {
	j, err := journal.FromConfig(e.Config)
	if err != nil {
		return err
	}

	s, u, err := timeRange(e.Since, e.Until)
	if err != nil {
		return err
	}

	rs, err := j.Read(e.Args.Account, s, u)
	if err != nil {
		return err
	}

	byMonth := j.GroupFunc(rs, func(t time.Time) string { return t.Format("2006-01") })
	return j.Export(os.Stdout, byMonth)
}
