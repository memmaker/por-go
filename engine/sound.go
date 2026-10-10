package engine

import (
	"math"
	"sync"
	"time"
)

// The sound driver, segment 08AE of START.EXE: a four channel sequencer run
// from the timer interrupt (INT 8 reprogrammed to 236.7 Hz). The interrupt
// steps two channel sets every tick: the PC speaker set (blocks at 0x22D,
// pitch table 0x3AD, songs 0x55F) and the Tandy/PCjr set (blocks at 0x3C7,
// pitch table 0x547, songs 0x607) written to the SN76489 at port C0.
// DS:5372 only picks which set PlaySound loads a song into. The segment's
// own data is read from the exe image.
//
// Nothing is audible, and nothing waits, until SoundReader is called, so the
// headless tools stay silent and fast.

const (
	sndRate     = 44100
	sndTimerHz  = 1193182.0 / 0x13B1 // PIT rate programmed by 08AE:137C
	sndChanSize = 0x30
	sndSetA     = 0x22D // PC speaker channel blocks (4 x 0x30)
	sndPitchA   = 0x3AD // 12 period words of the top octave
	sndSongsA   = 0x55F // 8 bytes per song: script pointer of each channel
	sndSetT     = 0x3C7 // Tandy channel blocks (3 tones, then noise)
	sndPitchT   = 0x547
	sndSongsT   = 0x607
	snClock     = 3579545.0 // SN76489 input clock on the Tandy 1000 / PCjr
)

// sound is the driver state. Offsets in comments are within segment 08AE.
type sound struct {
	mu       sync.Mutex
	seg      []byte // copy of the segment: scripts and channel blocks live here
	on       bool   // a sink is attached (SoundReader called)
	isr      bool   // timer interrupt hooked (137C .. 13B1)
	typ      int    // DS:5372: 0 Tandy, 1 PC speaker, 2 none
	saved    int    // DS:5373: typ before Ctrl-S turned the sound off
	wait     int    // DS:5374
	song     int    // 08AE:01FF
	ch       int    // channel being stepped (4 - 08AE:01D9)
	cur      int    // 08AE:0201 channel whose script is running
	base     int    // 08AE:0203
	pitch    int    // 08AE:0205
	ret      int    // 08AE:0207 script return address
	period   int    // timer 2 divisor sent to the speaker
	tone     bool   // speaker gate and data bits both on
	phase    float64
	tickLeft float64
	tandy    bool // Tandy sound chip present (config 'T'): render the SN76489
	sn       sn76489
}

// sn76489 is the Tandy/PCjr sound chip: three square wave tones and a noise
// channel, each with a 4-bit attenuation, programmed through port C0.
type sn76489 struct {
	latch int        // register selected by the last latch byte
	tone  [3]int     // 10-bit dividers
	noise int        // noise control (FB, rate)
	att   [4]int     // 0 loudest .. 15 off
	phase [4]float64 // tone phases; [3] is the noise shift clock
	lfsr  uint16
}

// out = a byte written to port C0.
func (c *sn76489) out(b byte) {
	if b&0x80 != 0 {
		c.latch = int(b>>4) & 7
		ch := c.latch >> 1
		switch {
		case c.latch&1 != 0:
			c.att[ch] = int(b & 0xF)
		case ch == 3:
			c.noise, c.lfsr = int(b&7), 0x4000 // a noise write resets the shift register
		default:
			c.tone[ch] = c.tone[ch]&0x3F0 | int(b&0xF)
		}
		return
	}
	if ch := c.latch >> 1; c.latch&1 == 0 && ch < 3 {
		c.tone[ch] = c.tone[ch]&0xF | int(b&0x3F)<<4
	}
}

// sample renders one output sample.
func (c *sn76489) sample() int {
	v := 0
	for ch := 0; ch < 4; ch++ {
		if c.att[ch] == 15 {
			continue
		}
		amp := int(2000 * math.Pow(10, -float64(c.att[ch])/10)) // 2 dB per step
		n := 0
		if ch < 3 {
			n = c.tone[ch]
		} else if c.noise&3 == 3 {
			n = c.tone[2]
		} else {
			n = 0x10 << uint(c.noise&3)
		}
		if n == 0 {
			n = 0x400
		}
		f := snClock / 32 / float64(n)
		if f >= sndRate/2 {
			continue // beyond Nyquist: the chip's output is a flat level
		}
		c.phase[ch] += f / sndRate
		if ch < 3 {
			c.phase[ch] -= math.Floor(c.phase[ch])
			if c.phase[ch] < 0.5 {
				v += amp
			} else {
				v -= amp
			}
			continue
		}
		for ; c.phase[3] >= 1; c.phase[3]-- {
			// ponytail: 15-bit LFSR with TI taps; the Tandy's NCR 8496 may differ.
			fb := c.lfsr & 1
			if c.noise&4 != 0 {
				fb ^= c.lfsr >> 1 & 1
			}
			c.lfsr = c.lfsr>>1 | fb<<14
		}
		if c.lfsr&1 != 0 {
			v += amp
		} else {
			v -= amp
		}
	}
	return v
}

