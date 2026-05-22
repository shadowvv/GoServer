package model

import (
	"fmt"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/tool"
)

const (
	// 面板固定 3 个槽位：所有候选列表/状态数组都按 3 个元素读写（与客户端协议一致）。
	petRecruitSlotCount = 3
	milliPerSecond      = int64(1000)
	petRecruitDayMillis = int64(24 * 60 * 60 * 1000)

	// 槽位招募状态。0=可招募，1=已招募（与 pb.PetRecruitSlot.state 一致）。
	petRecruitStateRecruitable int32 = 0
	petRecruitStateRecruited   int32 = 1

	// 通用开关标记：用于 DB 字段（0/1）与协议字段对齐，避免布尔落库带来的兼容问题。
	petRecruitFlagOff int32 = 0
	petRecruitFlagOn  int32 = 1

	// 默认展示：用于没有明确客户端选择时的面板初始值（实际展示会在 service 侧覆盖计算）。
	petRecruitDefaultRecruitType int32 = 1
	petRecruitDefaultSanctuaryLv int32 = 1
)

// petRecruitPrivilegeCheck 由 petRecruit 包注入，用于避免 model -> vipCard 的 import 循环依赖。
var petRecruitPrivilegeCheck func(*PlayerModel) bool

// SetPetRecruitPrivilegeChecker 注册“是否有特权”的判断函数，用于选择 PetSanctuaryCfg.DropGroupId2。
func SetPetRecruitPrivilegeChecker(fn func(*PlayerModel) bool) {
	petRecruitPrivilegeCheck = fn
}

// petRecruitInfoBuilder 由 petRecruit 包注入，用于在 push 场景下组装完整的招募面板信息（例如下一次报价）。
var petRecruitInfoBuilder func(*PlayerModel, *PetRecruitModel) *pb.PetRecruitInfo

// SetPetRecruitInfoBuilder 注册 push 路径使用的面板信息构建函数。
func SetPetRecruitInfoBuilder(fn func(*PlayerModel, *PetRecruitModel) *pb.PetRecruitInfo) {
	petRecruitInfoBuilder = fn
}

// PetRecruitEntity 玩家维度的“宠物招募”状态与计数（按轮次/按日）。
// 这里只存状态与计数；候选池/权重/价值等规则仍以配置为准。
type PetRecruitEntity struct {
	UserID int64 `gorm:"column:user_id;primaryKey"`

	// 今日已使用的“系统自动刷新”次数。
	AutoRefreshUsed int32 `gorm:"column:auto_refresh_used"`

	// 下次系统自动刷新触发时间（毫秒时间戳）。
	NextAutoRefreshTime int64 `gorm:"column:next_auto_refresh_time"`

	// 当前候选宠物ID列表。固定长度：3。
	CandidatePetIDs tool.JSONInt32Slice `gorm:"column:candidate_pet_ids;type:json"`

	// 槽位招募状态。固定长度：3。0=可招募，1=已招募。
	SlotStates tool.JSONInt32Slice `gorm:"column:slot_states;type:json"`

	// 本轮是否拥有“免费手动刷新”标记。
	FreeManualRefresh int32 `gorm:"column:free_manual_refresh"`

	// 今日已使用的钻石招募次数（用于折扣/限次等规则）。
	DiamondRecruitUsed int32 `gorm:"column:diamond_recruit_used"`

	// 本轮是否已触发过三倍招募。0=否，1=是。
	TripleRecruit int32 `gorm:"column:triple_recruit"`

	// 今日是否已触发“首次打开面板的系统刷新”。0=否，1=是。
	FirstOpenRefreshed int32 `gorm:"column:first_open_refreshed"`

	// 每日重置用的“自然日标记”（如 yyyymmdd）。
	LastResetDay int32 `gorm:"column:last_reset_day"`
}

func (e *PetRecruitEntity) TableName() string {
	return "pet_recruit"
}

type PetRecruitModel struct {
	Player  *PlayerModel
	UserId  int64
	Entity  *PetRecruitEntity
	Changed map[string]interface{} // 仅记录变更字段，用于增量落库
}

var _ logicCommon.PlayerModelInterface = (*PetRecruitModel)(nil)

