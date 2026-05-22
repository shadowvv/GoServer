# 玩家数据导出导入优化方案

## 一、背景与目标

当前 `exportPlayer` 导出玩家的全部数据为 INSERT SQL，`importPlayer` 直接执行这些 SQL。
存在以下问题：
- 导入时如果目标数据库已有相同主键数据，会产生主键冲突
- 无法实现「导出A玩家数据 → 覆盖到B玩家身上」的运营需求

### 优化目标
1. 导入时指定目标玩家（通过 account + server_id 定位）
2. account（UserEntity）表生成 UPDATE 语句：保持 `account` 和 `server_id` 不变，将 `user_id` 改为新生成的值，其他属性字段用导出数据覆盖
3. 其他所有子模型：直接 INSERT，使用新 `user_id` + 新唯一 ID
4. **不删除**目标玩家旧数据——旧 user_id 下的数据自然变成孤立数据，不再被查询到（开发服不介意残留）
5. 所有唯一 ID 重新生成，所有交叉引用同步映射

---

## 二、涉及的唯一 ID 及其关联

| 唯一ID字段 | 所属 Entity | 生成器类型 | 被其他 Entity 引用情况 |
|---|---|---|---|
| `user_id` | UserEntity | ID_GENERATOR_USER | 几乎所有子模型的 user_id 字段 |
| `hero_own_id` | HeroDetailsEntity | ID_GENERATOR_HERO | EquipmentEntity.hero_own_id, AccessoryEntity.hero_own_id, PetEntity.hero_own_id, HeroFormation.hero_own_id_list |
| `equipment_own_id` | EquipmentEntity | ID_GENERATOR_EQUIPMENT | HeroDetailsEntity.equipment_id (JSON数组) |
| `pet_own_id` | PetEntity | ID_GENERATOR_ITEM | — |

### 关联映射说明

- `HeroDetailsEntity.EquipmentId`（JSON int64 数组）→ 引用 `equipment_own_id`
- `EquipmentEntity.HeroOwnID` → 引用 `hero_own_id`
- `AccessoryEntity.HeroOwnId` → 引用 `hero_own_id`
- `PetEntity.HeroOwnId` → 引用 `hero_own_id`
- `HeroFormationEntity.HeroOwnIDList` → 引用 `hero_own_id`

---

## 三、整体方案设计

### 3.1 接口协议变更

#### 导入请求 `GmImportPlayerReq` 新增字段：
```go
type GmImportPlayerReq struct {
    Token        string `json:"token"`
    Sql          string `json:"sql"`           // 导出的原始 SQL（保留兼容）
    TargetUserId int64  `json:"target_user_id"` // 目标玩家ID（要覆盖的玩家）
}
```

#### 导入响应 `GmImportPlayerData` 补充：
```go
type GmImportPlayerData struct {
    UserId    int64 `json:"user_id"`     // 最终写入的 user_id
    Msg       string `json:"msg"`        // 详细信息
}
```

### 3.2 新增导出模式（推荐方案）

**不再以纯 SQL 文本为传输格式**，改为结构化 JSON 导出，在导入端统一处理 ID 重映射。

#### 新导出数据结构：
```go
type PlayerExportData struct {
    SourceUserId int64                    `json:"source_user_id"`
    User         *UserEntity              `json:"user"`
    Models       map[string]interface{}   `json:"models"` // key=模型名, value=实体切片JSON
}
```

**但考虑到改动成本和兼容性**，推荐以下分步方案：

---

### 3.3 最终确认方案（user_id 软切换 + 不删旧数据）

#### 核心思路
- 给目标玩家生成一个**全新的 user_id**
- UPDATE UserEntity，把目标玩家的 user_id 改成新值（account、server_id 不变）
- 导出数据中的所有子模型：反序列化后替换为新 user_id + 新唯一 ID，直接 INSERT
- 旧 user_id 下的数据留在库中不管（孤立数据，开发服无所谓）

#### 导出阶段（`GmExportPlayer`）
改造为结构化 JSON 导出（不再是纯 SQL 文本），返回所有 Entity 的序列化数据。

