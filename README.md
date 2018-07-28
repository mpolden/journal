# journal

[![Build Status](https://travis-ci.org/mpolden/journal.svg)](https://travis-ci.org/mpolden/journal)

`journal` is a program for recording and displaying financial records.

## Features

* Import financial records from multiple Norwegian banks, such as Eika Group
(including local banks), Storebrand, Bank Norwegian and Komplett Bank
* Identify spending habits using automatic grouping of records
* Define budgets per record group
* Export grouped records for further processing in other programs
* Persistent SQL database of imported records

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
Database = "/home/user/journal.db"
Comma = "."
DefaultGroup = "*** UNMATCHED ***"

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

`Database` specifies where the SQLite database containing our records should be
stored.

`Comma` is the decimal separator to use when displaying monetary amounts. It
defaults to `.`

`DefaultGroup` is the default group name to use for unmatched records. Defaults
to `*** UNMATCHED ***`.

`[[accounts]]` declares known bank accounts. The section can be repeated to
define multiple accounts. Importing records for an unknown account is an error.

`[[groups]]` declares how records should be grouped together. `name` sets the
group name and `patterns` sets the list of regular expressions that match record
texts. The section can be repeated to declare multiple groups.

If any of the patterns in `patterns` match, the group is considered a match.
Matching follows the order declared in the configuration file, where the first
matching group wins.

To avoid having to create patterns for records that may only occur once, it's
possible to pin records to a group using the record ID. Pinning takes precedence
over matching patterns. Record IDs can be found with `journal ls --explain`.

A monthly budget can be set per group by with the `budget` key. When listing
records for multiple months, the budget will be multiplied by the number of
months in the time range.

E.g. with `budget = -50000` and *2018-05-13 - 2018-07-05* as the time range, the
total budget displayed will be `2 * -50000 = -100000`.

Note that the budget is specified as one-hundredth of the currency. `budget =
-50000` means a budget of *-500.00 NOK* .

It's also possible to set a per-month budget using the `budgets` key. The value
of `budgets` has to be an array of 12 numbers, one per month. If `budgets` is
unset, the value of `budget` will be used for all months.

Unwanted records may pollute the journal (e.g. inter-account transfers), these
records can be ignored entirely by setting `discard = true` on the matching
group.

### Export file

Most Norwegian banks support export to CSV. This can usually be done through
your bank's web interface.

CSV export example:

```csv
"01.06.2018";"01.06.2018";"Rema 1000";"-1.000,00";"";"";""
"05.06.2018";"05.06.2018";"Rema 1000";"-500,00";"";"";""
"07.06.2018";"07.06.2018";"Atb";"-35,00";"";"";""
"09.06.2018";"09.06.2018";"Rema 1000";"-800,00";"";"";""
"15.06.2018";"15.06.2018";"Atb";"-35,00";"";"";""
"01.07.2018";"01.07.2018";"Rema 1000";"-250,00";"";"";""
"02.07.2018";"02.07.2018";"Atb";"-35,00";"";"";""
"05.07.2018";"05.07.2018";"Rema 1000";"-750,00";"";"";""
"07.07.2018";"07.07.2018";"Atb";"-35,00";"";"";""
"15.07.2018";"15.07.2018";"Atb";"-35,00";"";"";""
```

### Importing records

The command `journal import` is used to import records. Given the export file
and configuration above, records can be imported with:

```
$ journal import 1234.56.78900 example.csv
journal: created 1 new account(s)
journal: imported 10 new record(s) out of 10 total
```

Imported records have now been persisted in a SQLite database located in
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
journal: displaying records between 2018-07-01 and 2018-07-28
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
journal: displaying records between 2018-07-01 and 2018-07-28
+-----------------------+---------------+--------------+------------+------------+-----------+---------+
|         GROUP         |    ACCOUNT    | ACCOUNT NAME |     ID     |    DATE    |   TEXT    | AMOUNT  |
+-----------------------+---------------+--------------+------------+------------+-----------+---------+
| Groceries             | 1234.56.78900 | Example Bank | 77c2a500e1 | 2018-07-05 | Rema 1000 | -750.00 |
| Groceries             | 1234.56.78900 | Example Bank | 8f864212ce | 2018-07-01 | Rema 1000 | -250.00 |
| Public Transportation | 1234.56.78900 | Example Bank | 2e25c40379 | 2018-07-15 | Atb       |  -35.00 |
| Public Transportation | 1234.56.78900 | Example Bank | 84ca136809 | 2018-07-07 | Atb       |  -35.00 |
| Public Transportation | 1234.56.78900 | Example Bank | 5833456f0b | 2018-07-02 | Atb       |  -35.00 |
+-----------------------+---------------+--------------+------------+------------+-----------+---------+
```

If we want show older records, date ranges can be specified using `--since` and
`--until`:

```
$ journal ls --since=2018-06-01 --until=2018-06-30
journal: displaying records between 2018-06-01 and 2018-06-30
+-----------------------+---------+----------+----------+---------+--------------------------------+
|         GROUP         | RECORDS |   SUM    |  BUDGET  | BALANCE |          BALANCE BAR           |
+-----------------------+---------+----------+----------+---------+--------------------------------+
| Groceries             |       3 | -2300.00 | -1000.00 | 1300.00 |                 ++++++++++++++ |
| Public Transportation |       2 |   -70.00 |   -50.00 |   20.00 |                                |
+-----------------------+---------+----------+----------+---------+--------------------------------+
| Total                 |       5 | -2370.00 | -1050.00 | 1320.00 |                 ++++++++++++++ |
+-----------------------+---------+----------+----------+---------+--------------------------------+
```

Note that the slack and budget has been automatically adjusted to the number of
months that contain records.

Options also be combined:
```
$ journal ls --since=2018-01-01 --explain
journal: displaying records between 2018-01-01 and 2018-07-28
+-----------------------+---------------+--------------+------------+------------+-----------+----------+
|         GROUP         |    ACCOUNT    | ACCOUNT NAME |     ID     |    DATE    |   TEXT    |  AMOUNT  |
+-----------------------+---------------+--------------+------------+------------+-----------+----------+
| Groceries             | 1234.56.78900 | Example Bank | e6c18424ba | 2018-06-01 | Rema 1000 | -1000.00 |
| Groceries             | 1234.56.78900 | Example Bank | b6b2496771 | 2018-06-09 | Rema 1000 |  -800.00 |
| Groceries             | 1234.56.78900 | Example Bank | 77c2a500e1 | 2018-07-05 | Rema 1000 |  -750.00 |
| Groceries             | 1234.56.78900 | Example Bank | 2e1aa3cf1a | 2018-06-05 | Rema 1000 |  -500.00 |
| Groceries             | 1234.56.78900 | Example Bank | 8f864212ce | 2018-07-01 | Rema 1000 |  -250.00 |
| Public Transportation | 1234.56.78900 | Example Bank | 2e25c40379 | 2018-07-15 | Atb       |   -35.00 |
| Public Transportation | 1234.56.78900 | Example Bank | 84ca136809 | 2018-07-07 | Atb       |   -35.00 |
| Public Transportation | 1234.56.78900 | Example Bank | 5833456f0b | 2018-07-02 | Atb       |   -35.00 |
| Public Transportation | 1234.56.78900 | Example Bank | 2e8e1ac9e1 | 2018-06-15 | Atb       |   -35.00 |
| Public Transportation | 1234.56.78900 | Example Bank | 84c948c456 | 2018-06-07 | Atb       |   -35.00 |
+-----------------------+---------------+--------------+------------+------------+-----------+----------+
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
