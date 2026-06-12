package tool

import (
	"crypto/md5"
	"encoding/hex"
	"hash/crc32"
	"hash/fnv"
	"strconv"
)

func Md5(code string) string {
	sum := md5.Sum([]byte(code))      // [16]byte
	return hex.EncodeToString(sum[:]) // 转为 []byte
}

func Hash(s string) int32 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(s))
	return int32(h.Sum32() & 0x7fffffff)
}

func HashIndexByInt64(id int64, shardNum int) int {
	if shardNum <= 0 {
		return -1
	}
	hash := crc32.ChecksumIEEE([]byte(strconv.FormatInt(id, 10)))
	return int(hash % uint32(shardNum))
}
