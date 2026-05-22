package enum

type InstanceId int32

const (
	MAIN_INSTANCE_ID               InstanceId = 1   // 主线副本
	FIVE_VS_FIVE_TOWER_INSTANCE_ID InstanceId = 101 // 5v5爬塔副本
	ARENA_INSTANCE_ID              InstanceId = 102 // 竞技场副本
	ADVENTURE_INSTANCE_ID          InstanceId = 103 // 奇遇副本
	COIN_INSTANCE_ID               InstanceId = 104 // 金币副本
	CAPSULE_INSTANCE_ID            InstanceId = 105 // 胶囊副本
	HERO_INSTANCE_ID               InstanceId = 106 // 英雄副本
	PET_INSTANCE_ID                InstanceId = 107 // 宠物副本
	GLORY_ARENA_INSTANCE_ID        InstanceId = 108 // 荣耀擂台
)

// 副本类型枚举
// AdventureType 对应 adventure / dungeon_adventure 表里的秘境类型。
type AdventureType int32

const (
	AdventureType_GOLD    AdventureType = 1 // 金币秘境
	AdventureType_CAPSULE AdventureType = 2 // 胶囊秘境
	AdventureType_HERO    AdventureType = 3 // 英雄秘境
	AdventureType_PET     AdventureType = 4 // 宠物秘境
)

type DungeonAdventureInstanceType int32

const (
	DungeonAdventureInstanceType_ADVENTURE DungeonAdventureInstanceType = 1
	DungeonAdventureInstanceType_INSTANCE  DungeonAdventureInstanceType = 2
)

type InstanceTypeEnum int32

const (
	InstanceType_MAIN        InstanceTypeEnum = iota + 1 // 主线副本
	InstanceType_TOWER                                   // 爬塔副本
	InstanceType_ARENA                                   // 竞技场
	InstanceType_ENCOUNTER                               // 奇遇副本
	InstanceType_COIN                                    // 金币副本
	InstanceType_CAPSULE                                 // 胶囊副本
	InstanceType_HERO                                    // 英雄副本
	InstanceType_PET                                     // 宠物副本
	InstanceType_GLORY_ARENA                             // 荣耀擂台
)

// 是否有效副本类型
func IsValidInstanceType(v int32) bool {
	return v >= int32(InstanceType_MAIN) && v <= int32(InstanceType_GLORY_ARENA)
}

// 怪物类型枚举
type MonsterType int32

const (
	MonsterType_BUCKET MonsterType = iota + 1 // 桶
	MonsterType_NORMAL                        // 普通
	MonsterType_ELITE                         // 精英
	MonsterType_BOSS                          // Boss
	MonsterType_HERO                          // 英雄
)

// 是否有效怪物类型
func IsValidMonsterType(v int32) bool {
	return v >= int32(MonsterType_BUCKET) && v <= int32(MonsterType_HERO)
}

func IsResidentInstanceType(v int32) bool {
	switch InstanceTypeEnum(v) {
	case InstanceType_CAPSULE, InstanceType_HERO, InstanceType_COIN, InstanceType_PET:
		return true
	default:
		return false
	}
}

// 伤害类型枚举
type DamageType int32

const (
	DamageType_PHY DamageType = iota + 1 // 物理
	DamageType_MAG                       // 魔法
)

// 是否有效伤害类型
func IsValidDamageType(v int32) bool {
	return v >= int32(DamageType_PHY) && v <= int32(DamageType_MAG)
}

func GetResidentDungeonType(instanceType int32) (int32, bool) {
	switch InstanceTypeEnum(instanceType) {
	case InstanceType_COIN:
		return int32(COIN_INSTANCE_ID), true
	case InstanceType_CAPSULE:
		return int32(CAPSULE_INSTANCE_ID), true
	case InstanceType_HERO:
		return int32(HERO_INSTANCE_ID), true
	case InstanceType_PET:
		return int32(PET_INSTANCE_ID), true
	default:
		return 0, false
	}
}

func GetResidentInstanceTypeByDungeonType(dungeonType int32) (int32, bool) {
	switch InstanceId(dungeonType) {
	case COIN_INSTANCE_ID:
		return int32(InstanceType_COIN), true
	case CAPSULE_INSTANCE_ID:
		return int32(InstanceType_CAPSULE), true
	case HERO_INSTANCE_ID:
		return int32(InstanceType_HERO), true
	case PET_INSTANCE_ID:
		return int32(InstanceType_PET), true
	default:
		return 0, false
	}
}