// snSilence = the four attenuation-off writes (08AE:13D8, 1569).
func (s *sound) snSilence() {
	for _, b := range []byte{0x9F, 0xBF, 0xDF, 0xFF} {
		s.sn.out(b)
	}
}

func (s *sound) w(o int) int       { return int(wordAt(s.seg, o&0xFFFF)) }
func (s *sound) setw(o int, v int) { putWord(s.seg, o&0xFFFF, uint16(v)) }
func (s *sound) b(o int) int       { return int(s.seg[o&0xFFFF]) }

// SoundReader attaches an audio sink and returns its source: signed 16-bit
// little endian stereo at 44100 Hz, to be fed to an audio player.
func (e *Engine) SoundReader() *soundReader {
	s := &e.snd
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seg = make([]byte, 0x10000)
	copy(s.seg, e.X.Mem[0x8AE0:])
	s.sn.att = [4]int{15, 15, 15, 15}
	s.tandy = e.Cfg.Sound == 'T'
	switch e.Cfg.Sound { // main 0x4574
	case 'T':
		s.typ = 0
	case 'P':
		s.typ = 1
	default:
		s.typ = 2
	}
	s.wait = 0x64 // ovr12:045D: soundWait(0x64) at start-up
	s.on = true
	return &soundReader{s}
}

type soundReader struct{ s *sound }

// Read renders the speaker as a square wave (plus the SN76489 when the
// Tandy chip is present), running the sequencer once per timer tick while
// the interrupt is hooked.
func (r *soundReader) Read(p []byte) (int, error) {
	s := r.s
	s.mu.Lock()
	defer s.mu.Unlock()
	n := len(p) / 4 * 4
	for i := 0; i < n; i += 4 {
		v := 0
		if s.isr {
			if s.tickLeft--; s.tickLeft < 0 {
				s.tickLeft += sndRate / sndTimerHz
				s.tick()
			}
			if s.tone {
				per := s.period
				if per == 0 {
					per = 0x10000 // the PIT counts a 0 divisor as 65536
				}
				s.phase += 1193182.0 / float64(per) / sndRate
				s.phase -= math.Floor(s.phase)
				v = 4000
				if s.phase >= 0.5 {
					v = -4000
				}
			}
		}
		if s.tandy {
			v += s.sn.sample()
		}
		p[i], p[i+1], p[i+2], p[i+3] = byte(v), byte(v>>8), byte(v), byte(v>>8)
	}
	return n, nil
}

// tick = 08AE:13EA (timer interrupt): the speaker half, then the Tandy half.
func (s *sound) tick() {
	s.tickSpeaker()
	s.tickTandy()
}

// tickTandy = 08AE:14EC: step the Tandy set, then write each channel's
// divider (word +8 >> 6) and attenuation (top nibble of 0xFFFF - level +0xA)
// to the chip. The noise channel gets only the low nibble as its control
// byte (the 08AE:1720 "cmp di,0x457" special case jumps to its own next
// instruction).
func (s *sound) tickTandy() {
	if s.song == 0 {
		s.snSilence()
		return
	}
	s.pitch, s.base = sndPitchT, sndSetT
	for s.ch = 0; s.ch < 4; s.ch++ {
		if si := sndSetT + s.ch*sndChanSize; s.w(si) != 0 {
			s.step(si)
		}
	}
	if s.b(0x20D) == 0 {
		s.snSilence()
		return
	}
	for ch := 0; ch < 4; ch++ {
		si := sndSetT + ch*sndChanSize
		dl := byte(0x80 + 0x20*ch)
		ax := s.w(si+8) >> 2
		s.sn.out(byte(ax)>>4 ^ dl)
		if dl != 0xE0 {
			s.sn.out(byte(ax >> 8))
		}
		ah := byte((0xFFFF-s.w(si+0xA))>>8) >> 4
		s.sn.out(ah ^ dl + 0x10)
	}
}

// tickSpeaker = 08AE:1477 (PC speaker half): step the four channels, then
// give the first sounding one to the speaker.
func (s *sound) tickSpeaker() {
	s.tone = false
	if s.song == 0 {
		return
	}
	chosen := 0 // 08AE:03C5
	s.pitch, s.base = sndPitchA, sndSetA
	for s.ch = 0; s.ch < 4; s.ch++ {
		si := sndSetA + s.ch*sndChanSize
		if s.w(si) == 0 {
			continue
		}
		s.step(si)
		if chosen == 0 && s.w(si+0xA) != 0 && s.w(si) != 0 {
			chosen = si
		}
	}
	if s.b(0x20D) == 0 || chosen == 0 {
		return
	}
	s.period = s.w(chosen + 8)
	s.tone = s.w(chosen+0xA)&3 == 3 // bit 0 gates timer 2, bit 1 enables the speaker
}

