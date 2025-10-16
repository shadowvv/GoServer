package net

import (
	"google.golang.org/protobuf/proto"
)

// MarshalProto 将 proto.Message 编为 payload（用户侧）
func MarshalProto(m proto.Message) ([]byte, error) {
	return proto.Marshal(m)
}

// UnmarshalProto 将 payload 解为指定 message（用户侧）
func UnmarshalProto(data []byte, m proto.Message) error {
	return proto.Unmarshal(data, m)
}
