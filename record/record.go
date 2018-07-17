package record

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	decimalSeparator  = "."
	thousandSeparator = ","
	byteOrderMark     = '\uFEFF'
)

// Reader is the interface for record readers.
type Reader interface {
	Read() ([]Record, error)
}

// Account identifies a finanical account.
type Account struct {
	Number string
	Name   string
}

// Record contains details of a finanical record.
type Record struct {
	Account Account
	Time    time.Time
	Text    string
	Amount  int64
}

type defaultReader struct {
	rd       io.Reader
	replacer *strings.Replacer
}

func NewReader(rd io.Reader) Reader {
	return &defaultReader{
		rd:       rd,
		replacer: strings.NewReplacer(decimalSeparator, "", thousandSeparator, ""),
	}
}

// ID returns a shortened SHA-1 hash of the fields in this record
func (r *Record) ID() string {
	var buf bytes.Buffer
	buf.WriteString(r.Account.Number)
	buf.WriteString(r.Time.Format("2006-01-02"))
	buf.WriteString(r.Text)
	buf.WriteString(strconv.FormatInt(r.Amount, 10))
	sum := sha1.Sum(buf.Bytes())
	return fmt.Sprintf("%x", sum)[:10]
}

func (d *defaultReader) parseAmount(s string) (int64, error) {
	v := d.replacer.Replace(s)
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, err
	}
	return n, nil
}

func (r *defaultReader) Read() ([]Record, error) {
	buf := bufio.NewReader(r.rd)
	// Peek at the first rune see if the file starts with a byte order mark
	rune, _, err := buf.ReadRune()
	if err != nil {
		return nil, err
	}
	if rune != byteOrderMark {
		if err := buf.UnreadRune(); err != nil {
			return nil, err
		}
	}
	c := csv.NewReader(buf)
	c.Comma = ';'
	var rs []Record
	line := 0
	for {
		record, err := c.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		line++
		if len(record) < 4 {
			continue
		}
		t, err := time.Parse("02.01.2006", record[0])
		if err != nil {
			return nil, errors.Wrapf(err, "invalid time found on line %d: %q", line, record[0])
		}
		text := record[2]
		amount, err := r.parseAmount(record[3])
		if err != nil {
			return nil, errors.Wrapf(err, "invalid amount found on line %d: %q", line, record[3])
		}
		rs = append(rs, Record{Time: t, Text: text, Amount: amount})
	}
	return rs, nil
}
