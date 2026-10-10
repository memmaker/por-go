package engine

import "time"

func nowMicros() int64 { return time.Now().UnixMicro() }