#### 导入阶段（`GmImportPlayer`）核心改造

**流程：**
1. 解析请求中的 `target_account` + `target_server_id`（定位目标玩家）
2. 生成全新 `new_user_id`（雪花 ID）
3. 构建 ID 映射表：
   - `user_id` → `new_user_id`
   - 为每个旧 `hero_own_id` 生成新的映射
   - 为每个旧 `equipment_own_id` 生成新的映射
   - 为每个旧 `pet_own_id` 生成新的映射
4. 在事务中：
   - **Step 1**：UPDATE UserEntity SET user_id = new_user_id WHERE account = ? AND server_id = ?（同时覆盖昵称、等级等属性）
   - **Step 2**：遍历所有子模型 Entity，替换 user_id 和所有唯一 ID 及其交叉引用
   - **Step 3**：生成 INSERT SQL 并执行（全新数据，不会冲突）

---

## 四、详细执行计划

### Phase 1：后端改造

| 步骤 | 内容 | 文件 |
|---|---|---|
| 1 | 新增 `PlayerExportPayload` 结构体，包含所有子模型的序列化数据 | `model/playerModel.go` |
| 2 | 新增 `ExportStructured()` 方法，返回 `PlayerExportPayload` JSON | `model/playerModel.go` |
| 3 | 每个子模型新增 `ExportEntities()` 方法返回实体切片 | 各 model 文件 |
| 4 | 新增 `IdRemapper` 工具结构，负责维护 old→new 映射并生成新 ID | `backend/idRemapper.go`（新文件） |
| 5 | 改造 `GmImportPlayer`，解析结构化 JSON，执行 ID 重映射 + SQL 生成 | `backend/backendHandle.go` |
| 6 | 对 UserEntity 生成 UPDATE SQL（修改 user_id，保留 account/server_id） | `backend/backendHandle.go` |
| 7 | 更新 `GmImportPlayerReq` 协议，新增 `target_account` + `target_server_id` | `backend/backendProto.go` |

### Phase 2：需要处理 ID 重映射的字段清单

| Entity | 需替换的字段 | 说明 |
|---|---|---|
| UserEntity | user_id → new_user_id | UPDATE 语句，保留 account+server_id |
| StaticDataEntity | userId → new | — |
| StoryTriggerEntity | user_id → new | — |
| HeroDetailsEntity | user_id → new, hero_own_id → new | 主键重生成 |
| HeroDetailsEntity | equipment_id (JSON) | 内部引用 equipment_own_id 需映射 |
| HeroFormationEntity | user_id → new, hero_own_id_list(JSON数组) → new映射 | `[]int64` JSON格式 |
| HeroAlbumEntity | user_id → new | — |
| EquipmentEntity | user_id → new, equipment_own_id → new, hero_own_id → new映射 | 双向关联 |
| AccessoryEntity | user_id → new, hero_own_id → new映射 | — |
| PetEntity | user_id → new, pet_own_id → new, hero_own_id → new映射 | — |
| PetAffinityEntity | user_id → new | — |
| PetRecruitEntity | user_id → new | — |
| TaskEntity | user_id → new | — |
| BountyEntity | user_id → new | — |
| ExpeditionEntity | user_id → new | — |
| ExpeditionSlotEntity | user_id → new | — |
| LotteryEntity | user_id → new | — |
| LoopBoxEntity | user_id → new | — |
| PassEntity | user_id → new | — |
| VipCardEntity | user_id → new | — |
| PrivilegeRewardEntity | user_id → new | — |
| PlayerShopEntity | user_id → new | — |
| PlayerInstanceEntity | user_id → new | — |
| PlayerActivityEntity | user_id → new | — |
| PlayerSignEntity | user_id → new | — |
| IdleEntity | user_id → new | — |
| PlayerAdventureEntity | user_id → new | — |
| PlayerArenaEntity | user_id → new | — |
| ArchitectureEntity | user_id → new | — |
| StoneEntity | user_id → new | — |
| LumberEntity | user_id → new | — |
| FurnitureEntity | user_id → new | — |
| TrialEntity | user_id → new | — |
| CollectionEntity | user_id → new | — |
| AppearanceEntity | user_id → new | — |
| CityAgeEntity | user_id → new | — |
| PlayerFunctionEntity | user_id → new | — |
| AdChestEntity | user_id → new | — |
| TokenShopEntity | user_id → new | — |
| AlbumRewardEntity | user_id → new | — |

