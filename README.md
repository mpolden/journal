# journal

[![Build Status](https://travis-ci.org/mpolden/journal.svg)](https://travis-ci.org/mpolden/journal)

`journal` is a program for recording and displaying financial records.

## Features

* Import records from multiple Norwegian banks, such as Eika Group (including
local banks), Storebrand, Bank Norwegian and Komplett Bank
* Configurable grouping of records to identify spending habits
* Persistent database of imported records

## Examples

My bank account at *Example Bank* has the account number *1234.56.78900* and
supports export of transactions to CSV.

### Configuration

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
Matching follows the order declaring the configuration file, where the first
matching group wins.

### Example export file

Most Norwegian banks support export to CSV. This can accomplished through your
bank's web interface.

A CSV export typically looks like the following:

```csv
"01.05.2018";"01.02.2017";"Rema 1000";"1.337,00";"1.337,00";"";""
"10.06.2018";"10.03.2017";"Rema 1000";"-42,00";"1.295,00";"";""
"15.07.2018";"20.04.2017";"Atb";"42,00";"1.337,00";"";""
```

### Importing records

The command `journal import` is used to import records. Given the export file
and configuration above, records can be imported with:

```
$ journal import 1234.56.78900 example.csv
journal: created 1 new account(s)
journal: imported 3 new record(s)
```

Imported records have now been persisted in a SQLite database located at
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
 
### List records

Now that we have imported records, they can be listed with `journal ls`:

```
$ journal ls
+-----------------------+-------+------------+------------+
|         GROUP         |  SUM  |    FROM    |     TO     |
+-----------------------+-------+------------+------------+
| Public Transportation | 42,00 | 2018-07-01 | 2018-07-16 |
+-----------------------+-------+------------+------------+
```

By default, only records within the current month are listed. Records are
grouped together according configured matching groups. If we want to understand
why a record grouping, we can list individual records and their group:

```
$ journal ls --explain
+---------------+--------------+------------+------+--------+-----------------------+
|    ACCOUNT    | ACCOUNT NAME |    DATE    | TEXT | AMOUNT |         GROUP         |
+---------------+--------------+------------+------+--------+-----------------------+
| 1234.56.78900 | Example Bank | 2018-07-15 | Atb  | 42,00  | Public Transportation |
+---------------+--------------+------------+------+--------+-----------------------+
```

If we want show older records, date ranges can be specified using `--since` and
`--until`:

```
$ journal ls --since 2018-01-01
+-----------------------+---------+------------+------------+
|         GROUP         |   SUM   |    FROM    |     TO     |
+-----------------------+---------+------------+------------+
| Groceries             | 1295,00 | 2018-01-01 | 2018-07-16 |
| Public Transportation | 42,00   | 2018-01-01 | 2018-07-16 |
+-----------------------+---------+------------+------------+
```

Options can of course be combined:
```
$ journal ls --since 2018-01-01 --explain
+---------------+--------------+------------+-----------+---------+-----------------------+
|    ACCOUNT    | ACCOUNT NAME |    DATE    |   TEXT    | AMOUNT  |         GROUP         |
+---------------+--------------+------------+-----------+---------+-----------------------+
| 1234.56.78900 | Example Bank | 2018-06-10 | Rema 1000 | -42,00  | Groceries             |
| 1234.56.78900 | Example Bank | 2018-05-01 | Rema 1000 | 1337,00 | Groceries             |
| 1234.56.78900 | Example Bank | 2018-07-15 | Atb       | 42,00   | Public Transportation |
+---------------+--------------+------------+-----------+---------+-----------------------+
```

See `journal ls -h` for complete usage.

## Design

* `cmd` contains the command line interface
* `journal` contains logic for importing and displaying records. This is the
  bridge between `cmd` and `record` / `sql`.
* `record` contains record parsers for various banks
* `sql` contains the persistence layer
