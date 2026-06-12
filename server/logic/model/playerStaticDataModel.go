package model

import (
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
)

type StaticDataEntity struct {
	UserId                     int64 `gorm:"column:userId;primary_key"`
	ChangeNicknameTimes        int32 `gorm:"column:changeNicknameTimes"`
	ChargeTimes                int32 `gorm:"column:charge_times"`
	HeroHistoryMaxLevel        int32 `gorm:"column:hero_history_max_level"`
	DailyPrivilegeDrop         int32 `gorm:"column:daily_privilege_drop"`
	ArenaChallengeTimes        int32 `gorm:"column:arena_challenge_times"`
	ExpeditionNum              int32 `gorm:"column:expedition_num"`
	BuyDispatchFormationNum    int32 `gorm:"column:buy_dispatch_formation_num"`
	PetRecruitCount            int32 `gorm:"column:pet_recruit_count"`
	CollectionLotteryDrawCount int32 `gorm:"column:collection_lottery_draw_count"`
	GloryArenaJoinCount        int32 `gorm:"column:glory_arena_join_count"`
	ResidentInstanceJoinCount  int32 `gorm:"column:resident_instance_join_count"`
	BattleSpeedUpTimes         int32 `gorm:"column:battle_speed_up_times"`
	DailyBattleSpeedUpTimes    int32 `gorm:"column:daily_battle_speed_up_times"`
}

func (u *StaticDataEntity) TableName() string {
	return "player_static_data"
}

type StaticDataModel struct {
	Entity  *StaticDataEntity
	Changed map[string]interface{}
}

var _ logicCommon.PlayerModelInterface = (*StaticDataModel)(nil)

func NewStaticDataModel(entity *StaticDataEntity) *StaticDataModel {
	return &StaticDataModel{
		Entity:  entity,
		Changed: make(map[string]interface{}),
	}
}

func (s *StaticDataModel) UpdateChangeNicknameTimes(changeNumTimes int32) {
	s.Entity.ChangeNicknameTimes = changeNumTimes
	// 使用与列名一致的 key，确保写入数据库
	s.Changed["changeNicknameTimes"] = changeNumTimes
}

func (s *StaticDataModel) UpdateChargeTimes(chargeTimes int32) {
	s.Entity.ChargeTimes = chargeTimes
	s.Changed["charge_times"] = chargeTimes
}

func (s *StaticDataModel) AddExpeditionNum(num int32) {
	s.Entity.ExpeditionNum += num
	s.Changed["expedition_num"] = s.Entity.ExpeditionNum
}

func (s *StaticDataModel) GetExpeditionNum() int32 {
	return s.Entity.ExpeditionNum
}

func (s *StaticDataModel) UpdateHeroHistoryMaxLevel(heroHistoryMaxLevel int32) {
	s.Entity.HeroHistoryMaxLevel = heroHistoryMaxLevel
	s.Changed["hero_history_max_level"] = heroHistoryMaxLevel
}

func (s *StaticDataModel) GetBuyDispatchFormationNum() int32 {
	return s.Entity.BuyDispatchFormationNum
}

func (s *StaticDataModel) UpdateBuyDispatchFormationNum(num int32) {
	s.Entity.BuyDispatchFormationNum = num
	s.Changed["buy_dispatch_formation_num"] = num
}

func (s *StaticDataModel) GetPetRecruitCount() int32 {
	return s.Entity.PetRecruitCount
}

func (s *StaticDataModel) UpdatePetRecruitCount(num int32) {
	s.Entity.PetRecruitCount = num
	s.Changed["pet_recruit_count"] = num
}

func (s *StaticDataModel) GetCollectionLotteryDrawCount() int32 {
	return s.Entity.CollectionLotteryDrawCount
}

func (s *StaticDataModel) UpdateCollectionLotteryDrawCount(num int32) {
	s.Entity.CollectionLotteryDrawCount = num
	s.Changed["collection_lottery_draw_count"] = num
}

func (s *StaticDataModel) GetGloryArenaJoinCount() int32 {
	return s.Entity.GloryArenaJoinCount
}

