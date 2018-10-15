package calc

import (
	"fmt"
	"math"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.skia.org/infra/go/query"
	"go.skia.org/infra/go/testutils"
	"go.skia.org/infra/go/vec32"
)

var (
	e        = vec32.MISSING_DATA_SENTINEL
	testRows = Rows{
		",config=8888,os=Ubuntu12,": []float32{e, 1.234, e},
		",config=gpu,os=Ubuntu12,":  []float32{e, 1.236, e},
	}
)

func newTestContext(rows, shortcutRows Rows) *Context {
	if rows == nil {
		rows = testRows
	}
	if shortcutRows == nil {
		shortcutRows = testRows
	}

	from := func(s string) (Rows, error) {
		urlValues, err := url.ParseQuery(s)
		if err != nil {
			return nil, fmt.Errorf("Could not parse query: %s", err)
		}
		q, err := query.New(urlValues)
		ret := Rows{}
		for k, v := range rows {
			if q.Matches(k) {
				ret[k] = v
			}
		}
		return ret, nil
	}

	fromShortcut := func(s string) (Rows, error) {
		return shortcutRows, nil
	}

	return NewContext(from, fromShortcut)
}

func TestFilter(t *testing.T) {
	testutils.SmallTest(t)
	ctx := newTestContext(nil, nil)

	testCases := []struct {
		input  string
		length int
	}{
		{`filter("os=Ubuntu12")`, 2},
		{`filter("")`, 2},
		{`filter("config=8888")`, 1},
		{`filter("config=gpu")`, 1},
		{`filter("config=565")`, 0},
	}
	for _, tc := range testCases {
		rows, err := ctx.Eval(tc.input)
		if err != nil {
			t.Fatalf("Failed to run filter %q: %s", tc.input, err)
		}
		if got, want := len(rows), tc.length; got != want {
			t.Errorf("Wrong rows length %q: Got %v Want %v", tc.input, got, want)
		}
	}
}

func TestShortcut(t *testing.T) {
	testutils.SmallTest(t)
	ctx := newTestContext(nil, Rows{
		",name=t1,": []float32{1.0, -1.0, 2.0, e},
		",name=t2,": []float32{e, 2.0, 8.0, -2.0},
		",name=t3,": []float32{e, 1.0, 8.0, -3.0},
	})

	testCases := []struct {
		input  string
		length int
	}{
		{`shortcut("12")`, 3},
	}
	for _, tc := range testCases {
		rows, err := ctx.Eval(tc.input)
		if err != nil {
			t.Fatalf("Failed to run filter %q: %s", tc.input, err)
		}
		if got, want := len(rows), tc.length; got != want {
			t.Errorf("Wrong rows length %q: Got %v Want %v", tc.input, got, want)
		}
	}
}

func TestEvalNoModifyTile(t *testing.T) {
	testutils.SmallTest(t)
	ctx := newTestContext(nil, nil)

	rows, err := ctx.Eval(`fill(filter("config=8888"))`)
	if err != nil {
		t.Fatalf("Failed to run filter: %s", err)
	}
	assert.Equal(t, 1, len(rows))
	// Make sure we made a deep copy of the rows.
	if got, want := testRows[",config=8888,os=Ubuntu12,"][0], vec32.MISSING_DATA_SENTINEL; got != want {
		t.Errorf("Tile incorrectly modified: Got %v Want %v", got, want)
	}
	if got, want := rows["fill(,config=8888,os=Ubuntu12,)"][0], float32(1.234); got != want {
		t.Errorf("fill() failed: Got %v Want %v", got, want)
	}
}

func TestEvalErrors(t *testing.T) {
	testutils.SmallTest(t)
	ctx := newTestContext(nil, nil)

	testCases := []string{
		// Invalid forms.
		`filter("os=Ubuntu12"`,
		`filter(")`,
		`"config=8888"`,
		`("config=gpu")`,
		`{}`,
		// Forms that fail internal Func validation.
		`filter(2)`,
		`norm("foo")`,
		`norm(filter(""), "foo")`,
		`ave(2)`,
		`avg(2)`,
		`norm(2)`,
		`fill(2)`,
		`norm()`,
		`ave()`,
		`avg()`,
		`fill()`,
	}
	for _, tc := range testCases {
		_, err := ctx.Eval(tc)
		if err == nil {
			t.Fatalf("Expected %q to fail parsing:", tc)
		}
	}
}

func near(a, b float32) bool {
	return math.Abs(float64(a-b)) < 0.001
}

func TestNorm(t *testing.T) {
	testutils.SmallTest(t)
	ctx := newTestContext(Rows{
		",name=t1,": []float32{2.0, -2.0, e},
	}, nil)
	rows, err := ctx.Eval(`norm(filter(""))`)
	if err != nil {
		t.Fatalf("Failed to eval norm() test: %s", err)
	}

	if got, want := rows["norm(,name=t1,)"][0], float32(1.0); !near(got, want) {
		t.Errorf("Distance mismatch: Got %v Want %v Full %#v", got, want, rows)
	}
}

func TestAve(t *testing.T) {
	testutils.SmallTest(t)
	ctx := newTestContext(Rows{
		",name=t1,": []float32{1.0, -1.0, e, e},
		",name=t2,": []float32{e, 2.0, -2.0, e},
	}, nil)
	formula := `ave(filter(""))`
	rows, err := ctx.Eval(formula)
	if err != nil {
		t.Fatalf("Failed to eval ave() test: %s", err)
	}
	if got, want := len(rows), 1; got != want {
		t.Errorf("ave() returned wrong length: Got %v Want %v", got, want)
	}

	for i, want := range []float32{1.0, 0.5, -2.0, e} {
		if got := rows[formula][i]; !near(got, want) {
			t.Errorf("Distance mismatch: Got %v Want %v", got, want)
		}
	}
}