// step = 08AE:157A (si): one tick of a channel: envelope level, pitch slide,
// vibrato, the note timer (next script event when it expires) and the
// envelope table.
func (s *sound) step(si int) {
	s.setw(si+0xA, s.w(si+0xA)+s.w(si+0xC))
	s.setw(si+4, s.w(si+4)+s.w(si+6))
	dx := 0
	if ax := (s.w(si+0x1E) + s.w(si+0x20)) & 0xFFFF; ax != 0 {
		if ax >= s.w(si+0x24) {
			ax -= s.w(si + 0x24)
		}
		s.setw(si+0x1E, ax)
		di := int(int16(ax))>>4 + s.w(si+0x1C)
		hi := int16(s.b(di) << 8)
		dx = int(uint32(int32(hi)*int32(int16(s.w(si+0x22)))) >> 16) // imul, high word
	}
	s.setw(si+8, dx+s.w(si+4))
	if s.w(si+0x14) != 0 {
		s.setw(si+0x14, s.w(si+0x14)-1)
		if s.w(si+0x14) == 0 {
			s.setw(si+0x18, 0x10)
			s.setw(si+0x1A, 1)
		}
	}
	s.setw(si, s.w(si)-1)
	if s.w(si) == 0 {
		s.fetch(si)
	}
	if s.w(si+0x1A) == 0 {
		return
	}
	s.setw(si+0x1A, s.w(si+0x1A)-1)
	if s.w(si+0x1A) != 0 {
		return
	}
	for n := 0; n < 256; n++ { // ponytail: the original loops until a non-0xFFFF entry
		bx := s.w(si+0x16) + s.w(si+0x18)
		s.setw(si+0x18, s.w(si+0x18)+4)
		if s.w(bx+2) != 0xFFFF {
			s.setw(si+0xC, s.w(bx))
			s.setw(si+0x1A, s.w(bx+2))
			return
		}
		ax := s.w(bx)
		s.setw(si+0xA, ax)
		if ax == 0 {
			s.setw(si+0xC, 0)
		}
	}
}

// fetch = 08AE:160B (si): run the channel's script up to its next wait.
// Events are commands 0xFA..0xFF or 3 byte notes (voice<<5, pitch with bit 7
// = last note of the event, duration).
func (s *sound) fetch(si int) {
	di := s.w(si + 2)
	if di == 0 {
		return
	}
	s.cur = si
	reset := func(c int) {
		for _, o := range []int{4, 6, 8, 0xA, 0xC, 0x10, 0x12, 0x16, 0x18, 0x1A, 0x1C, 0x1E, 0x20, 0x22, 0x24} {
			s.setw(c+o, 0)
		}
	}
	// end = 168F: the script yields; a finished channel frees its slot.
	end := func() {
		c := s.cur
		if s.w(c) == 0 {
			s.seg[0x209+s.ch] = 0
			di = 0
		}
		s.setw(c+2, di)
	}
	for guard := 0; guard < 1<<16; guard++ {
		bl := s.b(di)
		di++
		if bl >= 0xFA {
			switch s.w(0x221 + 2*(bl-0xFA)) { // jump table in the segment
			case 0x164E: // reset this channel
				reset(si)
			case 0x16AC: // return
				di = s.ret
			case 0x16B3: // call
				ax := s.w(di)
				s.ret = di + 2
				di = ax
			case 0x1645: // switch to another channel and reset it
				si = s.w(di) + s.base
				di += 2
				reset(si)
			case 0x162E: // loop counter at si+bl, jump while it counts down
				c, ax := s.b(di), s.w(di+1)
				di += 3
				if v := s.w(si + c); v != 0 {
					s.setw(si+c, v-1)
					if v-1 == 0 {
						continue
					}
				}
				di = ax
			case 0x167F: // set the word at si+bl; offset 0 (duration) ends the event
				c, ax := s.b(di), s.w(di+1)
				di += 3
				s.setw(si+c, ax)
				if c == 0 {
					end()
					return
				}
			}
			continue
		}
		for { // 16C4: a note, then chained notes until the last-note bit
			voice := bl >> 5
			vc := voice*s.b(0x1DA) + s.base
			tempo := s.w(si + 0xE)
			dur := s.b(di + 1)
			if tempo&0xFF > 1 {
				dur = (tempo & 0xFF) * dur & 0xFFFF
			}
			s.setw(si, dur)
			p := s.b(di) & 0x7F
			// Quirk: only voice 0 sets a pitch, the others just load the timer.
			if p != 0x7F && voice == 0 {
				s.setw(vc, dur)
				s.setw(vc+0x14, dur)
				ax := (p + s.w(vc+0x12)) & 0xFFFF // 16-bit add
				s.setw(vc+0x18, 0)
				s.setw(vc+0x1A, 1)
				cl := 0
				for ax >= 12 {
					ax -= 12
					cl++
				}
				per := s.w(ax*2+s.pitch) >> uint(cl&31)
				s.setw(vc+4, per)
				s.setw(vc+8, per)
			}
			last := s.b(di)&0x80 != 0
			di += 2
			if last {
				end()
				return
			}
			bl = s.b(di)
			di++
		}
	}
	s.setw(s.cur+2, 0) // runaway script
}

