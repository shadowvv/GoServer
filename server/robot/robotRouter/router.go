package robotRouter

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/robot/robotCommon"
	"github.com/drop/GoServer/server/robot/robotUtils"
	"google.golang.org/protobuf/proto"
)

type RobotControllerInterface interface {
	RegisterRobotMessages()
	GetManualProto(msgID pb.MESSAGE_ID) proto.Message
}

type RobotReceiveMessageHandler func(message proto.Message, robot robotCommon.RobotInterface) bool

type robotMessageBinding struct {
	module    string
	prototype proto.Message
}

var (
	robotControllers             = make(map[string]RobotControllerInterface)
	robotMessageBindings         = make(map[pb.MESSAGE_ID]robotMessageBinding)
	robotReceiveMessageBindings  = make(map[pb.MESSAGE_ID]robotMessageBinding)
	robotMessageCallbacks        = make(map[pb.MESSAGE_ID]robotCommon.MessageCallback)
	registerRobotMessagesOnce    sync.Once
	registerRobotMessagesInitErr error
)

func RegisterRobotController(module string, controller RobotControllerInterface) {
	key := robotUtils.NormalizeString(module)
	if key == "" {
		panic("robot controller module is empty")
	}
	if controller == nil {
		panic(fmt.Sprintf("robot controller is nil for module %q", key))
	}
	if _, exists := robotControllers[key]; exists {
		panic(fmt.Sprintf("robot controller duplicated for module %q", key))
	}
	robotControllers[key] = controller
}

func RegisterAllRobotMessages() error {
	registerRobotMessagesOnce.Do(func() {
		if len(robotControllers) == 0 {
			registerRobotMessagesInitErr = fmt.Errorf("no robot controllers registered")
			return
		}
		for module, controller := range robotControllers {
			if controller == nil {
				registerRobotMessagesInitErr = fmt.Errorf("robot controller for module %q is nil", module)
				return
			}
			controller.RegisterRobotMessages()
		}
	})
	return registerRobotMessagesInitErr
}

func GetManualProtoByMessageID(messageID pb.MESSAGE_ID) proto.Message {
	binding, ok := robotMessageBindings[messageID]
	if !ok {
		return nil
	}

	controller, ok := robotControllers[binding.module]
	if !ok || controller == nil {
		return nil
	}

	return controller.GetManualProto(messageID)
}

func RegisterRobotReceiveMessageHandler(module string, messageID pb.MESSAGE_ID, response proto.Message, handler RobotReceiveMessageHandler) {
	moduleKey := robotUtils.NormalizeString(module)
	if moduleKey == "" {
		panic(fmt.Sprintf("robot receive message module is empty for messageId=%d", messageID))
	}
	if response == nil {
		panic(fmt.Sprintf("robot receive message prototype is nil for messageId=%d", messageID))
	}
	if handler == nil {
		panic(fmt.Sprintf("robot receive message handler is nil for messageId=%d", messageID))
	}
	if old, exists := robotReceiveMessageBindings[messageID]; exists {
		panic(fmt.Sprintf("robot receive message duplicated: messageId=%d oldModule=%q newModule=%q", messageID, old.module, moduleKey))
	}

	robotReceiveMessageBindings[messageID] = robotMessageBinding{
		module:    moduleKey,
		prototype: response,
	}
	robotMessageCallbacks[messageID] = func(r robotCommon.RobotInterface, msg *robotCommon.MessageStruct) bool {
		if msg == nil || msg.Message == nil {
			return false
		}
		return handler(msg.Message, r)
	}
}

func GetMessageCallback(msgID uint32) robotCommon.MessageCallback {
	return robotMessageCallbacks[pb.MESSAGE_ID(msgID)]
}

func BuildReceiveProtoMessageByMessageID(messageID pb.MESSAGE_ID) (proto.Message, error) {
	binding, ok := robotReceiveMessageBindings[messageID]
	if !ok {
		return nil, fmt.Errorf("messageId %d has no receive protobuf binding", messageID)
	}
	return cloneMessageByPrototype(messageID, binding.prototype)
}

func ValidateRobotOperationBinding(module string, messageID pb.MESSAGE_ID) error {
	moduleKey := robotUtils.NormalizeString(module)
	binding, ok := robotMessageBindings[messageID]
	if !ok {
		return fmt.Errorf("messageId %d in module %q is not registered by robot controllers", messageID, moduleKey)
	}
	if moduleKey != binding.module {
		return fmt.Errorf("messageId %d belongs to module %q, but configured in module %q", messageID, binding.module, moduleKey)
	}
	return nil
}

func BuildProtoMessageByMessageID(messageID pb.MESSAGE_ID) (proto.Message, error) {
	binding, ok := robotMessageBindings[messageID]
	if !ok {
		return nil, fmt.Errorf("messageId %d has no protobuf binding", messageID)
	}
	return cloneMessageByPrototype(messageID, binding.prototype)
}

func cloneMessageByPrototype(messageID pb.MESSAGE_ID, prototype proto.Message) (proto.Message, error) {
	typ := reflect.TypeOf(prototype)
	if typ == nil || typ.Kind() != reflect.Ptr || typ.Elem().Kind() != reflect.Struct {
		return nil, fmt.Errorf("invalid protobuf prototype for messageId %d", messageID)
	}

	msg, ok := reflect.New(typ.Elem()).Interface().(proto.Message)
	if !ok {
		return nil, fmt.Errorf("failed to create protobuf instance for messageId %d", messageID)
	}
	return msg, nil
}

func RegisterRobotMessageHandler(module string, messageID pb.MESSAGE_ID, request proto.Message) {
	moduleKey := robotUtils.NormalizeString(module)
	if moduleKey == "" {
		panic(fmt.Sprintf("robot message module is empty for messageId=%d", messageID))
	}
	if request == nil {
		panic(fmt.Sprintf("robot message prototype is nil for messageId=%d", messageID))
	}
	if old, exists := robotMessageBindings[messageID]; exists {
		panic(fmt.Sprintf("robot message duplicated: messageId=%d oldModule=%q newModule=%q", messageID, old.module, moduleKey))
	}
	robotMessageBindings[messageID] = robotMessageBinding{
		module:    moduleKey,
		prototype: request,
	}
}

func IsRobotMessageRegistered(messageID pb.MESSAGE_ID) bool {
	_, ok := robotMessageBindings[messageID]
	return ok
}
