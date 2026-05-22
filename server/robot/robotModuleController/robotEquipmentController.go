package robotModuleController

import (
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/robot/robotCommon"
	"github.com/drop/GoServer/server/robot/robotRouter"
	"google.golang.org/protobuf/proto"
)

func init() {
	RegisterRobotController("equipment", &RobotEquipmentController{})
}

type RobotEquipmentController struct{}

var _ robotRouter.RobotControllerInterface = (*RobotEquipmentController)(nil)

func (c *RobotEquipmentController) RegisterRobotMessages() {
	RegisterRobotMessageHandler("equipment", pb.MESSAGE_ID_EQUIPMENT_LIST_REQ, &pb.EquipmentListReq{})
	RegisterRobotMessageHandler("equipment", pb.MESSAGE_ID_EQUIPMENT_DETAIL_REQ, &pb.EquipmentDetailReq{})
	RegisterRobotMessageHandler("equipment", pb.MESSAGE_ID_EQUIPMENT_EQUIP_REQ, &pb.EquipmentEquipReq{})
	RegisterRobotMessageHandler("equipment", pb.MESSAGE_ID_EQUIPMENT_UNEQUIP_REQ, &pb.EquipmentUnequipReq{})
	RegisterRobotMessageHandler("equipment", pb.MESSAGE_ID_EQUIPMENT_SWAP_REQ, &pb.EquipmentSwapReq{})
	RegisterRobotMessageHandler("equipment", pb.MESSAGE_ID_EQUIPMENT_QUICK_EQUIP_REQ, &pb.EquipmentQuickEquipReq{})
	RegisterRobotMessageHandler("equipment", pb.MESSAGE_ID_EQUIPMENT_QUICK_UNEQUIP_REQ, &pb.EquipmentQuickUnequipReq{})
	RegisterRobotMessageHandler("equipment", pb.MESSAGE_ID_EQUIPMENT_LOCK_REQ, &pb.EquipmentLockReq{})
	RegisterRobotMessageHandler("equipment", pb.MESSAGE_ID_EQUIPMENT_DECOMPOSE_REQ, &pb.EquipmentDecomposeReq{})

	RegisterRobotReceiveMessageHandler("equipment", pb.MESSAGE_ID_EQUIPMENT_LIST_RESP, &pb.EquipmentListResp{}, onEquipmentListResp)
	RegisterRobotReceiveMessageHandler("equipment", pb.MESSAGE_ID_EQUIPMENT_DETAIL_RESP, &pb.EquipmentDetailResp{}, onEquipmentDetailResp)
	RegisterRobotReceiveMessageHandler("equipment", pb.MESSAGE_ID_EQUIPMENT_EQUIP_RESP, &pb.EquipmentEquipResp{}, onEquipmentEquipResp)
	RegisterRobotReceiveMessageHandler("equipment", pb.MESSAGE_ID_EQUIPMENT_UNEQUIP_RESP, &pb.EquipmentUnequipResp{}, onEquipmentUnequipResp)
	RegisterRobotReceiveMessageHandler("equipment", pb.MESSAGE_ID_EQUIPMENT_SWAP_RESP, &pb.EquipmentSwapResp{}, onEquipmentSwapResp)
	RegisterRobotReceiveMessageHandler("equipment", pb.MESSAGE_ID_EQUIPMENT_QUICK_EQUIP_RESP, &pb.EquipmentQuickEquipResp{}, onEquipmentQuickEquipResp)
	RegisterRobotReceiveMessageHandler("equipment", pb.MESSAGE_ID_EQUIPMENT_QUICK_UNEQUIP_RESP, &pb.EquipmentQuickUnequipResp{}, onEquipmentQuickUnequipResp)
	RegisterRobotReceiveMessageHandler("equipment", pb.MESSAGE_ID_EQUIPMENT_LOCK_RESP, &pb.EquipmentLockResp{}, onEquipmentLockResp)
	RegisterRobotReceiveMessageHandler("equipment", pb.MESSAGE_ID_EQUIPMENT_DECOMPOSE_RESP, &pb.EquipmentDecomposeResp{}, onEquipmentDecomposeResp)
	RegisterRobotReceiveMessageHandler("equipment", pb.MESSAGE_ID_EQUIPMENT_FORGE_RESP, &pb.EquipmentForgeResp{}, onEquipmentForgeResp)
	RegisterRobotReceiveMessageHandler("equipment", pb.MESSAGE_ID_EQUIPMENT_STRONG_RESP, &pb.EquipmentStrongResp{}, onEquipmentStrongResp)
	RegisterRobotReceiveMessageHandler("equipment", pb.MESSAGE_ID_EQUIPMENT_REBIRTH_RESP, &pb.EquipmentRebirthResp{}, onEquipmentRebirthResp)
}

func (c *RobotEquipmentController) GetManualProto(msgID pb.MESSAGE_ID) proto.Message {
	return nil
}

func onEquipmentListResp(message proto.Message, robot robotCommon.RobotInterface) bool {
	_, ok := message.(*pb.EquipmentListResp)
	_ = robot
	return ok
}

func onEquipmentDetailResp(message proto.Message, robot robotCommon.RobotInterface) bool {
	_, ok := message.(*pb.EquipmentDetailResp)
	_ = robot
	return ok
}

func onEquipmentEquipResp(message proto.Message, robot robotCommon.RobotInterface) bool {
	_, ok := message.(*pb.EquipmentEquipResp)
	_ = robot
	return ok
}

func onEquipmentUnequipResp(message proto.Message, robot robotCommon.RobotInterface) bool {
	_, ok := message.(*pb.EquipmentUnequipResp)
	_ = robot
	return ok
}

func onEquipmentSwapResp(message proto.Message, robot robotCommon.RobotInterface) bool {
	_, ok := message.(*pb.EquipmentSwapResp)
	_ = robot
	return ok
}

func onEquipmentQuickEquipResp(message proto.Message, robot robotCommon.RobotInterface) bool {
	_, ok := message.(*pb.EquipmentQuickEquipResp)
	_ = robot
	return ok
}

func onEquipmentQuickUnequipResp(message proto.Message, robot robotCommon.RobotInterface) bool {
	_, ok := message.(*pb.EquipmentQuickUnequipResp)
	_ = robot
	return ok
}

func onEquipmentLockResp(message proto.Message, robot robotCommon.RobotInterface) bool {
	_, ok := message.(*pb.EquipmentLockResp)
	_ = robot
	return ok
}

func onEquipmentDecomposeResp(message proto.Message, robot robotCommon.RobotInterface) bool {
	_, ok := message.(*pb.EquipmentDecomposeResp)
	_ = robot
	return ok
}

func onEquipmentForgeResp(message proto.Message, robot robotCommon.RobotInterface) bool {
	_, ok := message.(*pb.EquipmentForgeResp)
	_ = robot
	return ok
}

func onEquipmentStrongResp(message proto.Message, robot robotCommon.RobotInterface) bool {
	_, ok := message.(*pb.EquipmentStrongResp)
	_ = robot
	return ok
}

func onEquipmentRebirthResp(message proto.Message, robot robotCommon.RobotInterface) bool {
	_, ok := message.(*pb.EquipmentRebirthResp)
	_ = robot
	return ok
}
