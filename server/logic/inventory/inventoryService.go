package inventory

import (
	"errors"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/service/logger"
)

var _ InventoryServiceInterface = (*InventoryService)(nil)

// InventoryService 背包服务：
// 仅作为 PlayerModel.InventoryModel 的业务访问层，不再维护独立缓存/锁。
type InventoryService struct {
	sessionManager logicCommon.SessionManagerInterface
}

func NewInventoryService(sessionManager logicCommon.SessionManagerInterface) *InventoryService {
	return &InventoryService{
		sessionManager: sessionManager,
	}
}

func (s *InventoryService) Init() error {
	logger.InfoWithSprintf("[InventoryService] initialized")
	return nil
}

func (s *InventoryService) getInventoryModel(userId int64) (*model.InventoryModel, error) {
	if s.sessionManager == nil {
		return nil, errors.New("session manager is nil")
	}

	p := s.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return nil, errors.New("player not found")
	}

	player, ok := p.(*model.PlayerModel)
	if !ok || player == nil {
		return nil, errors.New("player type invalid")
	}

	if player.InventoryModel == nil {
		player.InventoryModel = model.CreateInventoryModel(userId)
		player.AppendPlayerModel(player.InventoryModel)
	}

	return player.InventoryModel, nil
}

func (s *InventoryService) AddItem(userId int64, itemId int32, quantity int64) (enum.InventoryResult, error) {
	if quantity <= 0 {
		return enum.INVENTORY_RESULT_INVALID_ITEM, nil
	}

	invModel, err := s.getInventoryModel(userId)
	if err != nil {
		return enum.INVENTORY_RESULT_INVALID_ITEM, err
	}

	return invModel.AddItem(itemId, quantity), nil
}

func (s *InventoryService) RemoveItem(userId int64, itemId int32, quantity int64) (enum.InventoryResult, error) {
	if quantity <= 0 {
		return enum.INVENTORY_RESULT_INVALID_ITEM, nil
	}

	invModel, err := s.getInventoryModel(userId)
	if err != nil {
		return enum.INVENTORY_RESULT_INVALID_ITEM, err
	}

	return invModel.RemoveItem(itemId, quantity), nil
}

func (s *InventoryService) GetItemList(userId int64) ([]*ItemStack, error) {
	invModel, err := s.getInventoryModel(userId)
	if err != nil {
		return nil, err
	}

	stacks := invModel.GetItemList()
	result := make([]*ItemStack, 0, len(stacks))
	for _, stack := range stacks {
		result = append(result, &ItemStack{
			ItemId:  stack.ItemId,
			ItemNum: stack.ItemNum,
		})
	}
	return result, nil
}

func (s *InventoryService) GetItemCount(userId int64, itemId int32) (int64, error) {
	invModel, err := s.getInventoryModel(userId)
	if err != nil {
		return 0, err
	}

	return invModel.GetItemCount(itemId), nil
}

func (s *InventoryService) HasItem(userId int64, itemId int32, quantity int64) (bool, error) {
	invModel, err := s.getInventoryModel(userId)
	if err != nil {
		return false, err
	}

	return invModel.HasItem(itemId, quantity), nil
}

func (s *InventoryService) AddItems(userId int64, items map[int32]int64) (map[int32]enum.InventoryResult, error) {
	invModel, err := s.getInventoryModel(userId)
	if err != nil {
		return nil, err
	}

	results := make(map[int32]enum.InventoryResult, len(items))
	for itemId, quantity := range items {
		if quantity <= 0 {
			continue
		}
		results[itemId] = invModel.AddItem(itemId, quantity)
	}
	return results, nil
}

func (s *InventoryService) RemoveItems(userId int64, items map[int32]int64) (map[int32]enum.InventoryResult, error) {
	invModel, err := s.getInventoryModel(userId)
	if err != nil {
		return nil, err
	}

	results := make(map[int32]enum.InventoryResult, len(items))
	for itemId, quantity := range items {
		if quantity <= 0 {
			continue
		}
		results[itemId] = invModel.RemoveItem(itemId, quantity)
	}
	return results, nil
}
