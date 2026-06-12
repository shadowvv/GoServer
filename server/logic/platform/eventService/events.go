package eventService

import (
	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/service/logger"
	"go.uber.org/zap"
)

type GameEvent interface {
	GetEventType() string
	GetObjectID() int64
}

type HeroLevelUpEvent struct {
	PlayerID int64
	HeroId   int32
	NewLevel int32
	OldLevel int32
}

func (e *HeroLevelUpEvent) GetEventType() string {
	return enum.EventTypeHeroLevelUp
}
func (e *HeroLevelUpEvent) GetObjectID() int64 {
	return e.PlayerID
}
func NewHeroLevelUpEvent(playerID int64, heroId int32, oldLevel, newLevel int32) *HeroLevelUpEvent {
	return &HeroLevelUpEvent{
		PlayerID: playerID,
		HeroId:   heroId,
		OldLevel: oldLevel,
		NewLevel: newLevel,
	}
}

type AccessoryLevelUpEvent struct {
	PlayerID    int64
	AccessoryID int32
	OldLevel    int32
	NewLevel    int32
}

func (e *AccessoryLevelUpEvent) GetEventType() string {
	return enum.EventTypeAccessoryLevelUp
}

func (e *AccessoryLevelUpEvent) GetObjectID() int64 {
	return e.PlayerID
}

func NewAccessoryLevelUpEvent(playerID int64, accessoryID int32, oldLevel, newLevel int32) *AccessoryLevelUpEvent {
	return &AccessoryLevelUpEvent{
		PlayerID:    playerID,
		AccessoryID: accessoryID,
		OldLevel:    oldLevel,
		NewLevel:    newLevel,
	}
}

type MonsterInfo struct {
	MonsterType int32 // 怪物类型
	MonsterId   int32 // 怪物ID
	Count       int32 // 数量
}

type KillMonsterEvent struct {
	PlayerID    int64
	SceneId     int32 // 副本ID
	MonsterList []*MonsterInfo
}

func (e *KillMonsterEvent) GetEventType() string {
	return enum.EventTypeKillMonster
}

func (e *KillMonsterEvent) GetObjectID() int64 { return e.PlayerID }
func NewKillMonsterEvent(playerID int64, sceneId int32, monsterIds []int32) *KillMonsterEvent {
	res := &KillMonsterEvent{
		PlayerID:    playerID,
		SceneId:     sceneId,
		MonsterList: make([]*MonsterInfo, 0),
	}
	monsterMap := make(map[int32]int32)
	for _, v := range monsterIds {
		monsterMap[v]++
	}
	for id, v := range monsterMap {
		res.MonsterList = append(res.MonsterList, &MonsterInfo{
			MonsterType: gameConfig.GetMonsterCfg(id).Type,
			MonsterId:   id,
			Count:       v,
		})
	}
	return res
}

type PassInstanceEvent struct {
	PlayerID       int64
	ServerId       int32
	InstanceTypeId enum.InstanceId // 副本类型ID
	InstanceId     int32           // 关卡ID
}

func (e *PassInstanceEvent) GetEventType() string {
	return enum.EventTypePassInstance
}

func (e *PassInstanceEvent) GetObjectID() int64 { return e.PlayerID }
func NewPassInstanceEvent(playerID int64, serverId int32, instanceTypeIdlId enum.InstanceId, instanceId int32) *PassInstanceEvent {
	return &PassInstanceEvent{
		PlayerID:       playerID,
		ServerId:       serverId,
		InstanceTypeId: instanceTypeIdlId,
		InstanceId:     instanceId,
	}
}

type ItemInfo struct {
	ItemType    int32
	ItemQuality int32
	ItemId      int32 // 物品ID
	Count       int64 // 数量
}

type ItemCollectEvent struct {
	PlayerID     int64
	ItemInfoList map[int32]*ItemInfo
}

func (e *ItemCollectEvent) GetEventType() string {
	return enum.EventTypeItemCollect
}

