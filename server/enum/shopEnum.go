package enum

// 商品类型枚举
type ShopItemType int32

const (
	ShopItemTypeDailyDiscount ShopItemType = 1  //每日特惠
	ShopItemTypeCombination   ShopItemType = 2  //组合礼包
	ShopItemTypeWeekly        ShopItemType = 3  //周卡月卡
	ShopItemTypeChain         ShopItemType = 4  //链式礼包
	ShopItemTypeTwoChoice     ShopItemType = 5  //两种自选礼包
	ShopItemTypeThreeChoice   ShopItemType = 6  //三种自选礼包
	ShopItemTypeRelay         ShopItemType = 7  //接力型礼包
	ShopItemTypeToken         ShopItemType = 8  //钻石储值
	ShopItemTypeCoupon        ShopItemType = 9  //代金券
	ShopItemTypeFirstCharge   ShopItemType = 10 //首充
	ShopItemTypeAirdrop       ShopItemType = 11 //空投
	ShopItemTypeBattlePass    ShopItemType = 12 //通行证
	ShopItemTypeVipCard       ShopItemType = 13 //特权
	ShopItemTypeTrial         ShopItemType = 14 //试炼
	ShopItemTypeCommon        ShopItemType = 15 //通用
)

// 验证商品类型枚举
func IsValidShopItemType(shopItemType int32) bool {
	if shopItemType >= int32(ShopItemTypeDailyDiscount) && shopItemType <= int32(ShopItemTypeCommon) {
		return true
	}
	return false
}
