package engine

import (
	"testing"

	"por/exe"
)

// A song started on the Tandy set reaches the SN76489, Ctrl-S mutes it.
func TestTandySound(t *testing.T) {
	x, err := exe.Load("../../START.EXE")
	if err != nil {
		t.Skip(err)
	}
	e := &Engine{X: x, Cfg: Config{Sound: 'T'}}
	r := e.SoundReader()
	e.soundWait(0)
	e.PlaySound(1) // unmute
	e.PlaySound(2)
	buf := make([]byte, 4*sndRate/2)
	r.Read(buf)
	loud := false
	for i := 0; i < len(buf); i += 4 {
		loud = loud || buf[i] != 0 || buf[i+1] != 0
	}
	if !loud {
		t.Fatal("Tandy song is silent")
	}
	e.soundToggle()
	if e.snd.typ != 2 || e.snd.song != 0 {
		t.Fatalf("Ctrl-S: typ %d song %d", e.snd.typ, e.snd.song)
	}
	e.soundToggle()
	if e.snd.typ != 0 {
		t.Fatalf("Ctrl-S again: typ %d", e.snd.typ)
	}
}
