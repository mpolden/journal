package record

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func testReadFrom(lines string, t *testing.T) {
	r := NewReader(strings.NewReader(lines))
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
		{date(2017, 2, 1), "Transaction 1", 133700, 133700},
		{date(2017, 3, 10), "Transaction 2", -4200, 129500},
		{date(2017, 4, 20), "Transaction 3", 4200, 133700},
		{date(2017, 5, 30), "Transaction 4", 4200, 0},
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

func TestReadFrom(t *testing.T) {
	lines := `"01.02.2017";"01.02.2017";"Transaction 1";"1.337,00";"1.337,00";"";""
"10.03.2017";"10.03.2017";"Transaction 2";"-42,00";"1.295,00";"";""
"20.04.2017";"20.04.2017";"Transaction 3";"42,00";"1.337,00";"";""
"30.05.2017";"30.05.2017";"Transaction 4";"42,00";"";"";""
`
	testReadFrom(lines, t)
	testReadFrom(string(byteOrderMark)+lines, t)
}

func TestID(t *testing.T) {
	var tests = []struct {
		r  Record
		id string
	}{
		{Record{
			Account: Account{Number: "1.2.3"},
			Time:    date(2017, 1, 1),
			Text:    "Transaction 1",
			Amount:  42,
		}, "f4fb9cb746"},
		{Record{
			Account: Account{Number: "1.2.4"},
			Time:    date(2017, 1, 1),
			Text:    "Transaction 1",
			Amount:  42,
		}, "3618a31f3c"},
		{Record{
			Account: Account{Number: "1.2.4"},
			Time:    date(2018, 1, 1),
			Text:    "Transaction 1",
			Amount:  42,
		}, "857bb800c9"},
		{Record{
			Account: Account{Number: "1.2.4"},
			Time:    date(2018, 1, 1),
			Text:    "Transaction 2",
			Amount:  42,
		}, "2c07328f92"},
		{Record{
			Account: Account{Number: "1.2.4"},
			Time:    date(2018, 1, 1),
			Text:    "Transaction 2",
			Amount:  42,
			Balance: 1337,
		}, "a56d3a1128"},
	}
	for i, tt := range tests {
		if got := tt.r.ID(); got != tt.id {
			t.Errorf("#%d: want ID = %q, got %q", i, tt.id, got)
		}
	}
}

func TestAssortFunc(t *testing.T) {
	rs := []Record{
		{Time: date(2017, 1, 1), Text: "Foo 1", Amount: 42},
		{Time: date(2017, 1, 1), Text: "Foo 2", Amount: 42},
		{Time: date(2017, 1, 1), Text: "Bar", Amount: 42},
		{Time: date(2017, 1, 1), Text: "Baz", Amount: 42},
	}
	gs := AssortFunc(rs, func(r Record) *Group {
		switch r.Text {
		case "Foo 1", "Foo 2":
			return &Group{Name: "A"}
		case "Bar":
			return &Group{Name: "B"}
		}
		return nil
	})
	var tests = []struct {
		g Group
	}{
		{Group{Name: "A", Records: rs[0:2]}},
		{Group{Name: "B", Records: rs[2:3]}},
	}
	if want, got := len(gs), len(tests); want != got {
		t.Errorf("want len = %d, got %d", want, got)
	}
	for i, tt := range tests {
		if want, got := tt.g.Name, gs[i].Name; want != got {
			t.Errorf("#%d: want Name = %q, got %q", i, want, got)
		}
		if !reflect.DeepEqual(gs[i].Records, tt.g.Records) {
			t.Errorf("#%d: want Records = %+v, got %+v", i, tt.g.Records, gs[i].Records)
		}
	}
}

func TestAssortPeriodFunc(t *testing.T) {
	rs := []Record{
		{Time: date(2017, 1, 10), Text: "Foo", Amount: 42},
		{Time: date(2017, 2, 20), Text: "Bar", Amount: 42},
		{Time: date(2017, 3, 30), Text: "Baz", Amount: 42},
	}
	ps := AssortPeriodFunc(rs,
		func(t time.Time) time.Time {
			return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
		},
		func(r Record) *Group {
			return &Group{Name: "A"}
		})

	var tests = []struct {
		p Period
	}{
		{Period{Time: date(2017, 3, 1), Groups: []Group{{Name: "A", Records: []Record{rs[2]}}}}},
		{Period{Time: date(2017, 2, 1), Groups: []Group{{Name: "A", Records: []Record{rs[1]}}}}},
		{Period{Time: date(2017, 1, 1), Groups: []Group{{Name: "A", Records: []Record{rs[0]}}}}},
	}
	if want, got := len(ps), len(tests); want != got {
		t.Errorf("want len = %d, got %d", want, got)
	}
	for i, tt := range tests {
		if !reflect.DeepEqual(ps[i], tt.p) {
			t.Errorf("#%d: want Period = %+v, got %+v", i, tt.p, ps[i])
		}
	}
}

func TestGroupMath(t *testing.T) {
	var tests = []struct {
		g       Group
		sum     int64
		budget  int64
		balance int64
		tp      TimePeriod
	}{
		{Group{ // 0: Sum and balance
			budget: Budget{Months: [12]int64{500}},
			Records: []Record{
				{Amount: 50},
				{Amount: 200},
				{Amount: 1000},
			},
		},
			1250, // sum
			500,  // budget
			-750, // balance
			TimePeriod{date(2017, 1, 1), date(2017, 2, 1)}},
		{Group{ // 1: Budget is multiplied by months in time range
			budget: Budget{Months: [12]int64{-500, -500, -500}},
			Records: []Record{
				{Time: date(2017, 1, 1), Amount: -500},
				{Time: date(2017, 1, 2), Amount: 100}, // Repeated month does not affect budget
				{Time: date(2017, 3, 1), Amount: -100},
			},
		},
			-500,     // sum
			-500 * 3, // budget
			-1000,    // balance
			TimePeriod{date(2017, 1, 1), date(2017, 3, 1)}},
		{Group{ // 2: Zero balance is considered balanced
			budget: Budget{Months: [12]int64{500}},
			Records: []Record{
				{Time: date(2017, 1, 1), Amount: 250},
				{Time: date(2017, 5, 1), Amount: 250},
			},
		},
			500, // sum
			500, // budget
			0,   // balance
			TimePeriod{date(2017, 1, 1), date(2017, 9, 1)}},
		{Group{ // 3: Defaults to default budget, include month in budget with no records
			budget: Budget{Default: 250},
			Records: []Record{
				{Time: date(2017, 1, 1), Amount: 250},
				{Time: date(2017, 2, 1), Amount: 250},
			},
		},
			500, // sum
			750, // budget
			250, // balance
			TimePeriod{date(2017, 1, 1), date(2017, 3, 1)}},
	}
	for i, tt := range tests {
		if want, got := tt.sum, tt.g.Sum(); want != got {
			t.Errorf("#%d: want Sum = %d, got %d", i, want, got)
		}
		if want, got := tt.budget, tt.g.Budget(tt.tp); want != got {
			t.Errorf("#%d: want Budget = %d, got %d", i, want, got)
		}
		if want, got := tt.balance, tt.g.Balance(tt.tp); want != got {
			t.Errorf("#%d: want Balance = %d, got %d", i, want, got)
		}
	}
}

func TestMaxBalance(t *testing.T) {
	var tests = []struct {
		gs  []Group
		tp  TimePeriod
		max int64
	}{
		{[]Group{
			{Records: []Record{{Amount: -5000}, {Amount: -1000}}},
			{Records: []Record{{Amount: -5000}, {Amount: -3000}}},
			{Records: []Record{{Amount: -5000}, {Amount: -2000}}},
		},
			TimePeriod{date(2017, 1, 1), date(2018, 1, 1)},
			8000},
	}
	for i, tt := range tests {
		if got, want := MaxBalance(tt.gs, tt.tp), tt.max; got != want {
			t.Errorf("#%d: want %d, got %d", i, want, got)
		}
	}
}

func TestMinBalance(t *testing.T) {
	var tests = []struct {
		gs  []Group
		tp  TimePeriod
		min int64
	}{
		{[]Group{
			{Records: []Record{{Amount: 5000}, {Amount: 1000}}},
			{Records: []Record{{Amount: 5000}, {Amount: 3000}}},
			{Records: []Record{{Amount: 5000}, {Amount: 2000}}},
		},
			TimePeriod{date(2017, 1, 1), date(2018, 1, 1)},
			-8000},
	}
	for i, tt := range tests {
		if got, want := MinBalance(tt.gs, tt.tp), tt.min; got != want {
			t.Errorf("#%d: want %d, got %d", i, want, got)
		}
	}
}

func TestSort(t *testing.T) {
	var tests = []struct {
		rs    []Record
		want  []Record
		field Field
	}{
		{
			[]Record{{Text: "B"}, {Text: "A"}},
			[]Record{{Text: "A"}, {Text: "B"}},
			NameField,
		},
		{
			[]Record{{Amount: 1000}, {Amount: 500}},
			[]Record{{Amount: 500}, {Amount: 1000}},
			SumField,
		},
		{
			[]Record{{Time: date(2018, 1, 1)}, {Time: date(2017, 1, 1)}},
			[]Record{{Time: date(2017, 1, 1)}, {Time: date(2018, 1, 1)}},
			TimeField,
		},
	}
	for i, tt := range tests {
		Sort(tt.rs, tt.field)
		if !reflect.DeepEqual(tt.rs, tt.want) {
			t.Errorf("#%d: want %+v, got %+v", i, tt.want, tt.rs)
		}
	}
}

func TestSortGroup(t *testing.T) {
	var tests = []struct {
		gs    []Group
		want  []Group
		field Field
	}{
		{
			[]Group{{Name: "B"}, {Name: "A"}},
			[]Group{{Name: "A"}, {Name: "B"}},
			GroupField,
		},
		{
			[]Group{{Records: []Record{{Amount: 1000}}}, {Records: []Record{{Amount: 500}}}},
			[]Group{{Records: []Record{{Amount: 500}}}, {Records: []Record{{Amount: 1000}}}},
			SumField,
		},
	}
	for i, tt := range tests {
		SortGroup(tt.gs, tt.field)
		if !reflect.DeepEqual(tt.gs, tt.want) {
			t.Errorf("#%d: want %+v, got %+v", i, tt.want, tt.gs)
		}
	}
}
