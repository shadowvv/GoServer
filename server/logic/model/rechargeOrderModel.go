package model

type RechargeOrderEntity struct {
	OrderId         int64  `gorm:"column:order_id;primaryKey"`
	PlatformOrderId string `gorm:"column:platform_order_id"`
	Account         string `gorm:"column:account"`
	ServerId        int32  `gorm:"column:server_id"`
	UserId          int64  `gorm:"column:user_id"`
	ShopItemId      int32  `gorm:"column:shop_item_id"`
	ProductId       int32  `gorm:"column:product_id"`
	Price           int32  `gorm:"column:price"`
	PickedItems     string `gorm:"column:picked_items"`
	Status          int32  `gorm:"column:status"`
	CreateTime      int64  `gorm:"column:create_time"`
	PayTime         int64  `gorm:"column:pay_time"`
	DeliverTime     int64  `gorm:"column:deliver_time"`
	PayType         int32  `gorm:"column:pay_type"`
	PayPlatform     string `gorm:"column:pay_platform"`
	PayToken        string `gorm:"column:pay_token"`
	IsSandBox       int32  `gorm:"column:is_sand_box"`
	ExtraInfo       string `gorm:"column:extra_info"`
}

func (r *RechargeOrderEntity) TableName() string {
	return "recharge_order"
}
