package enum

// GM后台权限枚举，与前端 permissData 对应
const (
	ViewUserInfoPermission      int32 = 1  // 基础信息（查看）
	ViewItemChgPermission       int32 = 2  // 资产流水（查看）
	ViewOrderPermission         int32 = 3  // 订单列表（查看）
	RepairOrderPermission       int32 = 4  // 补单（操作）
	ViewMailPermission          int32 = 6  // 玩家邮件（查看）
	SendMailPermission          int32 = 7  // 发送邮件（操作）
	DeleteMailPermission        int32 = 8  // 删除邮件（操作）
	ViewServerListPermission    int32 = 9  // 服务器列表（查看）
	EditServerPermission        int32 = 10 // 修改添加服务器（操作）
	EditGamePublicPermission    int32 = 15 // 游戏内公告（操作）
	EditActivityPermission      int32 = 16 // 活动管理/后台监控（操作）
	ViewRankPermission          int32 = 17 // 排行榜（查看）
	GmPermission                int32 = 19 // GM权限管理（操作）
	EditClientVersionPermission int32 = 20 // 客户端版本（操作）
	ViewUserLogPermission       int32 = 21 // 玩家操作（查看）
	EditBanUserPermission       int32 = 22 // 封禁用户（操作）
	EditMuteUserPermission      int32 = 23 // 禁言用户（操作）
	ImportPlayerPermission      int32 = 26 // 导入玩家数据（操作）
	ExportPlayerPermission      int32 = 27 // 导出玩家数据（操作）
	KickPlayerPermission        int32 = 28 // 踢人（操作）
	BuFaPermission              int32 = 29 // 补发（操作）
)

// IsPermiss 检查权限列表中是否包含指定权限
func IsPermiss(permissList []int32, permiss int32) bool {
	for _, v := range permissList {
		if v == permiss {
			return true
		}
	}
	return false
}
