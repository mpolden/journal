package cmd

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

const conf = `
Database = "%s"

[[accounts]]
number = "1234.56.78900"
name = "My account 1"

[[groups]]
name = "Everything"
patterns = [".*"]
`

const data = `"01.02.2017";"01.02.2017";"Transaction 1";"1.337,00";"1.337,00";"";""
"10.03.2017";"10.03.2017";"Transaction 2";"-42,00";"1.295,00";"";""
"20.04.2017";"20.04.2017";"Transaction 3";"42,00";"1.337,00";"";""
`

type files struct {
	db   string
	conf string
	data string
	dir  string
}

func (f *files) removeAll() { os.RemoveAll(f.dir) }

func testFiles(t *testing.T) files {
	tempDir, err := ioutil.TempDir("", "journal")
	if err != nil {
		t.Fatal(err)
	}

	dbName := filepath.Join(tempDir, "db")
	confName := filepath.Join(tempDir, "conf")
	dataName := filepath.Join(tempDir, "data")

	if err := ioutil.WriteFile(confName, []byte(fmt.Sprintf(conf, dbName)), 0644); err != nil {
		t.Fatal(err)
	}

	if err := ioutil.WriteFile(dataName, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	return files{db: dbName, conf: confName, data: dataName, dir: tempDir}
}

func importFile(t *testing.T, f files, stdout, stderr io.Writer) {
	opts := Options{Config: f.conf, Writer: stdout, Log: NewLogger(stderr)}
	imp := Import{Options: opts, Reader: "csv"}
	imp.Args.Account = "1234.56.78900"
	imp.Args.Files = []string{f.data}
	if err := imp.Execute(nil); err != nil {
		t.Fatal(err)
	}
}

func TestImport(t *testing.T) {
	f := testFiles(t)
	defer f.removeAll()

	var stdout, stderr bytes.Buffer
	importFile(t, f, &stdout, &stderr)

	want := fmt.Sprintf(`journal: importing records from %s
journal: created 1 new account(s)
journal: imported 3 new record(s) out of 3 total
`, f.data)
	if got := stderr.String(); want != got {
		t.Errorf("want %q, got %q", want, got)
	}

	if want, got := "", stdout.String(); want != got {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestExport(t *testing.T) {
	f := testFiles(t)
	defer f.removeAll()
	importFile(t, f, ioutil.Discard, ioutil.Discard)

	var stdout, stderr bytes.Buffer
	export := Export{
		Options: Options{Config: f.conf, Writer: &stdout, Log: NewLogger(&stderr)},
		Since:   "2017-01-01",
	}

	if err := export.Execute(nil); err != nil {
		t.Fatal(err)
	}

	want := `2017-04,Everything,42.00
2017-03,Everything,-42.00
2017-02,Everything,1337.00
`
	if got := stdout.String(); got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestList(t *testing.T) {
	f := testFiles(t)
	defer f.removeAll()
	importFile(t, f, ioutil.Discard, ioutil.Discard)

	var stdout, stderr bytes.Buffer
	ls := List{
		Options: Options{Config: f.conf, Writer: &stdout, Log: NewLogger(&stderr), Color: "never"},
		Since:   "2017-01-01",
	}
	if err := ls.Execute(nil); err != nil {
		t.Fatal(err)
	}

	want := `+------------+---------+---------+--------+----------+--------------------------------+
|   GROUP    | RECORDS |   SUM   | BUDGET | BALANCE  |          BALANCE BAR           |
+------------+---------+---------+--------+----------+--------------------------------+
| Everything |       3 | 1337.00 |   0.00 | -1337.00 | ----------------               |
+------------+---------+---------+--------+----------+--------------------------------+
| Total      |       3 | 1337.00 |   0.00 | -1337.00 | ----------------               |
+------------+---------+---------+--------+----------+--------------------------------+
`
	if got := stdout.String(); want != got {
		fmt.Println(got)
		t.Errorf("want %q, got %q", want, got)
	}

	ls.Explain = "all"
	stdout.Reset()

	if err := ls.Execute(nil); err != nil {
		t.Fatal(err)
	}

	want = `+---------------+--------------+------------+------------+------------+---------------+---------+
|    ACCOUNT    | ACCOUNT NAME |     ID     |    DATE    |   GROUP    |     TEXT      | AMOUNT  |
+---------------+--------------+------------+------------+------------+---------------+---------+
| 1234.56.78900 | My account 1 | 66e7fcce66 | 2017-03-10 | Everything | Transaction 2 |  -42.00 |
| 1234.56.78900 | My account 1 | 11485ce462 | 2017-04-20 | Everything | Transaction 3 |   42.00 |
| 1234.56.78900 | My account 1 | ed5c019f5d | 2017-02-01 | Everything | Transaction 1 | 1337.00 |
+---------------+--------------+------------+------------+------------+---------------+---------+
`
	if got := stdout.String(); want != got {
		fmt.Println(got)
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestAccounts(t *testing.T) {
	f := testFiles(t)
	defer f.removeAll()
	importFile(t, f, ioutil.Discard, ioutil.Discard)

	var stdout, stderr bytes.Buffer
	acct := Accounts{
		Options: Options{Config: f.conf, Writer: &stdout, Log: NewLogger(&stderr), Color: "never"},
	}
	if err := acct.Execute(nil); err != nil {
		t.Fatal(err)
	}

	want := `+---------------+--------------+---------+
|    NUMBER     |     NAME     | RECORDS |
+---------------+--------------+---------+
| 1234.56.78900 | My account 1 |       3 |
+---------------+--------------+---------+
`
	if got := stdout.String(); want != got {
		fmt.Println(got)
		t.Errorf("want %q, got %q", want, got)
	}
}
