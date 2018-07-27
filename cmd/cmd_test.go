package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
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

func tempFile(data string) (string, error) {
	f, err := ioutil.TempFile("", "journal")
	if err != nil {
		return "", err
	}
	return f.Name(), ioutil.WriteFile(f.Name(), []byte(data), 0644)
}

func testFiles(t *testing.T) (string, string, string) {
	db, err := ioutil.TempFile("", "journal")
	if err != nil {
		t.Fatal(err)
	}

	conf, err := tempFile(fmt.Sprintf(conf, db.Name()))
	if err != nil {
		t.Fatal(err)
	}

	data, err := tempFile(data)
	if err != nil {
		t.Fatal(err)
	}

	return db.Name(), conf, data
}

func TestImport(t *testing.T) {
	dbName, confName, dataName := testFiles(t)
	defer os.Remove(confName)
	defer os.Remove(dbName)
	defer os.Remove(dataName)

	var stdout, stderr bytes.Buffer
	opts := Options{Config: confName, Writer: &stdout, Log: NewLogger(&stderr)}
	imp := Import{Options: opts, Reader: "csv"}
	imp.Args.Account = "1234.56.78900"
	imp.Args.File = dataName

	if err := imp.Execute(nil); err != nil {
		t.Fatal(err)
	}

	want := `journal: created 1 new account(s)
journal: imported 3 new record(s) out of 3 total
`
	if got := stderr.String(); want != got {
		t.Errorf("want %q, got %q", want, got)
	}

	if want, got := "", stdout.String(); want != got {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestExport(t *testing.T) {
	dbName, confName, dataName := testFiles(t)
	defer os.Remove(confName)
	defer os.Remove(dbName)
	defer os.Remove(dataName)

	var stdout, stderr bytes.Buffer
	opts := Options{Config: confName, Writer: &stdout, Log: NewLogger(&stderr)}

	imp := Import{Options: opts, Reader: "csv"}
	imp.Args.Account = "1234.56.78900"
	imp.Args.File = dataName

	if err := imp.Execute(nil); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()

	export := Export{Options: opts, Since: "2017-01-01"}
	export.Args.Account = imp.Args.Account

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
	dbName, confName, dataName := testFiles(t)
	defer os.Remove(confName)
	defer os.Remove(dbName)
	defer os.Remove(dataName)

	var stdout, stderr bytes.Buffer
	opts := Options{Config: confName, Writer: &stdout, Log: NewLogger(&stderr), Color: "never"}
	imp := Import{Options: opts, Reader: "csv"}
	imp.Args.Account = "1234.56.78900"
	imp.Args.File = dataName
	if err := imp.Execute(nil); err != nil {
		t.Fatal(err)
	}
	ls := List{Options: opts, Since: "2017-01-01"}
	if err := ls.Execute(nil); err != nil {
		t.Fatal(err)
	}

	want := `+------------+---------+---------+--------+-------+----------+--------------------------------+
|   GROUP    | RECORDS |   SUM   | BUDGET | SLACK | BALANCE  |          BALANCE BAR           |
+------------+---------+---------+--------+-------+----------+--------------------------------+
| Everything |       3 | 1337.00 |   0.00 |  0.00 | -1337.00 | ----------------               |
+------------+---------+---------+--------+-------+----------+--------------------------------+
`
	if got := stdout.String(); want != got {
		t.Errorf("want %q, got %q", want, got)
	}

	ls.Explain = true
	stdout.Reset()

	if err := ls.Execute(nil); err != nil {
		t.Fatal(err)
	}

	want = `+------------+---------------+--------------+------------+------------+---------------+---------+
|   GROUP    |    ACCOUNT    | ACCOUNT NAME |     ID     |    DATE    |     TEXT      | AMOUNT  |
+------------+---------------+--------------+------------+------------+---------------+---------+
| Everything | 1234.56.78900 | My account 1 | 66e7fcce66 | 2017-03-10 | Transaction 2 |  -42.00 |
| Everything | 1234.56.78900 | My account 1 | 11485ce462 | 2017-04-20 | Transaction 3 |   42.00 |
| Everything | 1234.56.78900 | My account 1 | ed5c019f5d | 2017-02-01 | Transaction 1 | 1337.00 |
+------------+---------------+--------------+------------+------------+---------------+---------+
`
	if got := stdout.String(); want != got {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestBalanceBar(t *testing.T) {
	var tests = []struct {
		balance int64
		min     int64
		max     int64
		color   bool
		out     string
	}{
		{4000, 0, 10000, true, "                \x1b[0m\x1b[7m\x1b[1;31m            \x1b[0m  "},
		{-5000, -10000, 10000, true, "        \x1b[7m\x1b[1;32m        \x1b[0m              "},
		{-2000, -5000, -10000, true, "                \x1b[0m              "},
		{0, 0, 1000, true, "                \x1b[0m              "},
		{4000, 0, 10000, false, "                ++++++++++++  "},
		{-5000, -10000, 10000, false, "        --------              "},
		{0, 0, 1000, false, "                              "},
	}
	for i, tt := range tests {
		if got := balanceBar(tt.balance, tt.min, tt.max, tt.color); got != tt.out {
			t.Errorf("#%d: want '%q', got '%q'", i, tt.out, got)
		}
	}
}
