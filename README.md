# journal

[![Build Status](https://travis-ci.org/mpolden/journal.svg)](https://travis-ci.org/mpolden/journal)

`journal` is a program for recording and displaying financial records.

## Features

* Import records from multiple Norwegian banks, such as Eika Group (including
local banks), Storebrand, Bank Norwegian and Komplett Bank
* Configurable grouping of records to identify spending habits
* Persistent database of imported records

## Example

For the purposes of this example, a bank account exists at *Example Bank* with
the account number *1234.56.78900*, and the bank supports export of records to
CSV.

### Configuration

The first step is to configure our bank accounts and match groups.

`journal` uses the [TOML](https://github.com/toml-lang/toml) configuration
format. By default, the program expects to find the configuration file in
`~/.journalrc`.

Example:

```toml
Database = "/home/user/journal.db"

[[accounts]]
number = "1234.56.78900"
name = "Example Bank"

[[groups]]
name = "Public Transportation"
patterns = ["(?i)^Atb"]

[[groups]]
name = "Groceries"
patterns = ["(?i)^Rema"]

[[groups]]
name = "One-off purchases"
ids = [
  "deadbeef",
  "cafebabe",
]
```

`Database` specifies where the SQLite database containing our records should be
stored.

The `[[accounts]]` section defines known bank accounts. The section can be
repeated to define multiple accounts. Each account we're importing records for
must be defined here first.

The `[[groups]]` section defines how records should be grouped together. `name`
sets the group name and `patterns` sets the list of regular expressions that
must match the corresponding record text.

If any of the patterns in `patterns` match, the group is considered a match.
Matching follows the order declared in the configuration file, where the first
matching group wins.

To avoid having to create patterns for records that may only occur once, it's
possible to pin records to a group using the record ID. Pinning takes precedence
over pattern matching. The record ID can be viewed with `journal ls --explain`.

### Export file

Most Norwegian banks support export to CSV. This can usually be done through
your bank's web interface.

A CSV export typically looks like this:

```csv
"01.05.2018";"01.05.2018";"Rema 1000";"-1.337,00";"3.663,00";"";""
"10.06.2018";"10.06.2018";"Rema 1000";"-42,00";"3.621,00";"";""
"15.07.2018";"15.07.2018";"Atb";"-42,00";"3.579,00";"";""
```

### Importing records

The command `journal import` is used to import records. Given the export file
and configuration above, records can be imported with:

```
$ journal import 1234.56.78900 example.csv
journal: created 1 new account(s)
journal: imported 3 new record(s)
```

Imported records have now been persisted in a SQLite database located in
`/home/user/journal.db`.

Repeating the import only imports records `journal` hasn't seen before, so
running the above command again imports 0 records:

```
$ journal import 1234.56.78900 example.csv
journal: created 0 new account(s)
journal: imported 0 new record(s)
```

Some banks have their own export format, in such cases the correct reader must
be specified when importing records. Example for *Bank Norwegian*:

`$ journal import -r norwegian 1234.56.78900 norwegian-export.xlsx`

See `journal import -h` for complete usage.
 
### Listing records

Now that we have imported records, they can be listed with `journal ls`:

```
$ journal ls
          GROUP         |  SUM   | RECORDS |    FROM    |     TO
+-----------------------+--------+---------+------------+------------+
  Public Transportation | -42,00 |       1 | 2018-07-01 | 2018-07-17
```

By default, only records within the current month are listed and sorted
descending by sum.

Records are grouped together according configured match groups. If we want to
understand a record grouping, we can list individual records and their group:

```
$ journal ls --explain
          GROUP         |    ACCOUNT    | ACCOUNT NAME |     ID     |    DATE    | TEXT | AMOUNT
+-----------------------+---------------+--------------+------------+------------+------+--------+
  Public Transportation | 1234.56.78900 | Example Bank | c18225b0c9 | 2018-07-15 | Atb  | -42,00
```

If we want show older records, date ranges can be specified using `--since` and
`--until`:

```
$ journal ls --since 2018-01-01
          GROUP         |   SUM    | RECORDS |    FROM    |     TO
+-----------------------+----------+---------+------------+------------+
  Groceries             | -1379,00 |       2 | 2018-01-01 | 2018-07-17
  Public Transportation | -42,00   |       1 | 2018-01-01 | 2018-07-17
```

Options also be combined:
```
$ journal ls --since 2018-01-01 --explain
          GROUP         |    ACCOUNT    | ACCOUNT NAME |     ID     |    DATE    |   TEXT    |  AMOUNT
+-----------------------+---------------+--------------+------------+------------+-----------+----------+
  Groceries             | 1234.56.78900 | Example Bank | 51116a3a38 | 2018-05-01 | Rema 1000 | -1337,00
  Groceries             | 1234.56.78900 | Example Bank | eaacbfe8ed | 2018-06-10 | Rema 1000 | -42,00
  Public Transportation | 1234.56.78900 | Example Bank | c18225b0c9 | 2018-07-15 | Atb       | -42,00
```

See `journal ls -h` for complete usage.

## Design

* `cmd` contains the command line interface
* `journal` contains logic for importing and displaying records. This is the
  bridge between `cmd` and `record` / `sql`.
* `record` contains record parsers for various banks
* `sql` contains the persistence layer
