package tool

import "time"

func GetCurrentTimeMillis() int64 {
	return time.Now().UnixNano() / 1000000
}
