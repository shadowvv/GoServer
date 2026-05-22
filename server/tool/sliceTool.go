package tool

import "math/rand"

func Shuffle(ids []int32) []int32 {
	temp := make([]int32, len(ids))
	copy(temp, ids)
	for i := len(temp) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		temp[i], temp[j] = temp[j], temp[i]
	}
	return temp
}
