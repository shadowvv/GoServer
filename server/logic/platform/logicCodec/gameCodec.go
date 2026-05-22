package logicCodec

import (
	"fmt"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
)

type GameCodec struct {
}

var _ serviceInterface.CodecInterface = (*GameCodec)(nil)

func NewGameCodec() *GameCodec {
	return &GameCodec{}
}

func (c *GameCodec) Marshal(msgID int32, msg proto.Message) ([]byte, error) {
	data, err := proto.Marshal(msg)
	if err != nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[net] msg marshal error: %v", err))
		return nil, err
	}
	return data, nil
}

func (c *GameCodec) Unmarshal(data []byte, msg proto.Message) error {
	//msgID 解析已交给上层
	//msgID := int32(data[0])<<24 | int32(data[1])<<16 | int32(data[2])<<8 | int32(data[3])

	if err := proto.Unmarshal(data, msg); err != nil {
		logger.ErrorWithZapFields(fmt.Sprintf("[net] msg unmarshal error: %v", err))
		return err
	}
	return nil
}
