package petRecruit

import (
	"context"
	"errors"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/operationLogService"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/eventService"
	"github.com/drop/GoServer/server/logic/platform/platformLogger"
	unlockSvc "github.com/drop/GoServer/server/logic/unlockService"
	"github.com/drop/GoServer/server/logic/vipCard"
	"github.com/drop/GoServer/server/tool"
)

const (
	// 面板固定 3 个槽位：所有候选列表/状态数组都按 3 个元素读写（与客户端协议一致）。
	recruitSlotCount            = 3
	defaultSanctuaryLevel int32 = 1

	// 招募支付方式（与 pb.PetRecruitReq.recruitType 一致）。
	recruitTypeCoupon  int32 = 1
	recruitTypeDiamond int32 = 2

	// 槽位状态（与 pb.PetRecruitSlot.state 一致）。
	slotStateRecruitable int32 = 0
	slotStateRecruited   int32 = 1

	// 通用开关标记：用于 DB 字段（0/1）与协议字段对齐，避免布尔落库带来的兼容问题。
	featureDisabled int32 = 0
	featureEnabled  int32 = 1

	// 三倍招募：只影响“本次发放数量/本次消耗”，不影响折扣次数计数语义。
	tripleRecruitMultiplier int32 = 3
	// 单次招募的“最小单位”：券消耗 1 张、宠物卡发放 1 张。
	singleRecruitCost int32 = 1
	// 招募券 itemId：业务约定为固定值（展示与扣费必须一致）。
	petRecruitCouponItemID int32 = 1110049
)

var eventBusService *eventService.EventBus

func SetEventBusService(service *eventService.EventBus) {
	eventBusService = service
}

func init() {
	model.SetPetRecruitPrivilegeChecker(hasPrivilege)
	model.SetPetRecruitInfoBuilder(func(player *model.PlayerModel, m *model.PetRecruitModel) *pb.PetRecruitInfo {
		// 通过 model 包提供“组包回调”，避免 model ↔ service 的直接互相 import（打破循环依赖）。
		// Heartbeat/Push 场景只依赖 model 内部持有的函数指针，service 负责注入“如何计算展示给客户端的下一次报价”。
		info := BuildInfo(m)
		fillNextRecruitOffer(m, info)
		return info
	})
}

func GetOrLoadModel(player *model.PlayerModel) (*model.PetRecruitModel, error) {
	if player == nil || player.User == nil {
		return nil, errors.New("player is nil")
	}
	if player.PetRecruitModel != nil && player.PetRecruitModel.Entity != nil {
		player.PetRecruitModel.Player = player
		return player.PetRecruitModel, nil
	}
	m, err := model.LoadOrCreatePetRecruitModel(player)
	if err != nil {
		return nil, err
	}
	player.PetRecruitModel = m
	player.PlayerModels = append(player.PlayerModels, m)
	return m, nil
}

func GetSanctuaryLevel(player *model.PlayerModel) int32 {
	if player != nil && player.ArchitectureModel != nil {
		if ent := player.ArchitectureModel.Entities[int32(enum.ARCHITECTURE_TYPE_PET)]; ent != nil {
			if ent.Level > 0 {
				return ent.Level
			}
		}
	}
	return defaultSanctuaryLevel
}

func hasPrivilege(player *model.PlayerModel) bool {
	v, err := vipCard.Service.GetFunctionValue(player, enum.VIP_PRIVILEGE_RECRUITMENT)
	return err == nil && v > 0
}

// RollCandidates 按 dropGroup “独立 roll 3 次”，分别作为 3 个 slot 的 petId。
// 若配置允许重复，则允许同一个 petId 出现在多个 slot。
func RollCandidates(player *model.PlayerModel, sanctuaryLevel int32) (tool.JSONInt32Slice, error) {
	return model.RollPetRecruitCandidates(sanctuaryLevel, hasPrivilege(player))
}

