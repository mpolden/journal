# journal

[![Build Status](https://travis-ci.org/mpolden/journal.svg)](https://travis-ci.org/mpolden/journal)

`journal` is a program for storing and displaying financial records.

## Features

* Import financial records from multiple Norwegian banks, such as Eika Group
(most local banks), Storebrand, Bank Norwegian and Komplett Bank.
* Identify spending habits using automatic grouping of records.
* Define budgets for record groups.
* Export record groups for further processing in other programs.
* Take ownership of your financial records. All data is stored in a SQLite
  database.

## Installation

Building and installing `journal` requires the [Golang
compiler](https://golang.org/). With Go installed, `journal` can be installed
with:

`go get github.com/mpolden/journal/...`

This will build and install `journal` in `$GOPATH/bin/journal`.

For more information on building a Go project, see the [official Go
documentation](https://golang.org/doc/code.html).

## Example

My bank account exists at *Example Bank* with the account number
*1234.56.78900*. The bank supports export of records to CSV.

### Configuration

The first step is to configure our bank accounts and match groups.

`journal` uses the [TOML](https://github.com/toml-lang/toml) configuration
format and expects to find its configuration file in `~/.journalrc`.

Example:

```toml
database = "/home/user/journal.db"
comma = "."
defaultGroup = "* ungrouped *"

[[accounts]]
number = "1234.56.78900"
name = "Example Bank"

[[groups]]
name = "Public Transportation"
budgets = [
  -5000, # January
  -5000, # February
  -5000, # March
  -5000, # April
  -5000, # May
  -5000, # June
  0,     # July
  0,     # August
  -5000, # September
  -5000, # October
  -5000, # November
  -5000, # December
]
patterns = ["(?i)^Atb"]

[[groups]]
name = "Groceries"
budget = -100000
patterns = ["(?i)^Rema"]

[[groups]]
name = "One-off purchases"
ids = [
  "deadbeef",
  "cafebabe",
]

[[groups]]
name = "Ignored records"
pattern = ["^Spam"]
discard = true
```

`database` specifies where the SQLite database containing our records should be
stored.

`comma` is the decimal separator to use when displaying monetary amounts.
Defaults to `.`

`defaultGroup` is the default group name to use for unmatched records. Defaults
to `* ungrouped *`.

`[[accounts]]` declares known bank accounts. The section can be repeated to
define multiple accounts. Importing records for an unknown account is an error.

`[[groups]]` declares how records should be grouped together. `name` sets the
group name and `patterns` sets the list of regular expressions that match record
texts. The section can be repeated to declare multiple groups.

If any of the patterns in `patterns` match, the group is considered a match for
a given record. Matching follows the order declared in the configuration file,
where the first matching group wins.

Records can be pinned to a group using the `ids` key. This avoids the need to
create patterns for records that may only occur once. The `ids` key must be an
array of IDs to pin. Pinning takes precedence over matching patterns. Record IDs
can be found with `journal ls --explain`.

A monthly budget can be set per group by with the `budget` key. The budget is
specified as one-hundredth of the currency. `budget = -50000` means a budget of
*-500,00 NOK* .

When listing records for multiple months, the budget will be multiplied by the
number of months in the record time range. E.g. with `budget = -50000` and
records occurring in all months between *2018-05-13* and *2018-07-05*, the total
budget will be `3 * -50000 = -150000`.

It's also possible to set a custom budget for each month using the `budgets`
key. The value of `budgets` has to be an array of 12 numbers, one per month. If
`budgets` is unset, the value of `budget` will be used for all months.

Unwanted records may pollute the journal (e.g. inter-account transfers), these
records can be ignored entirely by setting `discard = true` on the matching
group.

### Export file

Most Norwegian banks support export to CSV. This can usually be done through
your bank's web interface.

CSV export example:

```csv
"01.06.2018";"01.06.2018";"Rema 1000";"-1.000,00";"5.000,00";"";""
"05.06.2018";"05.06.2018";"Rema 1000";"-500,00";"4.500,00";"";""
"07.06.2018";"07.06.2018";"Atb";"-35,00";"4.465,00";"";""
"09.06.2018";"09.06.2018";"Rema 1000";"-800,00";"3.665,00";"";""
"15.06.2018";"15.06.2018";"Atb";"-35,00";"3.630,00";"";""
"01.07.2018";"01.07.2018";"Rema 1000";"-250,00";"3.595,00";"";""
"02.07.2018";"02.07.2018";"Atb";"-35,00";"3.560,00";"";""
"05.07.2018";"05.07.2018";"Rema 1000";"-750,00";"2.810,00";"";""
"07.07.2018";"07.07.2018";"Atb";"-35,00";"2.775,00";"";""
"15.07.2018";"15.07.2018";"Atb";"-35,00";"2.740,00";"";""

```

### Importing records

The command `journal import` is used to import records. Given the export file
and configuration above, records can be imported with:

```
$ journal import 1234.56.78900 example.csv
journal: created 1 new account(s)
journal: imported 10 new record(s) out of 10 total
```

Records have now been stored in a SQLite database located in
`/home/user/journal.db`.

Repeating the import only imports records `journal` hasn't seen before, so
running the above command again imports 0 records:

```
$ journal import 1234.56.78900 example.csv
journal: created 0 new account(s)
journal: imported 0 new record(s) out of 10 total
```

Some banks have their own export format, in such cases the correct reader must
be specified when importing records. Example for *Bank Norwegian*:

`$ journal import -r norwegian 1234.56.78900 norwegian-export.xlsx`

See `journal import -h` for complete usage.
 
### Listing records

Now that we have imported records, they can be listed with `journal ls`:

```
$ journal ls
journal: displaying records for all accounts between 2018-07-01 and 2018-07-28
+-----------------------+---------+----------+----------+---------+--------------------------------+
|         GROUP         | RECORDS |   SUM    |  BUDGET  | BALANCE |          BALANCE BAR           |
+-----------------------+---------+----------+----------+---------+--------------------------------+
| Groceries             |       2 | -1000.00 | -1000.00 |    0.00 |                                |
| Public Transportation |       3 |  -105.00 |   -50.00 |   55.00 |                 ++++++++++++++ |
+-----------------------+---------+----------+----------+---------+--------------------------------+
| Total                 |       5 | -1105.00 | -1050.00 |   55.00 |                 ++++++++++++++ |
+-----------------------+---------+----------+----------+---------+--------------------------------+
```

By default, only records within the current month are listed and sorted
descending by sum.

Records are grouped together according to configured match groups. If we want to
understand a record grouping, we can list individual records and their group:

```
$ journal ls --explain
journal: displaying records for all accounts between 2018-07-01 and 2018-07-28
+---------------+--------------+------------+------------+-----------------------+-----------+---------+
|    ACCOUNT    | ACCOUNT NAME |     ID     |    DATE    |         GROUP         |   TEXT    | AMOUNT  |
+---------------+--------------+------------+------------+-----------------------+-----------+---------+
| 1234.56.78900 | Example Bank | 77c2a500e1 | 2018-07-05 | Groceries             | Rema 1000 | -750.00 |
| 1234.56.78900 | Example Bank | 8f864212ce | 2018-07-01 | Groceries             | Rema 1000 | -250.00 |
| 1234.56.78900 | Example Bank | 2e25c40379 | 2018-07-15 | Public Transportation | Atb       |  -35.00 |
| 1234.56.78900 | Example Bank | 84ca136809 | 2018-07-07 | Public Transportation | Atb       |  -35.00 |
| 1234.56.78900 | Example Bank | 5833456f0b | 2018-07-02 | Public Transportation | Atb       |  -35.00 |
+---------------+--------------+------------+------------+-----------------------+-----------+---------+
```

If we want show older records, date ranges can be specified using `--since` and
`--until`:

```
$ journal ls --since=2018-06-01 --until=2018-07-31
journal: displaying records for all accounts between 2018-06-01 and 2018-07-31
+-----------------------+---------+----------+----------+---------+--------------------------------+
|         GROUP         | RECORDS |   SUM    |  BUDGET  | BALANCE |          BALANCE BAR           |
+-----------------------+---------+----------+----------+---------+--------------------------------+
| Groceries             |       5 | -3300.00 | -4000.00 | -700.00 | ----------------               |
| Public Transportation |       5 |  -175.00 |  -100.00 |   75.00 |                 ++             |
+-----------------------+---------+----------+----------+---------+--------------------------------+
| Total                 |      10 | -3475.00 | -4100.00 | -625.00 | ----------------               |
+-----------------------+---------+----------+----------+---------+--------------------------------+
```

Note that the budget has been automatically adjusted to the number of months
that contain records.

Options also be combined:
```
$ journal ls --since=2018-01-01 --explain
journal: displaying records for all accounts between 2018-01-01 and 2018-07-28
+---------------+--------------+------------+------------+-----------------------+-----------+----------+
|    ACCOUNT    | ACCOUNT NAME |     ID     |    DATE    |         GROUP         |   TEXT    |  AMOUNT  |
+---------------+--------------+------------+------------+-----------------------+-----------+----------+
| 1234.56.78900 | Example Bank | e6c18424ba | 2018-06-01 | Groceries             | Rema 1000 | -1000.00 |
| 1234.56.78900 | Example Bank | b6b2496771 | 2018-06-09 | Groceries             | Rema 1000 |  -800.00 |
| 1234.56.78900 | Example Bank | 77c2a500e1 | 2018-07-05 | Groceries             | Rema 1000 |  -750.00 |
| 1234.56.78900 | Example Bank | 2e1aa3cf1a | 2018-06-05 | Groceries             | Rema 1000 |  -500.00 |
| 1234.56.78900 | Example Bank | 8f864212ce | 2018-07-01 | Groceries             | Rema 1000 |  -250.00 |
| 1234.56.78900 | Example Bank | 2e25c40379 | 2018-07-15 | Public Transportation | Atb       |   -35.00 |
| 1234.56.78900 | Example Bank | 84ca136809 | 2018-07-07 | Public Transportation | Atb       |   -35.00 |
| 1234.56.78900 | Example Bank | 5833456f0b | 2018-07-02 | Public Transportation | Atb       |   -35.00 |
| 1234.56.78900 | Example Bank | 2e8e1ac9e1 | 2018-06-15 | Public Transportation | Atb       |   -35.00 |
| 1234.56.78900 | Example Bank | 84c948c456 | 2018-06-07 | Public Transportation | Atb       |   -35.00 |
+---------------+--------------+------------+------------+-----------------------+-----------+----------+
```

See `journal ls -h` for complete usage.

### Export records

Record groups can be exported to
[CSV](https://en.wikipedia.org/wiki/Comma-separated_values) for further
processing in other programs such as a spreadsheet.

```
$ journal export --since=2018-01-01
2018-07,Groceries,-1000.00
2018-07,Public Transportation,-105.00
2018-06,Groceries,-2300.00
2018-06,Public Transportation,-70.00
```

See `journal export -h` for complete usage.