func NewPetRecruitModel(player *PlayerModel, userId int64, entity *PetRecruitEntity) *PetRecruitModel {
	if entity == nil {
		entity = &PetRecruitEntity{}
	}
	return &PetRecruitModel{
		Player:  player,
		UserId:  userId,
		Entity:  entity,
		Changed: make(map[string]interface{}),
	}
}

func CreatePetRecruitModel(player *PlayerModel, userId int64) (*PetRecruitModel, error) {
	today := tool.GetTodayDataIntByTimeStamp(tool.UnixNowMilli())
	entity := &PetRecruitEntity{
		UserID:              userId,
		AutoRefreshUsed:     0,
		NextAutoRefreshTime: 0,
		CandidatePetIDs:     tool.JSONInt32Slice{0, 0, 0},
		SlotStates:          tool.JSONInt32Slice{0, 0, 0},
		FreeManualRefresh:   petRecruitFlagOff,
		DiamondRecruitUsed:  0,
		TripleRecruit:       petRecruitFlagOff,
		FirstOpenRefreshed:  petRecruitFlagOff,
		LastResetDay:        today,
	}
	if err := easyDB.CreatePlayerEntity(entity); err != nil {
		return nil, err
	}
	return NewPetRecruitModel(player, userId, entity), nil
}

func (e *PetRecruitEntity) ValidateShape() error {
	if e == nil {
		return fmt.Errorf("petRecruit entity is nil")
	}
	if len(e.CandidatePetIDs) != petRecruitSlotCount {
		return fmt.Errorf("petRecruit invalid candidate_pet_ids length=%d", len(e.CandidatePetIDs))
	}
	if len(e.SlotStates) != petRecruitSlotCount {
		return fmt.Errorf("petRecruit invalid slot_states length=%d", len(e.SlotStates))
	}
	return nil
}

func LoadPetRecruitModel(player *PlayerModel, userId int64) (*PetRecruitModel, error) {
	entity := &PetRecruitEntity{UserID: userId}
	rows, err := easyDB.GetPlayerEntitiesByWhere[PetRecruitEntity](map[string]interface{}{"user_id": userId})
	if err != nil {
		return NewPetRecruitModel(player, userId, entity), err
	}
	if len(rows) > 0 && rows[0] != nil {
		entity = rows[0]
	}
	if err := entity.ValidateShape(); err != nil {
		return NewPetRecruitModel(player, userId, entity), err
	}
	if entity.LastResetDay <= 0 {
		entity.LastResetDay = tool.GetTodayDataIntByTimeStamp(tool.UnixNowMilli())
	}
	return NewPetRecruitModel(player, userId, entity), nil
}

func LoadOrCreatePetRecruitModel(player *PlayerModel) (*PetRecruitModel, error) {
	if player == nil || player.User == nil {
		return nil, fmt.Errorf("player is nil")
	}
	m, err := LoadPetRecruitModel(player, player.User.GetUserId())
	if err == nil {
		return m, nil
	}
	m2, err2 := CreatePetRecruitModel(player, player.User.GetUserId())
	if err2 == nil {
		return m2, nil
	}
	return nil, err
}

func (m *PetRecruitModel) SaveModelToDB() {
	if len(m.Changed) == 0 {
		return
	}
	if m.Entity != nil {
		easyDB.UpdatePlayerEntity[PetRecruitEntity](m.Entity, m.Changed, m.UserId)
	}
	m.Changed = make(map[string]interface{})
}

func (m *PetRecruitModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	// 心跳/登录路径：同步系统自动刷新状态机。
	changedBySystem := m.ApplySystemAutoRefreshTick(currentTime)
	if !changedBySystem || !senderMsg || m.Entity == nil || len(m.Changed) == 0 || m.Player == nil {
		return
	}
	info := m.buildPetRecruitInfoForPush()
	if petRecruitInfoBuilder != nil {
		if v := petRecruitInfoBuilder(m.Player, m); v != nil {
			info = v
		}
	}
	messageSender.SendMessage(m.Player, pb.MESSAGE_ID_PUSH_PET_RECRUIT_CHANGE, &pb.PushPetRecruitChange{
		BasicInfo: info,
	})
}

