package bulder

import (
	"strings"
	"testing"
	"time"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func TestRead(t *testing.T) {
	in := `Dato;Inn på konto;Ut fra konto;Balanse;Til konto;Til kontonummer;Fra konto;Fra kontonummer;Type;Tekst/KID;Hovedkategori;Underkategori
2021-11-10;2000,00;;2000,00;min konto;mitt kontonr;annen konto;annet kontonr;;Gave;;
2021-11-15;;-1000,00;1000,00;annen konto;annet kontonr;min konto;mitt kontonr;;Butikk 1;;
`
	r := NewReader(strings.NewReader(in))
	rs, err := r.Read()
	if err != nil {
		t.Fatal(err)
	}

	var tests = []struct {
		t       time.Time
		text    string
		amount  int64
		balance int64
	}{
		{date(2021, 11, 10), "Gave", 200000, 200000},
		{date(2021, 11, 15), "Butikk 1", -100000, 100000},
	}
	if len(rs) != len(tests) {
		t.Fatalf("want %d records, got %d", len(tests), len(rs))
	}

	for i, tt := range tests {
		if !rs[i].Time.Equal(tt.t) {
			t.Errorf("#%d: want Time = %s, got %s", i, tt.t, rs[i].Time)
		}
		if rs[i].Text != tt.text {
			t.Errorf("#%d: want Text = %q, got %q", i, tt.text, rs[i].Text)
		}
		if rs[i].Amount != tt.amount {
			t.Errorf("#%d: want Amount = %d, got %d", i, tt.amount, rs[i].Amount)
		}
		if rs[i].Balance != tt.balance {
			t.Errorf("#%d: want Balance = %d, got %d", i, tt.balance, rs[i].Balance)
		}
	}
}
