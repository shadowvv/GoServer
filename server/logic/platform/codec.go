package platform

import (
	"encoding/binary"
	"fmt"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
)

type Codec struct {
}

var _ serviceInterface.CodecInterface = (*Codec)(nil)

func NewCodec() *Codec {
	return &Codec{}
}

func (c *Codec) Marshal(msgID int32, msg proto.Message) ([]byte, error) {
	data, err := proto.Marshal(msg)
	if err != nil {
		logger.Error(fmt.Sprintf("[net] msg marshal error: %v", err))
		return nil, err
	}

	frame := make([]byte, 4+len(data))
	binary.BigEndian.PutUint32(frame[:4], uint32(msgID))
	copy(frame[4:], data)
	return frame, nil
}

func (c *Codec) Unmarshal(data []byte, msg proto.Message) error {
	//msgID 解析已交给上层
	//msgID := int32(data[0])<<24 | int32(data[1])<<16 | int32(data[2])<<8 | int32(data[3])

	if err := proto.Unmarshal(data, msg); err != nil {
		logger.Error(fmt.Sprintf("[net] msg unmarshal error: %v", err))
		return err
	}
	return nil
}