func (e *ItemCollectEvent) GetObjectID() int64 { return e.PlayerID }
func NewItemCollectEvent(playerID int64, itemInfoList []*gameConfig.ItemConfig) *ItemCollectEvent {
	res := &ItemCollectEvent{
		PlayerID:     playerID,
		ItemInfoList: make(map[int32]*ItemInfo),
	}
	for _, v := range itemInfoList {
		cfg := gameConfig.GetItemCfg(v.ID)
		if cfg == nil {
			logger.ErrorWithZapFields("NewItemCollectEvent failed config = nil", zap.Int64("playerID", playerID), zap.Int32("itemID", v.ID))
			continue
		}
		if res.ItemInfoList[v.ID] == nil {
			res.ItemInfoList[v.ID] = &ItemInfo{
				ItemId:      v.ID,
				ItemType:    cfg.Type,
				ItemQuality: cfg.Quality,
				Count:       v.Num,
			}
		} else {
			res.ItemInfoList[v.ID].Count += v.Num
		}
	}
	return res
}

type LuckyLotteryEvent struct {
	PlayerID    int64
	LotteryType string
	LotteryNum  int32
	RewardItems []*gameConfig.ItemConfig
}

func (e *LuckyLotteryEvent) GetEventType() string {
	return enum.EventTypeLuckyLottery
}

func (e *LuckyLotteryEvent) GetObjectID() int64 { return e.PlayerID }
func NewLuckyLotteryEvent(playerID int64, lotteryType string, lotteryNum int32, rewardItem []*gameConfig.ItemConfig) *LuckyLotteryEvent {
	return &LuckyLotteryEvent{
		PlayerID:    playerID,
		LotteryType: lotteryType,
		LotteryNum:  lotteryNum,
		RewardItems: rewardItem,
	}
}

type HeroStarUpEvent struct {
	PlayerID  int64
	HeroId    int32
	StarLevel int32
}

func (e *HeroStarUpEvent) GetEventType() string {
	return enum.EventTypeHeroStarUp
}
func (e *HeroStarUpEvent) GetObjectID() int64 {
	return e.PlayerID
}
func NewHeroStarUpEvent(playerID int64, heroId int32, starLevel int32) *HeroStarUpEvent {
	return &HeroStarUpEvent{
		PlayerID:  playerID,
		HeroId:    heroId,
		StarLevel: starLevel,
	}
}

type PlayerLevelUpEvent struct {
	PlayerID int64
	ServerId int32
	Level    int32
}

func (e *PlayerLevelUpEvent) GetEventType() string {
	return enum.EventTypePlayerLevelUp
}

func (e *PlayerLevelUpEvent) GetObjectID() int64 {
	return e.PlayerID
}

func NewPlayerLevelUpEvent(playerID int64, serverId int32, level int32) *PlayerLevelUpEvent {
	return &PlayerLevelUpEvent{
		PlayerID: playerID,
		ServerId: serverId,
		Level:    level,
	}
}

type JoinInstanceEvent struct {
	PlayerID       int64
	ServerId       int32
	InstanceTypeId enum.InstanceId // 副本类型ID
	InstanceId     int32           // 关卡ID
}

func (e *JoinInstanceEvent) GetEventType() string {
	return enum.EventTypeJoinInstance
}

func (e *JoinInstanceEvent) GetObjectID() int64 { return e.PlayerID }
func NewJoinInstanceEvent(playerID int64, serverId int32, instanceTypeIdlId enum.InstanceId, instanceId int32) *JoinInstanceEvent {
	return &JoinInstanceEvent{
		PlayerID:       playerID,
		ServerId:       serverId,
		InstanceTypeId: instanceTypeIdlId,
		InstanceId:     instanceId,
	}
}

type QuickClaimMachineRewardEvent struct {
	PlayerID int64
	ServerId int32
}

func (e *QuickClaimMachineRewardEvent) GetEventType() string {
	return enum.EventTypeQuickClaimMachineReward
}

func (e *QuickClaimMachineRewardEvent) GetObjectID() int64 { return e.PlayerID }
func NewQuickClaimMachineRewardEvent(playerID int64, serverId int32) *QuickClaimMachineRewardEvent {
	return &QuickClaimMachineRewardEvent{
		PlayerID: playerID,
		ServerId: serverId,
	}
}

type BuildLevelUpEvent struct {
	PlayerID   int64
	ServerId   int32
	BuildId    enum.ArchitectureType
	BuildLevel int32
}

func (e *BuildLevelUpEvent) GetEventType() string {
	return enum.EventTypeBuildLevelUp
}

func (e *BuildLevelUpEvent) GetObjectID() int64 { return e.PlayerID }
func NewBuildLevelUpEvent(playerID int64, serverId int32, buildId enum.ArchitectureType, buildLevel int32) *BuildLevelUpEvent {
	return &BuildLevelUpEvent{
		PlayerID:   playerID,
		ServerId:   serverId,
		BuildId:    buildId,
		BuildLevel: buildLevel,
	}
}

