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
	"github.com/olekukonko/tablewriter"
)

const timeLayout = "2006-01-02"

// Options represents command line options that are shared across sub-commands.
type Options struct {
	Config string `short:"f" long:"config" description:"Config file" value-name:"FILE" default:"~/.journalrc"`
	Color  string `short:"c" long:"color" description:"When to use colors in output. Default is to use colors if stdout is a TTY" default:"auto" choice:"always" choice:"never" choice:"auto"`
	IsPipe bool
	Writer io.Writer
	Log    *log.Logger
}

// Import represents options for the import sub-command.
type Import struct {
	Options
	Reader string `short:"r" long:"reader" description:"Name of reader to use when importing data" choice:"csv" choice:"komplett" choice:"komplettsparing" choice:"norwegian" choice:"auto" default:"auto"`
	Args   struct {
		Account string `description:"Account number" positional-arg-name:"account-number"`
		File    string `description:"File containing records to import" positional-arg-name:"import-file"`
	} `positional-args:"yes" required:"yes"`
}

// Export represents options for the export sub-command.
type Export struct {
	Options
	Since string `short:"s" long:"since" description:"Print records since this date" value-name:"YYYY-MM-DD"`
	Until string `short:"u" long:"until" description:"Print records until this date" value-name:"YYYY-MM-DD"`
	Args  struct {
		Account string `description:"Account number" positional-arg-name:"account-number"`
	} `positional-args:"yes"`
}

// Accounts reprents options for the acct sub-command
type Accounts struct {
	Options
}

// List represents options for the export sub-command.
type List struct {
	Options
	Explain bool   `short:"e" long:"explain" description:"Print all records and their group"`
	Since   string `short:"s" long:"since" description:"Print records since this date" value-name:"YYYY-MM-DD"`
	Until   string `short:"u" long:"until" description:"Print records until this date" value-name:"YYYY-MM-DD"`
	OrderBy string `short:"o" long:"order-by" description:"Print records ordered by a specific field" choice:"sum" choice:"date" choice:"name" default:"sum"`
	Args    struct {
		Account string `description:"Only print records for given account number" positional-arg-name:"account-number"`
	} `positional-args:"yes"`
}

// NewLogger creates a new preconfigured logger.
func NewLogger(w io.Writer) *log.Logger { return log.New(w, "journal: ", 0) }

func maxLen(column int, rows [][]string) int {
	max := 0
	for _, row := range rows {
		if l := len(sgrTrim.Replace(row[column])); l > max {
			max = l
		}
	}
	return max
}

func parseTime(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, nil
	}
	return time.Parse(timeLayout, s)
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

// Execute imports records into the journal from a file.
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

	rs, err := j.ReadFile(i.Reader, f)
	if err != nil {
		return err
	}

	writes, err := j.Write(i.Args.Account, rs)
	i.Log.Printf("created %d new account(s)", writes.Account)
	i.Log.Printf("imported %d new record(s) out of %d total", writes.Record, len(rs))
	return err
}

// Execute lists known accounts.
func (a *Accounts) Execute(args []string) error {
	j, err := journal.FromConfig(a.Config)
	if err != nil {
		return err
	}

	as, err := j.Accounts()
	if err != nil {
		return err
	}

	table := tablewriter.NewWriter(a.Writer)
	table.SetHeader([]string{"Number", "Name", "Records"})
	table.SetAutoWrapText(false)
	for _, a := range as {
		table.Append([]string{
			a.Number,
			a.Name,
			strconv.FormatInt(a.Records, 10),
		})
	}
	table.Render()

	return nil
}

// Execute lists records contained in the journal.
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

	rgs := j.Assort(rs)

	if err := l.sort(rgs); err != nil {
		return err
	}

	account := "all accounts"
	if l.Args.Account != "" {
		account = "account " + l.Args.Account
	}
	l.Log.Printf("displaying records for %s between %s and %s", account, s.Format(timeLayout), u.Format(timeLayout))
	if len(rgs) == 0 {
		l.Log.Printf("0 records found")
		return nil
	}

	if l.Explain {
		l.printAll(rgs, j.FormatAmount)
	} else {
		l.printGroups(rgs, j.FormatAmount)
	}
	return nil
}

