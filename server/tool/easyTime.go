package tool

import "time"

func GetCurrentTimeMillis() int64 {
	return int64(time.Now().UnixNano() / 1000000)
}