func (m *PetRecruitModel) buildPetRecruitInfoForPush() *pb.PetRecruitInfo {
	if m == nil || m.Entity == nil {
		return &pb.PetRecruitInfo{}
	}
	slots := make([]*pb.PetRecruitSlot, 0, petRecruitSlotCount)
	for i := 0; i < petRecruitSlotCount; i++ {
		petId := int32(0)
		state := petRecruitStateRecruited
		if len(m.Entity.CandidatePetIDs) == petRecruitSlotCount {
			petId = m.Entity.CandidatePetIDs[i]
		}
		if len(m.Entity.SlotStates) == petRecruitSlotCount {
			state = m.Entity.SlotStates[i]
		}
		slots = append(slots, &pb.PetRecruitSlot{
			Index: int32(i + 1),
			PetId: petId,
			State: state,
		})
	}
	return &pb.PetRecruitInfo{
		Slots:              slots,
		AutoRefreshUsed:    m.Entity.AutoRefreshUsed,
		NextRefreshTime:    m.Entity.NextAutoRefreshTime,
		ManualRefreshFree:  m.Entity.FreeManualRefresh,
		DiamondRecruitUsed: m.Entity.DiamondRecruitUsed,
		RecruitCost:        0,
		TripleRecruit:      m.Entity.TripleRecruit,
		RecruitType:        petRecruitDefaultRecruitType,
	}
}

func (m *PetRecruitModel) UpdateAutoRefreshUsed(v int32) {
	m.Entity.AutoRefreshUsed = v
	m.Changed["auto_refresh_used"] = v
}

func (m *PetRecruitModel) UpdateNextAutoRefreshTime(v int64) {
	m.Entity.NextAutoRefreshTime = v
	m.Changed["next_auto_refresh_time"] = v
}

func (m *PetRecruitModel) UpdateCandidates(ids tool.JSONInt32Slice) error {
	if len(ids) != petRecruitSlotCount {
		return fmt.Errorf("petRecruit invalid candidate_pet_ids length=%d", len(ids))
	}
	m.Entity.CandidatePetIDs = ids
	m.Changed["candidate_pet_ids"] = ids
	return nil
}

func (m *PetRecruitModel) UpdateSlotStates(states tool.JSONInt32Slice) {
	if len(states) != petRecruitSlotCount {
		return
	}
	m.Entity.SlotStates = states
	m.Changed["slot_states"] = states
}

func (m *PetRecruitModel) UpdateFreeManualRefresh(v int32) {
	m.Entity.FreeManualRefresh = v
	m.Changed["free_manual_refresh"] = v
}

func (m *PetRecruitModel) UpdateDiamondRecruitUsed(v int32) {
	m.Entity.DiamondRecruitUsed = v
	m.Changed["diamond_recruit_used"] = v
}

func (m *PetRecruitModel) UpdateTripleRecruit(v int32) {
	m.Entity.TripleRecruit = v
	m.Changed["triple_recruit"] = v
}

func (m *PetRecruitModel) UpdateFirstOpenRefreshed(v int32) {
	m.Entity.FirstOpenRefreshed = v
	m.Changed["first_open_refreshed"] = v
}

func (m *PetRecruitModel) UpdateLastResetDay(v int32) {
	m.Entity.LastResetDay = v
	m.Changed["last_reset_day"] = v
}

func (m *PetRecruitModel) EnsureCandidateShape() error {
	if m.Entity == nil {
		return fmt.Errorf("petRecruit entity is nil")
	}
	return m.Entity.ValidateShape()
}

func (m *PetRecruitModel) ResetDailyState(today int32) {
	if today <= 0 || m.Entity.LastResetDay == today {
		return
	}
	m.UpdateAutoRefreshUsed(0)
	m.UpdateDiamondRecruitUsed(0)
	m.UpdateFirstOpenRefreshed(petRecruitFlagOff)
	m.UpdateLastResetDay(today)
}

func (m *PetRecruitModel) nextDailyResetTime(nowMilli int64) int64 {
	zero := tool.GetTodayZeroByTimeStamp(nowMilli)
	return zero + petRecruitDayMillis
}

