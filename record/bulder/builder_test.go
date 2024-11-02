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
2021-11-10;2000,00;;2000,00;min konto;mitt kontonr;annen konto;annet kontonr;Betaling;Gave;;
2021-11-15;;-1000,00;1000,00;annen konto;annet kontonr;min konto;mitt kontonr;Betaling;Butikk 1;Mat og drikke;Dagligvarer
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
		{date(2021, 11, 10), "Betaling,Gave", 200000, 200000},
		{date(2021, 11, 15), "Betaling,Butikk 1,Mat og drikke,Dagligvarer", -100000, 100000},
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

func TestRead2(t *testing.T) {
	// Different format
	in := `Dato;Inn på konto;Ut fra konto;Til konto;Til kontonummer;Fra konto;Fra kontonummer;Type;Tekst;KID;Hovedkategori;Underkategori
2022-10-25;;-1050,00;;til kontonr. 11;fra konto 1;til kontonr. 1;Betaling;Vare 1;min kid;kategori 1;underkategori 1
2022-10-25;;-2500,00;min konto 22;mitt kontonr. 2;fra konto 2;fra kontonr. 2;Betaling;Nedbetaling Lån;;Hus og hjem;Lån
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
		{date(2022, 10, 25), "Betaling,Vare 1,kategori 1,underkategori 1", -105000, 0},
		{date(2022, 10, 25), "Betaling,Nedbetaling Lån,Hus og hjem,Lån", -250000, 0},
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

func TestRead3(t *testing.T) {
	// Yet another format
	in := `Dato;Inn på konto;Ut fra konto;Til konto;Fra konto;Type;Tekst;KID;Hovedkategori;Underkategori
2024-10-07;;-10000.00;;;Betaling;;;Bil;Billån
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
		{date(2024, 10, 07), "Betaling,Bil,Billån", -1000000, 0},
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
