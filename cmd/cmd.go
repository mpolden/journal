package cmd

import (
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/mpolden/journal/journal"
	"github.com/mpolden/journal/record"
	"github.com/mpolden/journal/record/komplett"
	"github.com/mpolden/journal/record/norwegian"
	"github.com/olekukonko/tablewriter"
)

const (
	timeLayout = "2006-01-02"
	darkGray   = "\033[1;30m"
	lightRed   = "\033[1;31m"
	lightGreen = "\033[1;32m"
	reverse    = "\033[7m"
	reset      = "\033[0m"
)

var ansiTrim = strings.NewReplacer(darkGray, "", lightRed, "", lightGreen, "", reverse, "", reset, "")

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
	Reader string `short:"r" long:"reader" description:"Name of reader to use when importing data" choice:"csv" choice:"komplett" choice:"komplettsparing" choice:"norwegian" default:"csv"`
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

	r, err := i.readerFrom(f)
	if err != nil {
		return err
	}

	rs, err := r.Read()
	if err != nil {
		return err
	}

	writes, err := j.Write(i.Args.Account, rs)
	i.Log.Printf("created %d new account(s)", writes.Account)
	i.Log.Printf("imported %d new record(s) out of %d total", writes.Record, len(rs))
	return err
}

func (i *Import) readerFrom(r io.Reader) (record.Reader, error) {
	var rr record.Reader
	switch i.Reader {
	case "csv":
		rr = record.NewReader(r)
	case "komplett", "komplettsparing":
		kr := komplett.NewReader(r)
		kr.JSON = i.Reader == "komplettsparing"
		rr = kr
	case "norwegian":
		rr = norwegian.NewReader(r)
	default:
		return nil, fmt.Errorf("invalid reader: %q", i.Reader)
	}
	return rr, nil
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

	l.Log.Printf("displaying records between %s and %s", s.Format(timeLayout), u.Format(timeLayout))

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
	headers := []string{"Group", "Records", "Sum", "Budget", "Slack", "Balance", "Balance bar"}
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
		min          = record.MinBalance(rgs)
		max          = record.MaxBalance(rgs)
		totalRecords = 0
		totalBalance int64
		totalSum     int64
		totalBudget  int64
		totalSlack   int64
	)
	for _, rg := range rgs {
		var (
			records = len(rg.Records)
			balance = rg.Balance()
			sum     = rg.Sum()
			budget  = rg.Budget()
			slack   = rg.Slack()
			c, d    = balanceColor(balance, rg.IsBalanced(), l.colorize())
		)
		totalRecords += records
		totalBalance += balance
		totalSum += sum
		totalBudget += budget
		totalSlack += slack
		row := []string{
			rg.Name,
			strconv.Itoa(records),
			fmtAmount(sum),
			fmtAmount(budget),
			fmtAmount(slack),
			c + fmtAmount(balance) + d,
			balanceBar(balance, min, max, l.colorize()),
		}
		rows = append(rows, row)
		table.Append(row)
	}

	footer := tablewriter.NewWriter(l.Writer)
	c, d := balanceColor(totalBalance, totalBalance == 0, l.colorize())
	footer.SetColumnAlignment(alignments)
	footer.SetAutoWrapText(false)
	footer.SetBorders(tablewriter.Border{Left: true, Right: true, Bottom: true})
	for column := range headers {
		footer.SetColMinWidth(column, maxLen(column, rows))
	}
	footer.Append([]string{
		"Total",
		strconv.Itoa(totalRecords),
		fmtAmount(totalSum),
		fmtAmount(totalBudget),
		fmtAmount(totalSlack),
		c + fmtAmount(totalBalance) + d,
		balanceBar(totalBalance, min, max, l.colorize()),
	})

	table.Render()
	footer.Render()
}

func maxLen(column int, rows [][]string) int {
	max := 0
	for _, row := range rows {
		if l := len(ansiTrim.Replace(row[column])); l > max {
			max = l
		}
	}
	return max
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

func balanceBar(balance, min, max int64, color bool) string {
	var (
		bars    int64 = 30
		barSize       = (max - min) / bars
	)
	var n int64
	if barSize > 0 {
		n = balance / barSize
	}
	sb := strings.Builder{}
	fill := ' '
	symbol := func(s rune, cs ...string) {
		if color {
			for _, c := range cs {
				sb.WriteString(c)
			}
		} else {
			fill = s
		}
	}
	for i, j, r := -bars/2, bars/2, false; i < j; i++ {
		if !r && i < 0 && i >= n {
			symbol('-', reverse, lightGreen)
			r = true
		} else if i > 0 {
			if !r && i <= n {
				symbol('+', reverse, lightRed)
				r = true
			} else if r && i > n {
				symbol(' ', reset)
				r = false
			}
		}
		sb.WriteRune(fill)
		if r && (i == 0 || i == j-1) {
			symbol(' ', reset)
			r = false
		}
	}
	return sb.String()
}

func balanceColor(balance int64, isBalanced, color bool) (string, string) {
	if !color {
		return "", ""
	}
	if isBalanced {
		return darkGray, reset
	} else if balance < 0 {
		return lightGreen, reset
	}
	return lightRed, reset
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
