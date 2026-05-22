package robotRouter

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/robot/robotCommon"
	"google.golang.org/protobuf/proto"
)

// EncodeMessage 编码消息：4字节消息ID + protobuf数据
func EncodeMessage(msgID uint32, pbMsg proto.Message) ([]byte, error) {
	if pbMsg == nil {
		return nil, fmt.Errorf("protobuf message is nil")
	}

	data, err := proto.Marshal(pbMsg)
	if err != nil {
		return nil, err
	}

	frame := make([]byte, 4+len(data))
	binary.BigEndian.PutUint32(frame[:4], msgID)
	copy(frame[4:], data)
	return frame, nil
}

// DecodeMessage 解码消息。
//
// 当前优先按 MsgID 取消息原型并反序列化；
// 如果 MsgID 未注册，则尝试 MsgID-1（兼容当前请求/响应 +1 的匹配逻辑）。
func DecodeMessage(data []byte) (*robotCommon.MessageStruct, error) {
	if len(data) < 4 {
		return nil, nil
	}

	msgID := binary.BigEndian.Uint32(data[:4])
	payloadBytes := data[4:]

	message, err := decodePayload(pb.MESSAGE_ID(msgID), payloadBytes)
	if err != nil {
		return nil, err
	}

	return &robotCommon.MessageStruct{
		MsgID:   msgID,
		Message: message,
		Time:    time.Now(),
	}, nil
}

func decodePayload(msgID pb.MESSAGE_ID, payloadBytes []byte) (proto.Message, error) {
	var unmarshalErr error
	msg, err := BuildReceiveProtoMessageByMessageID(msgID)
	if err != nil {
		msg, err = BuildProtoMessageByMessageID(msgID)
	}
	if err != nil {
		return nil, unmarshalErr
	}

	if err = proto.Unmarshal(payloadBytes, msg); err != nil {
		unmarshalErr = fmt.Errorf("unmarshal payload failed for msgID=%d: %w", msgID, err)
		return nil, unmarshalErr
	}
	return msg, nil
}
