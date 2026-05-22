package inventory

import "github.com/drop/GoServer/server/logic/model"

// 兼容外部引用（如 backend GM 查询）：
// inventory.PlayerInventoryEntity -> model.PlayerInventoryEntity
type PlayerInventoryEntity = model.PlayerInventoryEntity
