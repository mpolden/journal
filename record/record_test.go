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
		t      time.Time
		text   string
		amount int64
	}{
		{date(2017, 2, 1), "Transaction 1", 133700},
		{date(2017, 3, 10), "Transaction 2", -4200},
		{date(2017, 4, 20), "Transaction 3", 4200},
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
	}
}

func TestReadFrom(t *testing.T) {
	lines := `"01.02.2017";"01.02.2017";"Transaction 1";"1.337,00";"1.337,00";"";""
"10.03.2017";"10.03.2017";"Transaction 2";"-42,00";"1.295,00";"";""
"20.04.2017";"20.04.2017";"Transaction 3";"42,00";"1.337,00";"";""
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
		g        Group
		sum      int64
		budget   int64
		balance  int64
		balanced bool
	}{
		{Group{ // 0: Sum and balance
			MonthlyBudget: 500,
			Records: []Record{
				{Amount: 50},
				{Amount: 200},
				{Amount: 1000},
			},
		},
			1250,   // sum
			500,    // budget
			-750,   // balance
			false}, // balanced
		{Group{ // 1: Budget is multiplied by months in time range
			MonthlyBudget: -500,
			Records: []Record{
				{Time: date(2017, 1, 1), Amount: -500},
				{Time: date(2017, 3, 1), Amount: -100},
			},
		},
			-600,     // sum
			-500 * 2, // budget
			-400,     // balance
			false},   // balanced
		{Group{ // 2: Zero balance is considered balanced
			MonthlyBudget: 500,
			Records: []Record{
				{Amount: 250},
				{Amount: 250},
			},
		},
			500,   // sum
			500,   // budget
			0,     // balance
			true}, // balanced
		{Group{ // 3: Positive balance is balanced by positive slack
			MonthlyBudget: -500,
			MonthlySlack:  50,
			Records: []Record{
				{Amount: -500},
				{Amount: -50},
			},
		},
			-550,  // sum
			-500,  // budget
			50,    // balance
			true}, // balanced
		{Group{ // 4: Positive balance is not balanced by negative slack
			MonthlyBudget: -500,
			MonthlySlack:  -50,
			Records: []Record{
				{Amount: -500},
				{Amount: -50},
			},
		},
			-550,   // sum
			-500,   // budget
			50,     // balance
			false}, // balanced
		{Group{ // 5: Negative balance is balanced by exceeding negative slack
			MonthlyBudget: 500,
			MonthlySlack:  -60,
			Records: []Record{
				{Amount: 500},
				{Amount: 50},
			},
		},
			550,   // sum
			500,   // budget
			-50,   // balance
			true}, // balanced
		{Group{ // 6: Negative balance is not balanced by positive slack
			MonthlyBudget: 500,
			MonthlySlack:  50,
			Records: []Record{
				{Amount: 500},
				{Amount: 50},
			},
		},
			550,    // sum
			500,    // budget
			-50,    // balance
			false}, // balanced
		{Group{ // 7: Slack does not affect zero balance
			MonthlyBudget: -560,
			MonthlySlack:  -50,
			Records: []Record{
				{Amount: -500},
				{Amount: -60},
			},
		},
			-560,  // sum
			-560,  // budget
			0,     // balance
			true}, // balanced

	}
	for i, tt := range tests {
		if want, got := tt.sum, tt.g.Sum(); want != got {
			t.Errorf("#%d: want Sum = %d, got %d", i, want, got)
		}
		if want, got := tt.budget, tt.g.Budget(); want != got {
			t.Errorf("#%d: want Budget = %d, got %d", i, want, got)
		}
		if want, got := tt.balance, tt.g.Balance(); want != got {
			t.Errorf("#%d: want Balance = %d, got %d", i, want, got)
		}
		if want, got := tt.balanced, tt.g.IsBalanced(); want != got {
			t.Errorf("#%d: want Balanced = %t, got %t", i, want, got)
		}
	}
}
