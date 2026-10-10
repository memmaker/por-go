package engine

import "testing"

func TestPurse(t *testing.T) {
	if p := coinsToPurse([7]uint16{100, 10, 1, 3, 2, 4, 5}); p != (purse{gold: 15, gems: 4, jewelry: 5}) {
		t.Fatalf("coinsToPurse = %+v", p)
	}
	want := purse{gold: 1234, gems: 2, jewelry: 3}
	if got := readPurse(appendPurse(nil, want)); got != want {
		t.Fatalf("save round trip = %+v", got)
	}
	// a script paying 3 platinum out of 1234 gold
	e := &Engine{purse: want}
	e.setPurseCoin(4, e.purseCoin(4)-3)
	if e.purse != (purse{gold: 1219, gems: 2, jewelry: 3}) {
		t.Fatalf("after paying 3 platinum: %+v", e.purse)
	}
}