func BuildInfo(m *model.PetRecruitModel) *pb.PetRecruitInfo {
	if m == nil || m.Entity == nil {
		return &pb.PetRecruitInfo{}
	}
	slots := make([]*pb.PetRecruitSlot, 0, recruitSlotCount)
	for i := 0; i < recruitSlotCount; i++ {
		petId := int32(0)
		state := slotStateRecruited
		if len(m.Entity.CandidatePetIDs) == recruitSlotCount {
			petId = m.Entity.CandidatePetIDs[i]
		}
		if len(m.Entity.SlotStates) == recruitSlotCount {
			state = m.Entity.SlotStates[i]
		}
		slots = append(slots, &pb.PetRecruitSlot{Index: int32(i + 1), PetId: petId, State: state})
	}
	return &pb.PetRecruitInfo{
		Slots:              slots,
		AutoRefreshUsed:    m.Entity.AutoRefreshUsed,
		NextRefreshTime:    m.Entity.NextAutoRefreshTime,
		ManualRefreshFree:  m.Entity.FreeManualRefresh,
		DiamondRecruitUsed: m.Entity.DiamondRecruitUsed,
		RecruitCost:        0,
		TripleRecruit:      m.Entity.TripleRecruit,
		RecruitType:        recruitTypeCoupon,
	}
}

func collectRemainingSlotIndexes(slotStates tool.JSONInt32Slice) []int32 {
	remainSlotIdx := make([]int32, 0, recruitSlotCount)
	if len(slotStates) != recruitSlotCount {
		return remainSlotIdx
	}
	for i := 0; i < recruitSlotCount; i++ {
		if slotStates[i] == slotStateRecruitable {
			remainSlotIdx = append(remainSlotIdx, int32(i))
		}
	}
	return remainSlotIdx
}

func isFirstSummonInRound(slotStates tool.JSONInt32Slice) bool {
	if len(slotStates) != recruitSlotCount {
		return true
	}
	for _, state := range slotStates {
		if state != slotStateRecruitable {
			return false
		}
	}
	return true
}

func fillNextRecruitOffer(m *model.PetRecruitModel, info *pb.PetRecruitInfo) {
	if m == nil || m.Entity == nil || info == nil {
		return
	}
	cands := m.Entity.CandidatePetIDs
	if len(cands) != recruitSlotCount || len(m.Entity.SlotStates) != recruitSlotCount {
		return
	}

	remainSlotIdx := collectRemainingSlotIndexes(m.Entity.SlotStates)
	if len(remainSlotIdx) == 0 {
		return
	}

	var weightSum int64 = 0
	var weightedValueSum int64 = 0
	for _, si := range remainSlotIdx {
		base := gameConfig.GetPetBaseCfg(cands[si])
		if base == nil {
			continue
		}
		summon := gameConfig.GetPetSummonCfg(base.PetPotential)
		if summon == nil || summon.Weight <= 0 {
			continue
		}
		weightSum += int64(summon.Weight)
		weightedValueSum += int64(summon.Value) * int64(summon.Weight)
	}

	firstInRound := isFirstSummonInRound(m.Entity.SlotStates)
	isTriple := m.Entity.TripleRecruit == featureEnabled

	// 新规则：当轮首抽只能使用券（不再根据背包是否有券回退到钻石）。
	if firstInRound {
		info.RecruitType = recruitTypeCoupon
		info.RecruitCost = singleRecruitCost
		if isTriple {
			info.RecruitCost = tripleRecruitMultiplier
		}
		return
	}

	// 默认按钻石展示下一次花销
	info.RecruitType = recruitTypeDiamond
	info.RecruitCost = calcCostByRemaining(weightedValueSum, weightSum, m.Entity.DiamondRecruitUsed, isTriple)
}

func syncAutoRefresh(m *model.PetRecruitModel, nowMilli int64) {
	if m == nil {
		return
	}
	m.ApplySystemAutoRefreshTick(nowMilli)
}

