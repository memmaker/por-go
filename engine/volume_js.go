//go:build js

package engine

import "syscall/js"

// volume is the goplay desk's fader for kind ("music" or "sfx"): 0 to 1,
// 1 without the desk.
func volume(kind string) float64 {
	v := js.Global().Get("goplayVolume")
	if v.Type() != js.TypeObject || v.Get(kind).Type() != js.TypeNumber {
		return 1
	}
	return v.Get(kind).Float()
}
