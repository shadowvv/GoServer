package logicCodec

import (
	"encoding/binary"
	"fmt"
	"github.com/drop/GoServer/server/logic/rpcPb"
	"github.com/drop/GoServer/server/service/serviceInterface"
	"google.golang.org/protobuf/proto"
)

type GatewayCodec struct {
}

var _ serviceInterface.CodecInterface = (*GatewayCodec)(nil)

func NewGatewayCodec() *GatewayCodec {
	return &GatewayCodec{}
}

func (c *GatewayCodec) Marshal(msgID int32, msg proto.Message) ([]byte, error) {
	clientMessage, ok := msg.(*rpcPb.BackwardClientMessage)
	if !ok {
		return nil, fmt.Errorf("[net] msg type error")
	}
	frame := make([]byte, 4+len(clientMessage.Payload))
	binary.BigEndian.PutUint32(frame[:4], uint32(msgID))
	copy(frame[4:], clientMessage.Payload)
	return frame, nil
}

func (c *GatewayCodec) Unmarshal(data []byte, msg proto.Message) error {
	clientReq, ok := msg.(*rpcPb.ClientReq)
	if !ok {
		return fmt.Errorf("[net] msg type error")
	}
	clientReq.Data = data
	return nil
}
