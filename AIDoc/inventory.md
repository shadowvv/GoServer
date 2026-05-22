# Inventory 实现说明（并入 PlayerModel）

本文档描述当前背包系统的实现状态：背包数据已从独立缓存服务模式，收敛到玩家模型体系，作为 `PlayerModel` 的一个子模型统一加载、心跳、落库。

## 1. 目标与现状

之前的背包实现位于 `logic/inventory`，内部维护独立缓存、用户锁和手动落库逻辑，和其它玩家子系统（`playerModel` 下各 model）不一致。

现在改为：

1. 背包数据模型迁移到 `server/logic/model/inventoryModel.go`。
2. `PlayerModel` 持有 `InventoryModel` 字段。
3. 登录加载、创建角色时都把 `InventoryModel` 挂入 `PlayerModels`，统一参与 `SavePlayerToDB`。
4. `inventoryService` 变为薄服务层，只负责通过 `sessionManager` 找到在线玩家并调用 `player.InventoryModel`。

## 2. 关键代码位置

- `server/logic/model/inventoryModel.go`
  - `PlayerInventoryEntity`
  - `InventoryModel`
  - `LoadInventoryModel / CreateInventoryModel`
  - `AddItem / RemoveItem / GetItemList / GetItemCount / HasItem`
  - `SaveModelToDB`
- `server/logic/model/playerModel.go`
  - `PlayerModel.InventoryModel *InventoryModel`
- `server/logic/gameController/loginController.go`
  - 加载玩家时：`LoadInventoryModel` 并 append 到 `PlayerModels`
  - 创建玩家时：`CreateInventoryModel` 并 append 到 `PlayerModels`
- `server/logic/inventory/inventoryService.go`
  - `NewInventoryService(sessionManager)`
  - 对外接口全部转发到 `player.InventoryModel`
- `server/logic/gameController/inventoryController.go`
  - `InitinvService()` 改为传入 `sessionManager`
- `server/logic/inventory/inventoryModel.go`
  - 保留兼容别名：`type PlayerInventoryEntity = model.PlayerInventoryEntity`

## 3. 数据结构

`PlayerInventoryEntity`（表：`player_inventory`）：

- `id`
- `user_id`
- `item_id`
- `item_num`

`InventoryModel` 内存结构：

- `Items map[int32]*PlayerInventoryEntity`
- `ChangedItems map[int32]*PlayerInventoryEntity`
- `NewItems map[int32]bool`
- `DeletedItems map[int32]bool`

说明：当前主背包仍是“按 `itemId` 聚合计数”模型，不维护格子/槽位。

## 4. 生命周期与调用链

### 4.1 登录加载

`loadPlayerFromDB` 中执行：

1. `inventoryModel, err := model.LoadInventoryModel(userId)`
2. `player.InventoryModel = inventoryModel`
3. `player.PlayerModels = append(player.PlayerModels, player.InventoryModel)`

### 4.2 新号创建

`CreatePlayer` 中执行：

1. `player.InventoryModel = model.CreateInventoryModel(userId)`
2. `player.AppendPlayerModel(player.InventoryModel)`

### 4.3 保存落库

`PlayerModel.SavePlayerToDB()` 会遍历 `PlayerModels` 调用 `SaveModelToDB()`，背包模型自动参与统一保存。

## 5. InventoryService 新职责

`InventoryService` 不再维护独立 `cache/userLocks/tx`，只做业务访问入口：

1. 通过 `sessionManager.GetPlayerBasicInfoByUserId(userId)` 获取在线玩家。
2. 断言为 `*model.PlayerModel`。
3. 调用 `player.InventoryModel` 的增删查接口。
4. 若在线玩家未挂载 `InventoryModel`，按需 `CreateInventoryModel` 并 append 到 `PlayerModels`。

对外接口保持不变：

- `AddItem`
- `RemoveItem`
- `GetItemList`
- `GetItemCount`
- `HasItem`
- `AddItems`
- `RemoveItems`

## 6. 落库策略（InventoryModel.SaveModelToDB）

按变更集增量落库：

1. `DeletedItems`：按 `user_id + item_id` 删除。
2. `NewItems`：创建新记录。
3. 其余 `ChangedItems`：更新 `item_num`。
4. 落库后清空 `ChangedItems/NewItems/DeletedItems`。

## 7. 兼容性说明

为避免影响已有引用，`logic/inventory/inventoryModel.go` 保留了类型别名：

- `inventory.PlayerInventoryEntity` -> `model.PlayerInventoryEntity`

这样 backend/GM 等仍引用 `inventory.PlayerInventoryEntity` 的代码不会立即失效。

## 8. 影响与约束

1. 背包服务默认依赖在线玩家上下文（通过 session manager 获取 player）。
2. 背包数据的持久化时机与其它 player model 一致，由主保存链路驱动。
3. 旧的独立背包缓存管理逻辑不再是主路径，应避免继续在新逻辑中扩展该模式。

## 9. 后续建议

1. 清理 `logic/inventory` 内已退役实现（如旧 manager 逻辑），降低双实现维护成本。
2. 若需要离线改包（非在线玩家），建议在 model 层补充明确的离线访问入口，而不是回退独立缓存架构。
