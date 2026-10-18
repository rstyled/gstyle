package errorflow

import "testing"

func TestUnknownUseConjunction(t *testing.T) {
	used := Result{Used: true, Known: true}
	ignored := Result{Known: true}
	unknown := Result{}
	cases := []struct{ left, right, want Result }{{used, used, used}, {used, ignored, ignored}, {ignored, used, ignored}, {unknown, used, unknown}, {used, unknown, unknown}, {unknown, ignored, ignored}, {ignored, unknown, ignored}, {unknown, unknown, unknown}}
	for _, tc := range cases {
		if got := tc.left.All(tc.right); got != tc.want {
			t.Fatalf("%+v and %+v = %+v, want %+v", tc.left, tc.right, got, tc.want)
		}
	}
}

func TestMissingQueryIsUnknown(t *testing.T) {
	var query *Query
	if query.Value(1) != (Result{}) || query.Assignment(1, 1) != (Result{}) || query.Reachable(1) {
		t.Fatal("missing analysis becomes definite use/discard evidence")
	}
	if query, err := New(nil); query != nil || err != nil {
		t.Fatalf("missing graph: query=%v error=%v", query, err)
	}
}