// GetDetail 仅返回当前面板信息（不扣费）。会同步系统自动刷新状态；候选由「系统刷新/手动刷新/招满刷新」生成，此处不懒 roll。
func GetDetail(player *model.PlayerModel, nowMilli int64) (*pb.PetRecruitInfo, pb.ERROR_CODE, error) {
	m, err := GetOrLoadModel(player)
	if err != nil {
		return nil, pb.ERROR_CODE_INVALID_REQUEST_PARAM, err
	}
	syncAutoRefresh(m, nowMilli)
	if !m.IsCandidateReady() {
		return nil, pb.ERROR_CODE_PET_RECRUIT_CANDIDATES_NOT_READY, errors.New("petRecruit candidates not ready")
	}

	info := BuildInfo(m)
	fillNextRecruitOffer(m, info)
	return info, pb.ERROR_CODE_SUCCESS, nil
}

func ManualRefresh(player *model.PlayerModel, sanctuaryLevel int32, nowMilli int64) (*pb.PetRecruitInfo, pb.ERROR_CODE, error) {
	m, err := GetOrLoadModel(player)
	if err != nil {
		return nil, pb.ERROR_CODE_INVALID_REQUEST_PARAM, err
	}

	// 无心跳也能同步系统刷新状态
	syncAutoRefresh(m, nowMilli)

	manualFree := m.Entity.FreeManualRefresh == featureEnabled
	if !manualFree && len(m.Entity.SlotStates) == recruitSlotCount {
		for _, state := range m.Entity.SlotStates {
			if state == slotStateRecruited {
				manualFree = true
				break
			}
		}
	}
	if !manualFree {
		cost := gameConfig.GetPetSummonRefreshDiamond()
		if cost > 0 {
			costItems := []*gameConfig.ItemConfig{{ID: enum.DIAMOND_ITEM_ID, Num: int64(cost)}}
			ok, err := itemService.CheckItemsCount(player, costItems)
			if err != nil || !ok {
				return nil, pb.ERROR_CODE_PET_RECRUIT_REFRESH_NOT_ENOUGH, err
			}
			if err := itemService.RemoveItems(player, costItems, enum.ITEM_CHANGE_REASON_USE_ITEM); err != nil {
				return nil, pb.ERROR_CODE_SYSTEM_ERROR, err
			}
		}
	}

	ids, err := RollCandidates(player, sanctuaryLevel)
	if err != nil {
		return nil, pb.ERROR_CODE_CFG_NOT_FOUND, err
	}
	if err := m.UpdateCandidates(ids); err != nil {
		return nil, pb.ERROR_CODE_SYSTEM_ERROR, err
	}
	m.UpdateSlotStates(tool.JSONInt32Slice{slotStateRecruitable, slotStateRecruitable, slotStateRecruitable})
	m.UpdateFreeManualRefresh(featureDisabled)
	m.UpdateTripleRecruit(featureDisabled)
	m.ResetCountdownOnManualOrFullRecruit(nowMilli)

	info := BuildInfo(m)
	fillNextRecruitOffer(m, info)
	return info, pb.ERROR_CODE_SUCCESS, nil
}

func computeWeightedPickIndex(weights []int32) int32 {
	var total int32 = 0
	for _, w := range weights {
		if w > 0 {
			total += w
		}
	}
	if total <= 0 {
		return -1
	}
	r := tool.RandInt32(1, total)
	var acc int32 = 0
	for i, w := range weights {
		if w <= 0 {
			continue
		}
		acc += w
		if r <= acc {
			return int32(i)
		}
	}
	return -1
}

func calcCostByRemaining(weightedValueSum int64, weightSum int64, diamondUsedToday int32, isTriple bool) int32 {
	// base = (Σ value_i * weight_i) / (Σ weight_i)
	if weightedValueSum <= 0 || weightSum <= 0 {
		return 0
	}
	base := weightedValueSum / weightSum
	if base < 0 {
		base = 0
	}
	total := base
	if isTriple {
		total *= int64(tripleRecruitMultiplier)
	}

	discountCount := gameConfig.GetPetSummonDiscountCount()
	rates := gameConfig.GetPetSummonDiscount()
	// rates: 万分比。
	// - 若配置 2 个值，约定 [普通, 三倍]
	// - 若仅 1 个值，则普通/三倍共用
	rate := int32(10000)
	if len(rates) == 1 {
		rate = rates[0]
	} else if len(rates) >= 2 {
		if isTriple {
			rate = rates[1]
		} else {
			rate = rates[0]
		}
	}
	if discountCount > 0 && diamondUsedToday < discountCount && rate > 0 && rate < 10000 {
		total = total * int64(rate) / 10000
	}
	if total < 0 {
		total = 0
	}
	if total > int64(int32(^uint32(0)>>1)) {
		total = int64(int32(^uint32(0) >> 1))
	}
	return int32(total)
}

