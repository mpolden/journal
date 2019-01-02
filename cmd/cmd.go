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
	"github.com/olekukonko/tablewriter"
)

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
	Reader string `short:"r" long:"reader" description:"Name of reader to use when importing data" choice:"csv" choice:"komplett" choice:"norwegian" choice:"auto" default:"auto"`
	Args   struct {
		Account string   `description:"Account number" positional-arg-name:"account-number"`
		Files   []string `description:"File containing records to import" positional-arg-name:"import-file"`
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
	Explain string `short:"e" long:"explain" optional:"yes" optional-value:"all" value-name:"GROUP" description:"Print records in GROUP. Defaults to all groups"`
	Since   string `short:"s" long:"since" description:"Print records since this date" value-name:"YYYY-MM-DD"`
	Until   string `short:"u" long:"until" description:"Print records until this date" value-name:"YYYY-MM-DD"`
	Month   int    `short:"m" long:"month" description:"Print records in this month of the current year" value-name:"M"`
	OrderBy string `short:"o" long:"order" description:"Print records ordered by a specific field" choice:"sum" choice:"date" choice:"group" choice:"text" default:"sum"`
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

// Execute imports records into the journal from a file.
func (i *Import) Execute(args []string) error {
	j, err := journal.FromConfig(i.Config)
	if err != nil {
		return err
	}

	for _, file := range i.Args.Files {
		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()
		i.Log.Printf("importing records from %s", file)

		rs, err := j.ReadFile(i.Reader, f)
		if err != nil {
			return err
		}

		writes, err := j.Write(i.Args.Account, rs)
		i.Log.Printf("created %d new account(s)", writes.Account)
		i.Log.Printf("imported %d new record(s) out of %d total", writes.Record, len(rs))
		if err != nil {
			return err
		}
	}

	return nil
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

	clock := newClock()
	var s, u time.Time
	if l.Month != 0 {
		if l.Since != "" || l.Until != "" {
			return fmt.Errorf("--month cannot be combined with --since or --until")
		}
		s, u, err = clock.monthRange(l.Month)
	} else {
		s, u, err = clock.timeRange(l.Since, l.Until)

	}
	if err != nil {
		return err
	}

	sortField, err := l.sortField()
	if err != nil {
		return err
	}

	rs, err := j.Read(l.Args.Account, s, u)
	if err != nil {
		return err
	}

	account := "all accounts"
	if l.Args.Account != "" {
		account = "account " + l.Args.Account
	}
	l.Log.Printf("displaying records for %s between %s and %s", account, s.Format(timeLayout), u.Format(timeLayout))

	rgs := j.Assort(rs)
	if len(rgs) == 0 {
		l.Log.Printf("0 records found")
		return nil
	}

	if l.Explain != "" {
		l.printAll(rgs, l.Explain, j.FormatAmount, sortField)
	} else {
		l.printGroups(rgs, j.FormatAmount, sortField, record.Range{Since: s, Until: u})
	}
	return nil
}

func (l *List) sortField() (record.Field, error) {
	switch l.OrderBy {
	case "group":
		return record.GroupField, nil
	case "text":
		return record.NameField, nil
	case "date":
		if l.Explain == "" {
			return 0, fmt.Errorf("grouped output cannot be sorted by date")
		}
		return record.TimeField, nil
	case "sum", "":
		return record.SumField, nil
	}
	return 0, fmt.Errorf("invalid sort field: %q", l.OrderBy)
}

func (l *List) printGroups(rgs []record.Group, fmtAmount func(int64) string, sortField record.Field, r record.Range) {
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
		min:     record.MinBalance(rgs, r),
		max:     record.MaxBalance(rgs, r),
		enabled: l.colorize(),
	}
	record.SortGroup(rgs, sortField)
	for _, rg := range rgs {
		var (
			records = len(rg.Records)
			balance = rg.Balance(r)
			sum     = rg.Sum()
			budget  = rg.Budget(r)
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

	// Since length of strings containing SGR codes is longer than the display length, we can't use the built-in
	// footer support in tablewriter. The following code creates a new table without a top border, strips SGR codes
	// when calculating column width and renders it after the primary table. This gives the same visual effect as a
	// footer.
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

func (l *List) printAll(rgs []record.Group, group string, fmtAmount func(int64) string, sortField record.Field) {
	table := tablewriter.NewWriter(l.Writer)
	table.SetHeader([]string{"Account", "Account name", "ID", "Date", "Group", "Text", "Amount"})
	table.SetColumnAlignment([]int{
		0, 0, 0, 0, 0, 0, tablewriter.ALIGN_RIGHT,
	})
	gs := make(map[string]string)
	rs := []record.Record{}
	for _, rg := range rgs {
		for _, r := range rg.Records {
			gs[r.ID()] = rg.Name
			rs = append(rs, r)
		}
	}
	record.Sort(rs, sortField)
	var sum int64
	for _, r := range rs {
		groupName := gs[r.ID()]
		if group != "all" && group != groupName {
			continue
		}
		sum += r.Amount
		row := []string{
			r.Account.Number,
			r.Account.Name,
			r.ID(),
			r.Time.Format("2006-01-02"),
			groupName,
			r.Text,
			fmtAmount(r.Amount),
		}
		table.Append(row)
	}
	table.SetFooter([]string{"", "", "", "", "", "Total", fmtAmount(sum)})
	table.Render()
}

// Execute exports records from the journal.
func (e *Export) Execute(args []string) error {
	j, err := journal.FromConfig(e.Config)
	if err != nil {
		return err
	}

	clock := newClock()
	s, u, err := clock.timeRange(e.Since, e.Until)
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