func (l *List) sort(rgs []record.Group) error {
	switch l.OrderBy {
	case "name":
		break // default sorting in journal
	case "date":
		if !l.Explain {
			return fmt.Errorf("grouped output cannot be ordered by date")
		}
	default:
		sort.Slice(rgs, func(i, j int) bool { return rgs[i].Sum() < rgs[j].Sum() })
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

func (l *List) printGroups(rgs []record.Group, fmtAmount func(int64) string) {
	table := tablewriter.NewWriter(l.Writer)
	var rows [][]string
	headers := []string{"Group", "Records", "Sum", "Budget", "Balance", "Balance bar"}
	rows = append(rows, headers)
	table.SetHeader(headers)
	table.SetAutoWrapText(false)
	alignments := make([]int, len(headers))
	// Align all columns, except the first and last, to the right
	for i := 1; i < len(alignments)-1; i++ {
		alignments[i] = tablewriter.ALIGN_RIGHT
	}
	table.SetColumnAlignment(alignments)
	var (
		totalRecords = 0
		totalBalance int64
		totalSum     int64
		totalBudget  int64
	)
	s := sgr{
		min:     record.MinBalance(rgs),
		max:     record.MaxBalance(rgs),
		enabled: l.colorize(),
	}
	for _, rg := range rgs {
		var (
			records = len(rg.Records)
			balance = rg.Balance()
			sum     = rg.Sum()
			budget  = rg.Budget()
			c, d    = s.color(balance)
		)
		totalRecords += records
		totalBalance += balance
		totalSum += sum
		totalBudget += budget
		row := []string{
			rg.Name,
			strconv.Itoa(records),
			fmtAmount(sum),
			fmtAmount(budget),
			c + fmtAmount(balance) + d,
			s.bar(balance),
		}
		rows = append(rows, row)
		table.Append(row)
	}

	footer := tablewriter.NewWriter(l.Writer)
	footer.SetColumnAlignment(alignments)
	footer.SetAutoWrapText(false)
	footer.SetBorders(tablewriter.Border{Left: true, Right: true, Bottom: true})
	for column := range headers {
		footer.SetColMinWidth(column, maxLen(column, rows))
	}
	c, d := s.color(totalBalance)
	footer.Append([]string{
		"Total",
		strconv.Itoa(totalRecords),
		fmtAmount(totalSum),
		fmtAmount(totalBudget),
		c + fmtAmount(totalBalance) + d,
		s.bar(totalBalance),
	})

	table.Render()
	footer.Render()
}

func (l *List) colorize() bool {
	switch l.Color {
	case "always":
		return true
	case "never":
		return false
	}
	return !l.Options.IsPipe
}

func (l *List) printAll(rgs []record.Group, fmtAmount func(int64) string) {
	table := tablewriter.NewWriter(l.Writer)
	table.SetHeader([]string{"Group", "Account", "Account name", "ID", "Date", "Text", "Amount"})
	table.SetColumnAlignment([]int{
		0, 0, 0, 0, 0, 0, tablewriter.ALIGN_RIGHT,
	})
	for _, rg := range rgs {
		for _, r := range rg.Records {
			row := []string{
				rg.Name,
				r.Account.Number,
				r.Account.Name,
				r.ID(),
				r.Time.Format("2006-01-02"),
				r.Text,
				fmtAmount(r.Amount),
			}
			table.Append(row)
		}
	}
	table.Render()
}

// Execute exports records from the journal.
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

	periods := j.AssortPeriod(rs, func(t time.Time) time.Time {
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	})
	return j.Export(e.Writer, periods, "2006-01")
}