type recruitContext struct {
	pickedSlotIndex      int32
	firstPetID           int32
	weightSum            int64
	weightedValueSum     int64
	isFirstTripleInRound bool
	isTriple             bool
}

func buildRecruitContext(m *model.PetRecruitModel, triple int32) (*recruitContext, pb.ERROR_CODE, error) {
	cands := m.Entity.CandidatePetIDs
	if len(cands) != recruitSlotCount {
		return nil, pb.ERROR_CODE_SYSTEM_ERROR, errors.New("petRecruit invalid candidates")
	}

	remainSlotIdx := collectRemainingSlotIndexes(m.Entity.SlotStates)
	if len(remainSlotIdx) == 0 {
		return nil, pb.ERROR_CODE_INVALID_REQUEST_PARAM, errors.New("petRecruit no remaining slots")
	}

	remainWeights := make([]int32, 0, len(remainSlotIdx))
	var weightSum int64
	var weightedValueSum int64
	for _, si := range remainSlotIdx {
		base := gameConfig.GetPetBaseCfg(cands[si])
		if base == nil {
			return nil, pb.ERROR_CODE_CFG_NOT_FOUND, errors.New("petRecruit petBase cfg nil")
		}
		summon := gameConfig.GetPetSummonCfg(base.PetPotential)
		if summon == nil {
			return nil, pb.ERROR_CODE_CFG_NOT_FOUND, errors.New("petRecruit petSummon cfg nil")
		}
		remainWeights = append(remainWeights, summon.Weight)
		if summon.Weight > 0 {
			weightSum += int64(summon.Weight)
			weightedValueSum += int64(summon.Value) * int64(summon.Weight)
		}
	}

	pickInRemain := computeWeightedPickIndex(remainWeights)
	if pickInRemain < 0 || int(pickInRemain) >= len(remainSlotIdx) {
		return nil, pb.ERROR_CODE_SYSTEM_ERROR, errors.New("petRecruit pick index failed")
	}
	pickedSlotIndex := remainSlotIdx[pickInRemain] // 0-based

	requestedTriple := triple != featureDisabled
	isFirstTripleInRound := requestedTriple && m.Entity.TripleRecruit == featureDisabled

	return &recruitContext{
		pickedSlotIndex:      pickedSlotIndex,
		firstPetID:           cands[pickedSlotIndex],
		weightSum:            weightSum,
		weightedValueSum:     weightedValueSum,
		isFirstTripleInRound: isFirstTripleInRound,
		isTriple:             requestedTriple,
	}, pb.ERROR_CODE_SUCCESS, nil
}

func consumeCostItems(player *model.PlayerModel, items []*gameConfig.ItemConfig, notEnoughCode pb.ERROR_CODE) (pb.ERROR_CODE, error) {
	ok, err := itemService.CheckItemsCount(player, items)
	if err != nil || !ok {
		return notEnoughCode, err
	}
	if err := itemService.RemoveItems(player, items, enum.ITEM_CHANGE_REASON_USE_ITEM); err != nil {
		return pb.ERROR_CODE_SYSTEM_ERROR, err
	}
	return pb.ERROR_CODE_SUCCESS, nil
}