func TestAvg(t *testing.T) {
	testutils.SmallTest(t)
	ctx := newTestContext(Rows{
		",name=t1,": []float32{1.0, -1.0, e, e},
		",name=t2,": []float32{e, 2.0, -2.0, e},
	}, nil)
	formula := `avg(filter(""))`
	rows, err := ctx.Eval(formula)
	if err != nil {
		t.Fatalf("Failed to eval avg() test: %s", err)
	}
	if got, want := len(rows), 1; got != want {
		t.Errorf("avg() returned wrong length: Got %v Want %v", got, want)
	}

	for i, want := range []float32{1.0, 0.5, -2.0, e} {
		if got := rows[formula][i]; !near(got, want) {
			t.Errorf("Distance mismatch: Got %v Want %v", got, want)
		}
	}
}

func TestCount(t *testing.T) {
	testutils.SmallTest(t)
	ctx := newTestContext(Rows{
		",name=t1,": []float32{1.0, -1.0, e, e},
		",name=t2,": []float32{e, 2.0, -2.0, e},
	}, nil)
	formula := `count(filter(""))`
	rows, err := ctx.Eval(formula)
	if err != nil {
		t.Fatalf("Failed to eval count() test: %s", err)
	}
	if got, want := len(rows), 1; got != want {
		t.Errorf("count() returned wrong length: Got %v Want %v", got, want)
	}

	for i, want := range []float32{1.0, 2.0, 1.0, 0.0} {
		if got := rows[formula][i]; !near(got, want) {
			t.Errorf("Distance mismatch: Got %v Want %v", got, want)
		}
	}
}

func TestRatio(t *testing.T) {
	testutils.SmallTest(t)
	ctx := newTestContext(Rows{
		",name=t1,": []float32{10, 4, 100, 50, 9999, 0},
		",name=t2,": []float32{5, 2, 4, 5, 0, 1000},
	}, nil)

	formula := `ratio(ave(fill(filter("name=t1"))),ave(fill(filter("name=t2"))))`
	rows, err := ctx.Eval(formula)
	if err != nil {
		t.Fatalf("Failed to eval ratio() test: %s", err)
	}
	if got, want := len(rows), 1; got != want {
		t.Errorf("ratio() returned wrong length: Got %v Want %v", got, want)
	}
	for i, want := range []float32{2.0, 2.0, 25, 10, e, 0} {
		if got := rows[formula][i]; got != want {
			t.Errorf("Ratio mismatch: Got %v Want %v", got, want)
		}
	}
}

func TestFill(t *testing.T) {
	testutils.SmallTest(t)
	ctx := newTestContext(Rows{
		",name=t1,": []float32{e, e, 2, 3, e, 5},
	}, nil)
	formula := `fill(filter("name=t1"))`
	rows, err := ctx.Eval(formula)
	if err != nil {
		t.Fatalf("Failed to eval fill() test: %s", err)
	}
	if got, want := len(rows), 1; got != want {
		t.Errorf("fill() returned wrong length: Got %v Want %v", got, want)
	}

	for i, want := range []float32{2, 2, 2, 3, 5, 5} {
		if got := rows["fill(,name=t1,)"][i]; !near(got, want) {
			t.Errorf("Distance mismatch: Got %v Want %v", got, want)
		}
	}
}

func TestSum(t *testing.T) {
	testutils.SmallTest(t)
	ctx := newTestContext(Rows{
		",name=t1,": []float32{1.0, -1.0, e, e},
		",name=t2,": []float32{e, 2.0, -2.0, e},
	}, nil)
	formula := `sum(filter(""))`
	rows, err := ctx.Eval(formula)
	if err != nil {
		t.Fatalf("Failed to eval sum() test: %s", err)
	}
	if got, want := len(rows), 1; got != want {
		t.Errorf("sum() returned wrong length: Got %v Want %v", got, want)
	}

	for i, want := range []float32{1.0, 1.0, -2.0, e} {
		if got := rows[formula][i]; !near(got, want) {
			t.Errorf("Distance mismatch: Got %v Want %v", got, want)
		}
	}
}

func TestGeo(t *testing.T) {
	testutils.SmallTest(t)
	ctx := newTestContext(Rows{
		",name=t1,": []float32{1.0, -1.0, 2.0, e},
		",name=t2,": []float32{e, 2.0, 8.0, -2.0},
	}, nil)
	formula := `geo(filter(""))`
	rows, err := ctx.Eval(formula)
	if err != nil {
		t.Fatalf("Failed to eval geo() test: %s", err)
	}
	if got, want := len(rows), 1; got != want {
		t.Errorf("geo() returned wrong length: Got %v Want %v", got, want)
	}

	for i, want := range []float32{1.0, 2.0, 4.0, e} {
		if got := rows[formula][i]; !near(got, want) {
			t.Errorf("Distance mismatch: Got %v Want %v", got, want)
		}
	}
}

func TestLog(t *testing.T) {
	testutils.SmallTest(t)
	ctx := newTestContext(Rows{
		",name=t1,": []float32{1, 10, 100, -1, 0, e},
	}, nil)
	formula := `log(filter(""))`
	rows, err := ctx.Eval(formula)
	if err != nil {
		t.Fatalf("Failed to eval log() test: %s", err)
	}
	if got, want := len(rows), 1; got != want {
		t.Errorf("log() returned wrong length: Got %v Want %v", got, want)
	}

	wanted := []float32{0, 1, 2, e, e, e}
	for i, want := range wanted {
		if got := rows[",name=t1,"][i]; !near(got, want) {
			t.Errorf("Distance mismatch: Got %v Want %v", got, want)
		}
	}
}