type LoopBoxLevelUpEvent struct {
	PlayerID  int64
	ServerId  int32
	OldLevel  int32
	NewLevel  int32
	SystemExp int32
}

func (e *LoopBoxLevelUpEvent) GetEventType() string {
	return enum.EventTypeLoopBoxLevelUp
}

func (e *LoopBoxLevelUpEvent) GetObjectID() int64 {
	return e.PlayerID
}

func NewLoopBoxLevelUpEvent(playerID int64, serverId int32, oldLevel, newLevel, systemExp int32) *LoopBoxLevelUpEvent {
	return &LoopBoxLevelUpEvent{
		PlayerID:  playerID,
		ServerId:  serverId,
		OldLevel:  oldLevel,
		NewLevel:  newLevel,
		SystemExp: systemExp,
	}
}

type DispatchKillMonsterEvent struct {
	PlayerID int64
	ServerId int32
}

func (e *DispatchKillMonsterEvent) GetEventType() string {
	return enum.EventTypeDispatchKillMonster
}
func (e *DispatchKillMonsterEvent) GetObjectID() int64 { return e.PlayerID }
func NewDispatchKillMonsterEvent(playerID int64, serverId int32) *DispatchKillMonsterEvent {
	return &DispatchKillMonsterEvent{
		PlayerID: playerID,
		ServerId: serverId,
	}
}

type PlayerPowerChangeEvent struct {
	PlayerID int64
	ServerId int32
	Power    int64
}

func (e *PlayerPowerChangeEvent) GetEventType() string {
	return enum.EventTypePlayerPowerChange
}
func (e *PlayerPowerChangeEvent) GetObjectID() int64 { return e.PlayerID }
func NewPlayerPowerChangeEvent(playerID int64, serverId int32, power int64) *PlayerPowerChangeEvent {
	return &PlayerPowerChangeEvent{
		PlayerID: playerID,
		ServerId: serverId,
		Power:    power,
	}
}

type EquipmentStrongEvent struct {
	PlayerID       int64
	EquipmentOwnID int64
	OldLevel       int32
	NewLevel       int32
}

func (e *EquipmentStrongEvent) GetEventType() string {
	return enum.EventTypeEquipmentStrong
}

func (e *EquipmentStrongEvent) GetObjectID() int64 {
	return e.PlayerID
}

func NewEquipmentStrongEvent(playerID int64, equipmentOwnID int64, oldLevel, newLevel int32) *EquipmentStrongEvent {
	return &EquipmentStrongEvent{
		PlayerID:       playerID,
		EquipmentOwnID: equipmentOwnID,
		OldLevel:       oldLevel,
		NewLevel:       newLevel,
	}
}

type AllianceJoinEvent struct {
	PlayerID int64
}

func (e *AllianceJoinEvent) GetEventType() string {
	return enum.EventTypeAllianceJoin
}

func (e *AllianceJoinEvent) GetObjectID() int64 {
	return e.PlayerID
}

func NewAllianceJoinEvent(playerID int64) *AllianceJoinEvent {
	return &AllianceJoinEvent{PlayerID: playerID}
}

type PetStarUpEvent struct {
	PlayerID int64
	PetOwnID int64
	OldStar  int32
	NewStar  int32
}

func (e *PetStarUpEvent) GetEventType() string {
	return enum.EventTypePetStarUp
}

func (e *PetStarUpEvent) GetObjectID() int64 {
	return e.PlayerID
}

func NewPetStarUpEvent(playerID int64, petOwnID int64, oldStar, newStar int32) *PetStarUpEvent {
	return &PetStarUpEvent{
		PlayerID: playerID,
		PetOwnID: petOwnID,
		OldStar:  oldStar,
		NewStar:  newStar,
	}
}

type EquipmentForgeEvent struct {
	PlayerID   int64
	ForgeCount int32
}

func (e *EquipmentForgeEvent) GetEventType() string {
	return enum.EventTypeEquipmentForge
}

func (e *EquipmentForgeEvent) GetObjectID() int64 {
	return e.PlayerID
}

func NewEquipmentForgeEvent(playerID int64, forgeCount int32) *EquipmentForgeEvent {
	return &EquipmentForgeEvent{
		PlayerID:   playerID,
		ForgeCount: forgeCount,
	}
}