func autoRefreshIntervalMillis() int64 {
	return int64(gameConfig.GetPetSummonAutoRefreshIntervalSeconds()) * milliPerSecond
}

func emptyPetRecruitSlots() tool.JSONInt32Slice {
	return tool.JSONInt32Slice{petRecruitStateRecruitable, petRecruitStateRecruitable, petRecruitStateRecruitable}
}

func (m *PetRecruitModel) autoRefreshLimit() int32 {
	return gameConfig.GetPetSummonAutoRefreshCount()
}

func (m *PetRecruitModel) hasReachedAutoRefreshLimit() bool {
	return m.Entity.AutoRefreshUsed >= m.autoRefreshLimit()
}

func (m *PetRecruitModel) ensureDailyResetCountdown(nowMilli int64) bool {
	if m.Entity.NextAutoRefreshTime <= nowMilli {
		m.UpdateNextAutoRefreshTime(m.nextDailyResetTime(nowMilli))
		return true
	}
	return false
}

func (m *PetRecruitModel) ensureIntervalCountdown(nowMilli int64) bool {
	if m.Entity.NextAutoRefreshTime > 0 {
		return false
	}
	m.UpdateNextAutoRefreshTime(nowMilli + autoRefreshIntervalMillis())
	return true
}

func (m *PetRecruitModel) setCountdownAfterSystemRefresh(nowMilli int64) {
	if m.hasReachedAutoRefreshLimit() {
		// 达到每日上限：倒计时指向下一个自然日0点（下一次日重置）。
		m.UpdateNextAutoRefreshTime(m.nextDailyResetTime(nowMilli))
		return
	}
	m.UpdateNextAutoRefreshTime(nowMilli + autoRefreshIntervalMillis())
}
func (m *PetRecruitModel) resetCountdownIfAllowed(nowMilli int64) {
	if m.hasReachedAutoRefreshLimit() {
		// 达到每日上限：保持倒计时为下一个自然日0点。
		_ = m.ensureDailyResetCountdown(nowMilli)
		return
	}
	m.UpdateNextAutoRefreshTime(nowMilli + autoRefreshIntervalMillis())
}
func (m *PetRecruitModel) triggerSystemRefresh(nowMilli int64) {
	used := m.Entity.AutoRefreshUsed + 1
	limit := m.autoRefreshLimit()
	if used > limit {
		used = limit
	}
	m.UpdateAutoRefreshUsed(used)
	m.UpdateFreeManualRefresh(petRecruitFlagOff)

	priv := false
	if petRecruitPrivilegeCheck != nil && m.Player != nil {
		priv = petRecruitPrivilegeCheck(m.Player)
	}
	ids, err := RollPetRecruitCandidates(m.petSanctuaryLevel(), priv)
	if err != nil {
		_ = m.UpdateCandidates(emptyPetRecruitSlots())
	} else {
		_ = m.UpdateCandidates(ids)
	}
	m.UpdateSlotStates(emptyPetRecruitSlots())
	m.UpdateTripleRecruit(petRecruitFlagOff)
	m.setCountdownAfterSystemRefresh(nowMilli)
}

// ApplySystemAutoRefreshTick 系统自动刷新状态机：
// 1) 每日首次打开（FirstOpenRefreshed==0）触发一次“立即系统刷新”
// 2) 未达每日上限前：倒计时到点触发下一次系统刷新
// 3) 达到每日上限后：倒计时固定指向下一次日重置时间
func (m *PetRecruitModel) ApplySystemAutoRefreshTick(nowMilli int64) bool {
	if m == nil || m.Entity == nil {
		return false
	}

	today := tool.GetTodayDataIntByTimeStamp(nowMilli)
	m.ResetDailyState(today)

	if m.hasReachedAutoRefreshLimit() {
		return m.ensureDailyResetCountdown(nowMilli)
	}

	if m.Entity.FirstOpenRefreshed == petRecruitFlagOff {
		m.UpdateFirstOpenRefreshed(petRecruitFlagOn)
		m.triggerSystemRefresh(nowMilli)
		return true
	}

	if m.ensureIntervalCountdown(nowMilli) {
		return true
	}

	if nowMilli >= m.Entity.NextAutoRefreshTime {
		m.triggerSystemRefresh(nowMilli)
		return true
	}
	return false
}