func (s *StaticDataModel) UpdateGloryArenaJoinCount(num int32) {
	s.Entity.GloryArenaJoinCount = num
	s.Changed["glory_arena_join_count"] = num
}

func (s *StaticDataModel) GetResidentInstanceJoinCount() int32 {
	return s.Entity.ResidentInstanceJoinCount
}

func (s *StaticDataModel) UpdateResidentInstanceJoinCount(num int32) {
	s.Entity.ResidentInstanceJoinCount = num
	s.Changed["resident_instance_join_count"] = num
}

func (s *StaticDataModel) UpdateDailyPrivilegeDrop(dailyPrivilegeDrop int32) {
	if dailyPrivilegeDrop <= 0 {
		dailyPrivilegeDrop = 0
	}
	s.Entity.DailyPrivilegeDrop = dailyPrivilegeDrop
	s.Changed["daily_privilege_drop"] = dailyPrivilegeDrop
}

func (s *StaticDataModel) AddArenaChallengeTimes(times int32) {
	s.Entity.ArenaChallengeTimes = s.Entity.ArenaChallengeTimes + times
	s.Changed["arena_challenge_times"] = s.Entity.ArenaChallengeTimes
}

func (s *StaticDataModel) UpdateBattleSpeedUpTimes(times int32) {
	s.Entity.BattleSpeedUpTimes = s.Entity.BattleSpeedUpTimes + times
	s.Changed["battle_speed_up_times"] = s.Entity.BattleSpeedUpTimes
}

func (s *StaticDataModel) UpdateDailyBattleSpeedUpTimes(times int32) {
	s.Entity.DailyBattleSpeedUpTimes = s.Entity.DailyBattleSpeedUpTimes + times
	s.Changed["daily_battle_speed_up_times"] = s.Entity.DailyBattleSpeedUpTimes
}

func (s *StaticDataModel) GetChangeNicknameTimes() int32 {
	return s.Entity.ChangeNicknameTimes
}

func (s *StaticDataModel) GetDailyPrivilegeDrop() int32 {
	return s.Entity.DailyPrivilegeDrop
}

func (s *StaticDataModel) GetChargeTimes() int32 {
	return s.Entity.ChargeTimes
}

func (s *StaticDataModel) GetHeroHistoryMaxLevel() int32 {
	return s.Entity.HeroHistoryMaxLevel
}

func (s *StaticDataModel) GetArenaChallengeTimes() int32 {
	return s.Entity.ArenaChallengeTimes
}

func (s *StaticDataModel) GetBattleSpeedUpTimes() int32 {
	return s.Entity.BattleSpeedUpTimes
}

func (s *StaticDataModel) GetDailyBattleSpeedUpTimes() int32 {
	return s.Entity.DailyBattleSpeedUpTimes
}

func (s *StaticDataModel) SaveModelToDB() {
	if s.Changed == nil || len(s.Changed) == 0 {
		return
	}
	easyDB.UpdatePlayerEntity(s.Entity, s.Changed, s.Entity.UserId)
	s.Changed = make(map[string]interface{})
}

func (s *StaticDataModel) Heartbeat(lastTickTime int64, currentTime int64, passDay int32, senderMsg bool) {
	if passDay > 0 {
		s.UpdateDailyPrivilegeDrop(gameConfig.GetDailyPrivilegeDropQuantityLimit())
		s.UpdateDailyBattleSpeedUpTimes(0)
	}
}

func LoadStaticDataModel(userId int64) (*StaticDataModel, error) {
	entity, err := easyDB.GetPlayerEntityByID[StaticDataEntity](userId)
	if err != nil {
		return nil, err
	}
	model := NewStaticDataModel(entity)
	return model, nil
}

func CreateStaticDataModel(userId int64) (*StaticDataModel, error) {
	entity := &StaticDataEntity{
		UserId:              userId,
		ChangeNicknameTimes: 0,
		HeroHistoryMaxLevel: 0,
	}
	model := NewStaticDataModel(entity)
	err := easyDB.CreatePlayerEntity(entity)
	if err != nil {
		return nil, err
	}
	return model, nil
}
