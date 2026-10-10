package main

import (
	"fmt"
	"os"
	"strconv"
	"por/dax"
)

func main() {
	f, _ := dax.Open(os.Args[1])
	id, _ := strconv.Atoi(os.Args[2])
	b := f.Block(id)
	fmt.Fprintln(os.Stderr, os.Args[1], id, len(b))
	os.Stdout.Write(b)
}