// start = 08AE:1742 (song): load the song's four PC speaker channel scripts.
func (s *sound) start(song int) { s.load(song, sndSetA, sndSongsA) }

// startTandy = 08AE:17AE (song): the same for the Tandy channel set.
func (s *sound) startTandy(song int) { s.load(song, sndSetT, sndSongsT) }

func (s *sound) load(song, set, songs int) {
	s.song = song
	for ch := 0; ch < 4; ch++ {
		p := s.w(songs + song*8 + ch*2)
		if p == 0 {
			continue
		}
		c := set + ch*sndChanSize
		for o := 0; o < sndChanSize; o += 2 {
			s.setw(c+o, 0)
		}
		s.setw(c+2, p)
		s.seg[0x209+ch] = byte(song)
		s.setw(c, 1)
	}
}

// PlaySound = 08AE:00DC (n): 0 stops everything and mutes, 1 unmutes, 0xFF
// stops, anything else starts song n-1. While DS:5374 is non-zero the sound
// is played for 0.1 s and cut; after soundWait(0) it plays on until
// soundWait(n) unhooks the interrupt.
func (e *Engine) PlaySound(n int) {
	s := &e.snd
	if !s.on || s.typ == 2 {
		return
	}
	s.mu.Lock()
	if s.wait > 0 {
		s.isr = true
	}
	switch n {
	case 0:
		s.startTandy(0)
		s.start(0)
		s.seg[0x20D] = 0
	case 1:
		s.seg[0x20D] = 1
	case 0xFF:
		s.startTandy(0)
		s.start(0)
	default:
		if s.typ == 1 {
			s.start(n - 1)
		} else {
			s.startTandy(n - 1)
		}
	}
	if s.wait > 0 {
		s.mu.Unlock()
		time.Sleep(100 * time.Millisecond) // 08AE:006C(0xA): ten hundredths
		s.mu.Lock()
		s.startTandy(0)
		s.start(0)
		s.unhook()
	}
	s.mu.Unlock()
}

// soundWait = 08AE:01A5 (n): 0 hooks the timer interrupt so sounds play
// while the caller animates; n > 0 unhooks it (when it was hooked) and sets
// the cut-after-0.1 s mode used by PlaySound.
func (e *Engine) soundWait(n int) {
	s := &e.snd
	if !s.on || s.typ == 2 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if n == 0 {
		s.isr = true
	} else if s.wait == 0 {
		s.unhook()
	}
	s.wait = n
}

// SoundReset = 08AE:017B: silence everything (used by the sound toggle and
// at start-up). It also copies DS:5372 to DS:5373.
func (e *Engine) SoundReset() {
	s := &e.snd
	s.saved = s.typ
	if !s.on || s.typ == 2 {
		return
	}
	s.mu.Lock()
	s.start(0)
	s.startTandy(0)
	s.unhook()
	s.mu.Unlock()
}

// unhook = 08AE:13B1: restore INT 8, speaker off, all Tandy voices off.
func (s *sound) unhook() {
	s.isr, s.tone = false, false
	s.snSilence()
}

// soundToggle = 0899:006E (Ctrl-S in the key reader): sound on means
// remember DS:5372 in DS:5373, stop the song (PlaySound(DS:2612 = 0xFF))
// and set 5372 to 2, which mutes every driver call; off again restores the
// saved type (the set test [0,1] always holds) and resets the driver.
func (e *Engine) soundToggle() {
	s := &e.snd
	if s.typ != 2 {
		s.saved = s.typ
		e.PlaySound(e.ds2(0x2612)) // 0xFF: stop
		s.mu.Lock()
		s.typ = 2
		s.mu.Unlock()
		return
	}
	if s.saved == 0 || s.saved == 1 {
		s.mu.Lock()
		s.typ = s.saved
		s.mu.Unlock()
		e.SoundReset()
	}
}