// ResetCountdownOnManualOrFullRecruit 在“手动刷新”或“本轮招满3个槽位”后，按规则重置倒计时。
func (m *PetRecruitModel) ResetCountdownOnManualOrFullRecruit(nowMilli int64) {
	if m == nil || m.Entity == nil {
		return
	}
	m.resetCountdownIfAllowed(nowMilli)
}

func (m *PetRecruitModel) petSanctuaryLevel() int32 {
	if m.Player != nil && m.Player.ArchitectureModel != nil {
		if ent := m.Player.ArchitectureModel.Entities[int32(enum.ARCHITECTURE_TYPE_PET)]; ent != nil && ent.Level > 0 {
			return ent.Level
		}
	}
	return petRecruitDefaultSanctuaryLv
}

func selectPetRecruitDropGroupID(cfg *gameConfig.PetSanctuaryCfg, privilege bool) int32 {
	if cfg == nil {
		return 0
	}
	if privilege && cfg.DropGroupId2 > 0 {
		return cfg.DropGroupId2
	}
	return cfg.DropGroupId1
}

func rollOnePetIDFromDropGroupForRecruit(dropGroupId int32) (int32, error) {
	items := gameConfig.DropGroupItems(dropGroupId, nil)
	for _, it := range items {
		if it == nil || it.ID <= 0 {
			continue
		}
		itemCfg := gameConfig.GetItemCfg(it.ID)
		if itemCfg == nil || itemCfg.ShowGroup != int32(enum.ITEM_TYPE_PET) {
			continue
		}
		if itemCfg.TargetId > 0 {
			return itemCfg.TargetId, nil
		}
	}
	return 0, fmt.Errorf("petRecruit roll petId failed")
}

// RollPetRecruitCandidates 根据神殿等级选择掉落组，并独立 roll 3 次作为候选（允许重复，取决于掉落配置）。
func RollPetRecruitCandidates(sanctuaryLevel int32, privilege bool) (tool.JSONInt32Slice, error) {
	cfg := gameConfig.GetPetSanctuaryCfg(sanctuaryLevel)
	dropGroupId := selectPetRecruitDropGroupID(cfg, privilege)
	if dropGroupId <= 0 {
		return nil, fmt.Errorf("petRecruit invalid dropGroupId")
	}
	candidates := make(tool.JSONInt32Slice, 0, petRecruitSlotCount)
	for i := 0; i < petRecruitSlotCount; i++ {
		petID, err := rollOnePetIDFromDropGroupForRecruit(dropGroupId)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, petID)
	}
	return candidates, nil
}

func (m *PetRecruitModel) IsCandidateReady() bool {
	if len(m.Entity.CandidatePetIDs) != petRecruitSlotCount || len(m.Entity.SlotStates) != petRecruitSlotCount {
		return false
	}
	for _, id := range m.Entity.CandidatePetIDs {
		if id <= 0 {
			return false
		}
	}
	return true
}

func (m *PetRecruitModel) GetSlotState(index int32) int32 {
	if index < 1 || index > petRecruitSlotCount || len(m.Entity.SlotStates) != petRecruitSlotCount {
		return petRecruitStateRecruited
	}
	return m.Entity.SlotStates[index-1]
}

func (m *PetRecruitModel) MarkSlotRecruited(index int32) bool {
	if index < 1 || index > petRecruitSlotCount || len(m.Entity.SlotStates) != petRecruitSlotCount {
		return false
	}
	i := index - 1
	if m.Entity.SlotStates[i] == petRecruitStateRecruited {
		return false
	}
	newStates := append(tool.JSONInt32Slice(nil), m.Entity.SlotStates...)
	newStates[i] = petRecruitStateRecruited
	m.UpdateSlotStates(newStates)
	return true
}

func (m *PetRecruitModel) AllSlotsTaken() bool {
	if len(m.Entity.SlotStates) != petRecruitSlotCount {
		return false
	}
	for _, state := range m.Entity.SlotStates {
		if state != petRecruitStateRecruited {
			return false
		}
	}
	return true
}