### Phase 3：前端改造

| 步骤 | 内容 | 文件 |
|---|---|---|
| 1 | 导入时增加 `target_account` + `target_server_id` 输入框 | `src/views/coin.vue` |
| 2 | 更新 `GmImportPlayerReq` TypeScript 接口 | `src/def/user.ts` |

---

## 五、ID 重映射伪代码

```go
type IdRemapper struct {
    heroOwnIdMap      map[int64]int64  // oldHeroOwnId → newHeroOwnId
    equipOwnIdMap     map[int64]int64  // oldEquipOwnId → newEquipOwnId
    petOwnIdMap       map[int64]int64  // oldPetOwnId → newPetOwnId
    newUserId         int64            // 新生成的 user_id
}

func NewIdRemapper() *IdRemapper {
    return &IdRemapper{
        heroOwnIdMap:  make(map[int64]int64),
        equipOwnIdMap: make(map[int64]int64),
        petOwnIdMap:   make(map[int64]int64),
        newUserId:     UserIdGenerator.NextId(),  // 生成全新 user_id
    }
}

func (r *IdRemapper) RemapHeroOwnId(oldId int64) int64 {
    if oldId == 0 { return 0 }
    if newId, ok := r.heroOwnIdMap[oldId]; ok { return newId }
    newId := HeroIdGenerator.NextId()
    r.heroOwnIdMap[oldId] = newId
    return newId
}

// RemapEquipOwnId, RemapPetOwnId 类似...

// 处理 HeroFormation 的 hero_own_id_list (JSON数组)
func (r *IdRemapper) RemapHeroOwnIdList(oldList []int64) []int64 {
    newList := make([]int64, len(oldList))
    for i, oldId := range oldList {
        newList[i] = r.RemapHeroOwnId(oldId)
    }
    return newList
}
```

---

## 六、数据处理策略（不删旧数据）

- **不执行任何 DELETE**
- UserEntity：UPDATE 修改 user_id 为新值（account + server_id 不变）
- 所有子模型：直接 INSERT 新数据（新 user_id + 新唯一 ID，不会与旧数据冲突）
- 旧 user_id 下的数据自然成为孤立数据，不再被任何查询命中
- 后续如需清理，可另写脚本按孤立 user_id 批量删除

---

## 七、风险与注意事项

1. **联盟/社交数据**：联盟成员关联、好友关系等不在此导出范围，需确认是否需要处理
2. **排行榜数据**：排行榜是独立服务，导入后可能需要手动触发刷新
3. **邮件数据**：当前未导出邮件，导入不影响邮件
4. **缓存一致性**：导入后如果目标玩家在线，需要踢下线或刷新内存缓存
5. **事务安全**：UPDATE + 所有 INSERT 必须在同一事务中执行
6. **ID 生成器并发安全**：雪花 ID 生成器已内置 mutex，并发调用安全

---

## 八、已确认事项

1. ✅ 导入时保留目标玩家的 `account`/`server_id` 不变，只更新 `user_id` 及属性字段
2. ✅ 旧数据不删除，留在库中作为孤立数据（开发服不介意）
3. ✅ `HeroFormationEntity.hero_own_id_list` 为 JSON 数组格式（`tool.JSONInt64Slice` = `[]int64`，存储如 `[123,456,789]`）
4. ✅ 所有唯一 ID 通过雪花 ID 生成器重新生成，保证不与库中任何数据冲突

---

## 九、预计工作量

- 后端核心改造：约 2-3 天
- 前端适配：约 0.5 天
- 联调测试：约 1 天
