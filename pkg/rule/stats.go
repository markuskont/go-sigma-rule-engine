package rule

import "time"

type Stats struct {
	Hits, Total int64
	Took
}

type Took struct {
	Min, Max, Avg time.Duration
	ringBuffer    chan time.Duration
}
