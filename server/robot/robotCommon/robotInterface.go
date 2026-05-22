package robotCommon

import (
	"time"

	"google.golang.org/protobuf/proto"
)

// RobotInterface is used by robot message callbacks.
type RobotInterface interface {
	GetName() string
	Send(msgID uint32, pbMsg proto.Message) error
}

type MessageCallback func(r RobotInterface, msg *MessageStruct) bool

type OperationHandler func(r RobotInterface, op *Operation) bool

type MessageStruct struct {
	MsgID   uint32
	Message proto.Message
	Time    time.Time
}

type Operation struct {
	Type   string                 `yaml:"type"`
	Params map[string]interface{} `yaml:"params"`
}
