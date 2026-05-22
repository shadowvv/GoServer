package enum

type ArchitectureType int32

const (
	ARCHITECTURE_TYPE_MAIN       ArchitectureType = 1 // 城镇中心
	ARCHITECTURE_TYPE_STONE      ArchitectureType = 2 // 传承石像
	ARCHITECTURE_TYPE_PET        ArchitectureType = 3 // 宠物圣殿
	ARCHITECTURE_TYPE_COLLECTION ArchitectureType = 4 // 藏品楼
	ARCHITECTURE_TYPE_EQUIPMENT  ArchitectureType = 5 // 装备圣殿
	ARCHITECTURE_TYPE_LUMBERYARD ArchitectureType = 6 // 伐木场
)

var AllArchitectureType = map[int32]ArchitectureType{
	int32(ARCHITECTURE_TYPE_MAIN):       ARCHITECTURE_TYPE_MAIN,
	int32(ARCHITECTURE_TYPE_STONE):      ARCHITECTURE_TYPE_STONE,
	int32(ARCHITECTURE_TYPE_COLLECTION): ARCHITECTURE_TYPE_COLLECTION,
	int32(ARCHITECTURE_TYPE_PET):        ARCHITECTURE_TYPE_PET,
	int32(ARCHITECTURE_TYPE_EQUIPMENT):  ARCHITECTURE_TYPE_EQUIPMENT,
	int32(ARCHITECTURE_TYPE_LUMBERYARD): ARCHITECTURE_TYPE_LUMBERYARD,
}

func IsValidArchitectureType(t int32) bool {
	_, ok := AllArchitectureType[t]
	return ok
}

func GetArchitectureTypeName(t int32) int32 {
	systemUnlockId := int32(0)
	switch ArchitectureType(t) {
	case ARCHITECTURE_TYPE_STONE:
		systemUnlockId = FUNCTION_ID_STONE
	case ARCHITECTURE_TYPE_PET:
		systemUnlockId = FUNCTION_ID_PET_SANCTUARY
	case ARCHITECTURE_TYPE_COLLECTION:
		systemUnlockId = FUNCTION_ID_COLLECTION
	case ARCHITECTURE_TYPE_EQUIPMENT:
		systemUnlockId = FUNCTION_ID_EQUIPMENT
	case ARCHITECTURE_TYPE_LUMBERYARD:
		systemUnlockId = FUNCTION_ID_LUMBERYARD
	}
	return systemUnlockId
}