func payRecruitCost(player *model.PlayerModel, m *model.PetRecruitModel, recruitType int32, ctx *recruitContext) (pb.ERROR_CODE, error) {
	// 新规则：当轮首抽只能券招募，不允许钻石招募。
	if isFirstSummonInRound(m.Entity.SlotStates) && recruitType != recruitTypeCoupon {
		return pb.ERROR_CODE_PET_RECRUIT_COUPON_ONLY_FIRST_SUMMON, errors.New("petRecruit first summon only allows coupon")
	}

	switch recruitType {
	case recruitTypeDiamond:
		cost := calcCostByRemaining(ctx.weightedValueSum, ctx.weightSum, m.Entity.DiamondRecruitUsed, ctx.isTriple)
		if cost > 0 {
			code, err := consumeCostItems(player, []*gameConfig.ItemConfig{{ID: enum.DIAMOND_ITEM_ID, Num: int64(cost)}}, pb.ERROR_CODE_PET_RECRUIT_NOT_ENOUGH)
			if code != pb.ERROR_CODE_SUCCESS {
				return code, err
			}
		}
		// 折扣次数：按“点击招募次数”计数（与是否三倍无关），避免三倍把折扣次数消耗翻倍导致体验不一致。
		m.UpdateDiamondRecruitUsed(m.Entity.DiamondRecruitUsed + 1)
		return pb.ERROR_CODE_SUCCESS, nil
	case recruitTypeCoupon:
		// 招募券仅允许“当轮第一次召唤”，防止玩家用券在同一轮连续招（与 fillNextRecruitOffer 的展示规则一致）。
		if !isFirstSummonInRound(m.Entity.SlotStates) {
			return pb.ERROR_CODE_PET_RECRUIT_COUPON_ONLY_FIRST_SUMMON, errors.New("petRecruit coupon only allowed on first summon")
		}
		couponNum := int64(singleRecruitCost)
		if ctx.isTriple {
			// 三倍招募：券按 3 张消耗（与钻石招募保持同倍率语义）
			couponNum = int64(tripleRecruitMultiplier)
		}
		return consumeCostItems(player, []*gameConfig.ItemConfig{{ID: petRecruitCouponItemID, Num: couponNum}}, pb.ERROR_CODE_PET_RECRUIT_COUPON_NOT_ENOUGH)
	default:
		return pb.ERROR_CODE_INVALID_REQUEST_PARAM, errors.New("petRecruit invalid recruitType")
	}
}

func grantRecruitPet(player *model.PlayerModel, petID int32, isTriple bool) (pb.ERROR_CODE, error) {
	// 发放宠物：不直接写入 PetModel，而是通过“宠物卡 itemId”走 item 自动使用链路，
	// 复用背包变更、获得宠物 push、统计上报等通用逻辑，避免重复实现两套“发宠物”路径。
	cardItemID := gameConfig.GetPetCardItemID(petID)
	if cardItemID <= 0 {
		return pb.ERROR_CODE_PET_RECRUIT_PET_CARD_ITEM_NOT_FOUND, errors.New("petRecruit pet card item not found")
	}
	num := int64(singleRecruitCost)
	if isTriple {
		num = int64(tripleRecruitMultiplier)
	}
	if err := itemService.AddItems(player, []*gameConfig.ItemConfig{{ID: cardItemID, Num: num}}, enum.ITEM_CHANGE_REASON_PET_RECRUIT); err != nil {
		return pb.ERROR_CODE_SYSTEM_ERROR, err
	}
	eventBusService.SubmitLuckyLotteryEvent(player.GetUserId(), "pet", 1, []*gameConfig.ItemConfig{{ID: cardItemID, Num: num}})
	return pb.ERROR_CODE_SUCCESS, nil
}