type EquipmentWearEvent struct {
	PlayerID int64
}

func (e *EquipmentWearEvent) GetEventType() string {
	return enum.EventTypeEquipmentWear
}

func (e *EquipmentWearEvent) GetObjectID() int64 {
	return e.PlayerID
}

func NewEquipmentWearEvent(playerID int64) *EquipmentWearEvent {
	return &EquipmentWearEvent{PlayerID: playerID}
}

type ArenaScoreChangeEvent struct {
	PlayerID int64
	Score    int32
}

func (e *ArenaScoreChangeEvent) GetEventType() string {
	return enum.EventTypeArenaScoreChange
}

func (e *ArenaScoreChangeEvent) GetObjectID() int64 {
	return e.PlayerID
}

func NewArenaScoreChangeEvent(playerID int64, score int32) *ArenaScoreChangeEvent {
	return &ArenaScoreChangeEvent{
		PlayerID: playerID,
		Score:    score,
	}
}

type AdChestOpenEvent struct {
	PlayerID  int64
	OpenCount int32
}

func (e *AdChestOpenEvent) GetEventType() string {
	return enum.EventTypeAdChestOpen
}

func (e *AdChestOpenEvent) GetObjectID() int64 {
	return e.PlayerID
}

func NewAdChestOpenEvent(playerID int64, openCount int32) *AdChestOpenEvent {
	return &AdChestOpenEvent{
		PlayerID:  playerID,
		OpenCount: openCount,
	}
}

type MainTaskChangeEvent struct {
	PlayerID int64
}

func (e *MainTaskChangeEvent) GetEventType() string {
	return enum.EventTypeMainTaskChange
}

func (e *MainTaskChangeEvent) GetObjectID() int64 {
	return e.PlayerID
}

func NewMainTaskChangeEvent(playerID int64) *MainTaskChangeEvent {
	return &MainTaskChangeEvent{PlayerID: playerID}
}

type StoneAttrLevelUpEvent struct {
	PlayerID   int64
	Class      int32
	AttrId     int32
	LevelUpNum int32
}

func (e *StoneAttrLevelUpEvent) GetEventType() string {
	return enum.EventTypeStoneAttrLevelUp
}

func (e *StoneAttrLevelUpEvent) GetObjectID() int64 {
	return e.PlayerID
}

func NewStoneAttrLevelUpEvent(playerID int64, class int32, attrId int32, levelUpNum int32) *StoneAttrLevelUpEvent {
	return &StoneAttrLevelUpEvent{PlayerID: playerID, Class: class, AttrId: attrId, LevelUpNum: levelUpNum}
}

type AddHeroAlbumEvent struct {
	PlayerID int64
	HeroID   int32
}

func (e *AddHeroAlbumEvent) GetEventType() string {
	return enum.EventTypeAddHeroAlbum
}
func (e *AddHeroAlbumEvent) GetObjectID() int64 { return e.PlayerID }
func NewAddHeroAlbumEvent(playerID int64, heroID int32) *AddHeroAlbumEvent {
	return &AddHeroAlbumEvent{
		PlayerID: playerID,
		HeroID:   heroID,
	}
}

type PetLevelUpEvent struct {
	PlayerID int64
	PetOwnID int64
	OldLevel int32
	NewLevel int32
}

func (e *PetLevelUpEvent) GetEventType() string {
	return enum.EventTypePetLevelUp
}

func (e *PetLevelUpEvent) GetObjectID() int64 {
	return e.PlayerID
}

func NewPetLevelUpEvent(playerID int64, petOwnID int64, oldLevel, newLevel int32) *PetLevelUpEvent {
	return &PetLevelUpEvent{
		PlayerID: playerID,
		PetOwnID: petOwnID,
		OldLevel: oldLevel,
		NewLevel: newLevel,
	}
}

type EventTypePlayerLogin struct {
	ServerId int32
}

type EventTypeCityAgeChange struct {
	PlayerID int64
	CityAge  int32
}

func (e *EventTypeCityAgeChange) GetEventType() string {
	return enum.EventTypeCityAgeChange
}
func (e *EventTypeCityAgeChange) GetObjectID() int64 {
	return e.PlayerID
}
func NewEventTypeCityAgeChangeEvent(playerID int64, cityAge int32) *EventTypeCityAgeChange {
	return &EventTypeCityAgeChange{
		PlayerID: playerID,
		CityAge:  cityAge,
	}
}
