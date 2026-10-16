package engine

import "testing"

func TestPurse(t *testing.T) {
	if p := coinsToPurse([7]uint16{100, 10, 1, 3, 2, 4, 5}); p != (purse{gold: 14, copper: 100, gems: 4, jewelry: 5}) {
		t.Fatalf("coinsToPurse = %+v", p)
	}
	want := purse{gold: 1234, copper: 7, gems: 2, jewelry: 3}
	if got := readPurse(appendPurse(nil, want)); got != want {
		t.Fatalf("save round trip = %+v", got)
	}
	// a script paying 3 platinum out of 1234 gold
	e := &Engine{purse: want}
	e.setPurseCoin(4, e.purseCoin(4)-3)
	if e.purse != (purse{gold: 1219, copper: 7, gems: 2, jewelry: 3}) {
		t.Fatalf("after paying 3 platinum: %+v", e.purse)
	}
	// 150 copper twice is a gold piece and 100 copper, not rounded per pile
	var p purse
	p.add(coinsToPurse([7]uint16{150}))
	p.add(coinsToPurse([7]uint16{150}))
	if p != (purse{gold: 1, copper: 100}) {
		t.Fatalf("two piles = %+v", p)
	}
	if got := readPurse(appendPurse(nil, want)[:12]); got != (purse{gold: 1234, gems: 2, jewelry: 3}) {
		t.Fatalf("old save = %+v", got)
	}
	// the tavern game sees gold; winning 100 out of 60000
	e = &Engine{purse: purse{gold: 60000}}
	e.run.gambling = true
	e.setPurseCoin(4, e.purseCoin(4)+100)
	if e.purse.gold != 60100 {
		t.Fatalf("after winning 100 gold: %+v", e.purse)
	}
}