func applyRecruitResultState(m *model.PetRecruitModel, ctx *recruitContext, player *model.PlayerModel, sanctuaryLevel int32, nowMilli int64) (pb.ERROR_CODE, error) {
	if ctx.isFirstTripleInRound {
		// TripleRecruit 是“当轮是否已触发过三倍”的持久状态，用于：
		// - 影响客户端展示下一次招募类型与成本
		m.UpdateTripleRecruit(featureEnabled)
	}
	// 槽位标记已招募（triple 仅影响数量/价格）
	_ = m.MarkSlotRecruited(ctx.pickedSlotIndex + 1)

	// 招募过至少 1 只：手动刷新免费
	if m.Entity.FreeManualRefresh == featureDisabled {
		m.UpdateFreeManualRefresh(featureEnabled)
	}

	// 招满 3 只：立即自动刷新一次（不计入每日系统刷新次数）。
	// 目的：保证面板永远有可招募候选；并按现有规则重置倒计时（到10次后不重置）。
	if !m.AllSlotsTaken() {
		return pb.ERROR_CODE_SUCCESS, nil
	}
	ids, err := RollCandidates(player, sanctuaryLevel)
	if err != nil {
		return pb.ERROR_CODE_CFG_NOT_FOUND, err
	}
	_ = m.UpdateCandidates(ids)
	m.UpdateSlotStates(tool.JSONInt32Slice{slotStateRecruitable, slotStateRecruitable, slotStateRecruitable})
	m.UpdateFreeManualRefresh(featureDisabled)
	m.UpdateTripleRecruit(featureDisabled)
	m.ResetCountdownOnManualOrFullRecruit(nowMilli)
	return pb.ERROR_CODE_SUCCESS, nil
}

// RecruitPetFromCandidates 按 petSummon 权重从 3 个候选中 roll 获得 1（或3）只宠物，并扣费。
func RecruitPetFromCandidates(player *model.PlayerModel, recruitType int32, triple int32, sanctuaryLevel int32, nowMilli int64) (*pb.PetRecruitResp, pb.ERROR_CODE, error) {
	m, err := GetOrLoadModel(player)
	if err != nil {
		return nil, pb.ERROR_CODE_INVALID_REQUEST_PARAM, err
	}

	// 同步系统自动刷新（心跳可能没来）；候选须已由刷新逻辑生成
	syncAutoRefresh(m, nowMilli)
	if !m.IsCandidateReady() {
		return nil, pb.ERROR_CODE_PET_RECRUIT_CANDIDATES_NOT_READY, errors.New("petRecruit candidates not ready")
	}

	ctx, code, err := buildRecruitContext(m, triple)
	if code != pb.ERROR_CODE_SUCCESS {
		return nil, code, err
	}
	code, err = payRecruitCost(player, m, recruitType, ctx)
	if code != pb.ERROR_CODE_SUCCESS {
		return nil, code, err
	}
	oldPetOwnIDs := make(map[int64]bool)
	if player.PetModel != nil {
		for ownID := range player.PetModel.Entities {
			oldPetOwnIDs[ownID] = true
		}
	}
	code, err = grantRecruitPet(player, ctx.firstPetID, ctx.isTriple)
	if code != pb.ERROR_CODE_SUCCESS {
		return nil, code, err
	}
	recruitCount := int32(0)
	if player.PetModel != nil {
		for ownID, ent := range player.PetModel.Entities {
			if oldPetOwnIDs[ownID] || ent == nil || ent.IsDeleted || ent.PetID != ctx.firstPetID {
				continue
			}
			player.StaticData.UpdatePetRecruitCount(player.StaticData.GetPetRecruitCount() + 1)
			recruitCount++
			operationLogService.OnUserPetRecruit(player.GetUserId(), ent.PetID, ent.PetOwnID)
		}
	}
	if recruitCount > 0 {
		err = unlockSvc.DailyCache.RecordPetRecruit(context.Background(), player.GetUserId(), recruitCount)
		if err != nil {
			platformLogger.ErrorWithUser("record pet recruit error", player, err)
		}
	}
	code, err = applyRecruitResultState(m, ctx, player, sanctuaryLevel, nowMilli)
	if code != pb.ERROR_CODE_SUCCESS {
		return nil, code, err
	}

	info := BuildInfo(m)
	info.TripleRecruit = m.Entity.TripleRecruit
	fillNextRecruitOffer(m, info)

	return &pb.PetRecruitResp{
		BasicInfo:   info,
		GainedPetId: ctx.firstPetID,
	}, pb.ERROR_CODE_SUCCESS, nil
}
