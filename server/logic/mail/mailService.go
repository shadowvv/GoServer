// File: mailService.go
// Description: 邮件服务实现
// Author: 木村
// Create Time: 2026.01

package mail

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"sort"

	"github.com/drop/GoServer/server/logic/platform/logicSessionManager"
	"github.com/drop/GoServer/server/logic/platform/nodeConfig"

	"github.com/drop/GoServer/server/enum"
	"github.com/drop/GoServer/server/logic/gameConfig"
	"github.com/drop/GoServer/server/logic/gameServerInfoService"
	"github.com/drop/GoServer/server/logic/itemService"
	"github.com/drop/GoServer/server/logic/logicCommon"
	"github.com/drop/GoServer/server/logic/model"
	"github.com/drop/GoServer/server/logic/pb"
	"github.com/drop/GoServer/server/logic/platform/easyDB"
	"github.com/drop/GoServer/server/service/dbService"
	"github.com/drop/GoServer/server/service/logger"
	"github.com/drop/GoServer/server/tool"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var _ logicCommon.MailServiceInterface = (*MailService)(nil)

// MailService 邮件服务实现
// 扩展：其他系统可直插 mail 表并调用 NotifyMailRefresh(userId)，本服务通过心跳检测 Redis key 后从 DB 重载该用户缓存。
type MailService struct {
	db                *gorm.DB
	cache             *MailCache
	userLocks         sync.Map          // key: int64(userId), value: *sync.Mutex
	idGenerator       *tool.IdGenerator // 统一的ID生成器（用于个人邮件和全服邮件）
	sessionManager    logicCommon.SessionManagerInterface
	messageSender     logicCommon.MessageSenderInterface
	unlockService     logicCommon.UnlockServiceInterface
	serverInfoService *gameServerInfoService.GameServerInfoService // 用于 serverDay 触发，可为 nil
}

// MailCache 邮件缓存
type MailCache struct {
	mu    sync.RWMutex
	cache map[int64]*MailManager // key: userId
}

// NewMailService 创建邮件服务
func NewMailService(sessionManager logicCommon.SessionManagerInterface, messageSender logicCommon.MessageSenderInterface, unlockService logicCommon.UnlockServiceInterface, serverInfoService *gameServerInfoService.GameServerInfoService) *MailService {
	return &MailService{
		db:                easyDB.GetPlayerDB(),
		cache:             &MailCache{cache: make(map[int64]*MailManager)},
		idGenerator:       tool.NewIdGenerator(int64(nodeConfig.NodeId), int64(enum.ID_GENERATOR_MAIL)),
		sessionManager:    sessionManager,
		messageSender:     messageSender,
		unlockService:     unlockService,
		serverInfoService: serverInfoService,
	}
}

// Init 初始化邮件服务
func (s *MailService) Init() error {
	logger.InfoWithSprintf("[MailService] Initializing mail service")
	logger.InfoWithSprintf("[MailService] Mail service initialized successfully")
	return nil
}

// NotifyMailRefresh 通知邮件服务刷新指定用户的缓存（供其他系统在直插 mail 表后调用）。
// 仅向 Redis 集合写入 userId，邮件服务心跳会检测并从 DB 重载该用户邮件缓存。
func NotifyMailRefresh(userId int64) error {
	if dbService.RDB == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return dbService.RDB.SAdd(ctx, enum.REDIS_MAIL_REFRESH_USERS, strconv.FormatInt(userId, 10)).Err()
}

// NotifyServerMailRefresh 通知邮件服务：有新的全服邮件（供其他系统在直插 server_mail 表后调用）。
// 仅向 Redis 写 key，心跳会 DEL 并向当前节点在线玩家推送邮件红点。
func NotifyServerMailRefresh() error {
	if dbService.RDB == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return dbService.RDB.Set(ctx, enum.REDIS_MAIL_REFRESH_SERVER, "1", 0).Err()
}

// StartRefreshHeartbeat 启动邮件刷新心跳：周期检测 Redis 待刷新集合，重载对应用户的 DB 缓存并清除 key。
func (s *MailService) StartRefreshHeartbeat(interval time.Duration) {
	if dbService.RDB == nil {
		logger.InfoWithSprintf("[MailService] Redis not available, skip mail refresh heartbeat")
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.ErrorWithZapFields("[MailService] refresh heartbeat panic", zap.Any("recover", r))
			}
		}()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			s.runRefreshFromRedis()
		}
	}()
	logger.InfoWithSprintf("[MailService] mail refresh heartbeat started, interval=%v", interval)
}

// runRefreshFromRedis 从 Redis 读取待刷新用户 ID / 全服邮件标记，重载 DB、清除 key，并可选推送在线玩家红点。
func (s *MailService) runRefreshFromRedis() {
	// 增加超时时间到 8 秒，与 Redis ReadTimeout(10s) 对齐，避免 context 过早超时
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	// 1）个人邮件：待刷新用户集合
	key := enum.REDIS_MAIL_REFRESH_USERS
	userIds, err := dbService.RDB.SMembers(ctx, key).Result()
	if err != nil {
		// 如果是超时错误，记录更详细的信息
		if err == context.DeadlineExceeded {
			logger.ErrorWithZapFields("[MailService] SMembers mail refresh key timeout (context deadline exceeded)",
				zap.String("key", key), zap.Error(err))
		} else {
			logger.ErrorWithZapFields("[MailService] SMembers mail refresh key failed",
				zap.String("key", key), zap.Error(err))
		}
		// 超时或错误时直接返回，避免继续执行可能失败的操作
		return
	}

	// 处理用户刷新列表（限制单次处理数量，避免阻塞）
	maxProcessPerCycle := 100
	processed := 0
	for _, idStr := range userIds {
		if processed >= maxProcessPerCycle {
			logger.Warn("[MailService] too many refresh users, will process in next cycle",
				zap.Int("processed", processed), zap.Int("total", len(userIds)))
			break
		}

		userId, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			logger.ErrorWithZapFields("[MailService] invalid refresh user id", zap.String("id", idStr), zap.Error(err))
			// 使用新的 context 删除无效 ID，避免使用已超时的 context
			delCtx, delCancel := context.WithTimeout(context.Background(), 2*time.Second)
			_ = dbService.RDB.SRem(delCtx, key, idStr).Err()
			delCancel()
			continue
		}

		if _, err := s.reloadMailManager(userId); err != nil {
			logger.ErrorWithZapFields("[MailService] reload mail manager failed", zap.Int64("user_id", userId), zap.Error(err))
			// 重载失败时不删除 key，下次重试
			continue
		}

		// 重载成功后删除 key
		delCtx, delCancel := context.WithTimeout(context.Background(), 2*time.Second)
		if err := dbService.RDB.SRem(delCtx, key, idStr).Err(); err != nil {
			logger.ErrorWithZapFields("[MailService] SRem mail refresh user failed",
				zap.Int64("user_id", userId), zap.Error(err))
		}
		delCancel()
		processed++
	}

	// 2）全服邮件：若有更新标记，清除并给当前节点在线玩家推红点
	serverKey := enum.REDIS_MAIL_REFRESH_SERVER
	serverCtx, serverCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer serverCancel()
	n, err := dbService.RDB.Del(serverCtx, serverKey).Result()
	if err != nil {
		if err == context.DeadlineExceeded {
			logger.ErrorWithZapFields("[MailService] Del server mail refresh key timeout", zap.Error(err))
		} else {
			logger.ErrorWithZapFields("[MailService] Del server mail refresh key failed", zap.Error(err))
		}
		return
	}
	if n > 0 {
		manager := s.sessionManager.(*logicSessionManager.GameSessionManager)
		if manager != nil {
			manager.RangeOnlinePlayers(func(p *model.PlayerModel) {
				s.messageSender.SendMessage(p, pb.MESSAGE_ID_PUSH_MAIL_NEW, &pb.PushMailNew{UnreadCount: 1})
			})
		}
	}
}

// getUserLock 获取用户级锁（懒加载）
func (s *MailService) getUserLock(userId int64) *sync.Mutex {
	if v, ok := s.userLocks.Load(userId); ok {
		return v.(*sync.Mutex)
	}
	m := &sync.Mutex{}
	actual, _ := s.userLocks.LoadOrStore(userId, m)
	return actual.(*sync.Mutex)
}

// loadMailsFromDB 从数据库加载邮件数据
func (s *MailService) loadMailsFromDB(userId int64) (*MailManager, error) {
	manager := NewMailManager(userId)

	// 加载邮件数据
	var entities []MailEntity
	result := s.db.Where("user_id = ? AND deleted_at IS NULL", userId).Find(&entities)
	if result.Error != nil {
		return nil, result.Error
	}

	for _, entity := range entities {
		mail, err := MailFromEntity(&entity)
		if err != nil {
			logger.ErrorWithZapFields("[MailService] Failed to convert entity to mail", zap.Error(err), zap.Int64("mail_id", entity.MailID))
			continue
		}
		manager.Mails[mail.MailID] = mail
	}

	return manager, nil
}

// LoadMailManager 加载玩家邮件管理器
func (s *MailService) LoadMailManager(userId int64) (*MailManager, error) {
	s.cache.mu.RLock()
	if manager, exists := s.cache.cache[userId]; exists {
		s.cache.mu.RUnlock()
		return manager, nil
	}
	s.cache.mu.RUnlock()

	// 从数据库加载
	manager, err := s.loadMailsFromDB(userId)
	if err != nil {
		return nil, err
	}

	// 缓存到内存
	s.cache.mu.Lock()
	s.cache.cache[userId] = manager
	s.cache.mu.Unlock()

	return manager, nil
}

// reloadMailManager 强制从数据库重载邮件数据并覆盖缓存
// 说明：当外部（GM/脚本/手工SQL）直接修改 mail 表时，内存缓存不会自动感知，需要重载。
func (s *MailService) reloadMailManager(userId int64) (*MailManager, error) {
	manager, err := s.loadMailsFromDB(userId)
	if err != nil {
		return nil, err
	}
	s.cache.mu.Lock()
	s.cache.cache[userId] = manager
	s.cache.mu.Unlock()
	return manager, nil
}

// SaveMailManager 保存邮件管理器数据
func (s *MailService) SaveMailManager(manager *MailManager) error {
	if !manager.Changed {
		return nil
	}

	tx := s.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	// 保存变更的邮件（跳过待删除的邮件，避免 UPDATE 覆盖状态后再 DELETE 导致竞态）
	for mailID, mail := range manager.ChangedMails {
		if manager.DeletedMails[mailID] {
			continue
		}
		entity := MailToEntity(mail)
		if manager.NewMailIDs[mailID] {
			// 新建邮件：使用 Create 插入，Save 在主键非零时会执行 UPDATE 导致 0 行影响
			if err := tx.Select("*").Create(entity).Error; err != nil {
				tx.Rollback()
				return err
			}
			logger.InfoWithZapFields("[MailService] Created mail", zap.Int64("mail_id", mailID), zap.Int64("user_id", manager.UserID))
		} else {
			// 修改邮件：记录已存在，Save 会执行 UPDATE
			if err := tx.Select("*").Save(entity).Error; err != nil {
				tx.Rollback()
				return err
			}
			logger.InfoWithZapFields("[MailService] Updated mail", zap.Int64("mail_id", mailID), zap.Int64("user_id", manager.UserID))
		}
	}

	// 软删除
	for mailID := range manager.DeletedMails {
		if err := tx.Where("mail_id = ? AND user_id = ?", mailID, manager.UserID).Delete(&MailEntity{}).Error; err != nil {
			tx.Rollback()
			return err
		}
		logger.InfoWithZapFields("[MailService] Deleted mail", zap.Int64("mail_id", mailID), zap.Int64("user_id", manager.UserID))
	}

	if err := tx.Commit().Error; err != nil {
		return err
	}

	manager.ClearChanged()
	return nil
}

// checkAndCreateServerMails 检查并创建全服邮件
func (s *MailService) checkAndCreateServerMails(userId int64) error {
	p := s.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return errors.New("player not found")
	}
	player := p.(*model.PlayerModel)
	if player == nil {
		return errors.New("player not found")
	}
	registerTimeLowerBound := getRegisterTimeLowerBoundInSeconds(player.User.GetRegisterTime())

	// 查询符合条件的全服邮件
	var serverMailEntities []ServerMailEntity
	query := s.db.Where("status = ? AND expire_time > ? AND alliance_id = 0", ServerMailStatusSent, time.Now().Unix())
	query = query.Where("send_time >= ?", registerTimeLowerBound)

	if player.GetUserServerId() > 0 {
		query = query.Where("(server_id = 0 OR server_id = ?)", player.GetUserServerId())
	}

	if err := query.Find(&serverMailEntities).Error; err != nil {
		return err
	}

	// 批量查询已创建的server_mail_id
	var createdServerMailIDs []int64
	// todo 优化 全服邮件映射表
	s.db.Unscoped().Model(&MailEntity{}).
		Where("user_id = ? AND server_mail_id > 0", userId).
		Pluck("server_mail_id", &createdServerMailIDs)
	createdMap := make(map[int64]bool)
	for _, id := range createdServerMailIDs {
		createdMap[id] = true
	}

	// 创建未创建的全服邮件副本
	for _, serverMailEntity := range serverMailEntities {
		if createdMap[serverMailEntity.ServerMailID] {
			continue
		}

		// 检查玩家是否符合条件
		if !s.checkServerMailCondition(&serverMailEntity, player) {
			continue
		}

		// 创建邮件副本（DB 已改为直接存 items）
		items, _ := serverMailEntity.GetItems()

		mailID := s.idGenerator.NextId()
		mail := &Mail{
			MailID:       mailID,
			UserID:       userId,
			MailType:     serverMailEntity.MailType,
			Title:        serverMailEntity.Title,
			Content:      serverMailEntity.Content,
			SenderAvatar: serverMailEntity.SenderAvatar,
			ServerMailID: serverMailEntity.ServerMailID,
			TemplateID:   serverMailEntity.TemplateID,
			Status:       MailStatusUnread,
			IsConvenient: serverMailEntity.IsConvenient, // 从数据库读取的值（TINYINT(1): 0=false, 1=true）
			Items:        items,
			SendTime:     serverMailEntity.SendTime,
			ExpireTime:   serverMailEntity.ExpireTime,
		}

		//// 调试日志：确认从数据库读取的 IsConvenient 值
		//logger.Info("[MailService] Creating server mail copy",
		//	zap.Int64("server_mail_id", serverMailEntity.ServerMailID),
		//	zap.Bool("is_convenient_from_db", serverMailEntity.IsConvenient),
		//	zap.Int64("user_id", userId))

		entity := MailToEntity(mail)
		// 使用 Select 明确指定要保存的字段，确保 IsConvenient=false 也能正确保存
		if err := s.db.Select("*").Create(entity).Error; err != nil {
			logger.ErrorWithZapFields("[MailService] Failed to create server mail copy", zap.Error(err), zap.Int64("server_mail_id", serverMailEntity.ServerMailID))
			continue
		}

		logger.InfoWithZapFields("[MailService] Created server mail copy", zap.Int64("mail_id", mailID), zap.Int64("server_mail_id", serverMailEntity.ServerMailID), zap.Int64("user_id", userId))
	}

	return nil
}

// checkAndCreateAllianceMails 检查并创建联盟邮件
func (s *MailService) checkAndCreateAllianceMails(userId int64) error {
	p := s.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return errors.New("player not found")
	}
	player := p.(*model.PlayerModel)
	if player == nil {
		return errors.New("player not found")
	}

	allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(userId)
	if allianceInfo == nil || allianceInfo.AllianceId <= 0 {
		return nil
	}
	allianceID := allianceInfo.AllianceId

	// 查询符合条件的联盟邮件（复用 server_mail 表）
	var allianceMailEntities []ServerMailEntity
	if err := s.db.Where("alliance_id = ? AND status = ? AND expire_time > ? AND send_time >= ?",
		allianceID, ServerMailStatusSent, time.Now().Unix(), allianceInfo.JoinTime).
		Find(&allianceMailEntities).Error; err != nil {
		return err
	}

	// 批量查询已创建的server_mail_id
	var createdServerMailIDs []int64
	s.db.Unscoped().Model(&MailEntity{}).
		Where("user_id = ? AND server_mail_id > 0", userId).
		Pluck("server_mail_id", &createdServerMailIDs)
	createdMap := make(map[int64]bool, len(createdServerMailIDs))
	for _, id := range createdServerMailIDs {
		createdMap[id] = true
	}

	// 创建未创建的联盟邮件副本
	for _, allianceMailEntity := range allianceMailEntities {
		if createdMap[allianceMailEntity.ServerMailID] {
			continue
		}

		items, _ := allianceMailEntity.GetItems()
		mailID := s.idGenerator.NextId()
		playerMail := &Mail{
			MailID:       mailID,
			UserID:       userId,
			MailType:     allianceMailEntity.MailType,
			Title:        allianceMailEntity.Title,
			Content:      allianceMailEntity.Content,
			SenderAvatar: allianceMailEntity.SenderAvatar,
			ServerMailID: allianceMailEntity.ServerMailID,
			TemplateID:   allianceMailEntity.TemplateID,
			Status:       MailStatusUnread,
			IsConvenient: allianceMailEntity.IsConvenient,
			Items:        items,
			SendTime:     allianceMailEntity.SendTime,
			ExpireTime:   allianceMailEntity.ExpireTime,
		}

		entity := MailToEntity(playerMail)
		if err := s.db.Select("*").Create(entity).Error; err != nil {
			logger.ErrorWithZapFields("[MailService] Failed to create alliance mail copy",
				zap.Error(err),
				zap.Int64("server_mail_id", allianceMailEntity.ServerMailID),
				zap.Int64("alliance_id", allianceID),
				zap.Int64("user_id", userId))
			continue
		}

		logger.InfoWithZapFields("[MailService] Created alliance mail copy",
			zap.Int64("mail_id", mailID),
			zap.Int64("server_mail_id", allianceMailEntity.ServerMailID),
			zap.Int64("alliance_id", allianceID),
			zap.Int64("user_id", userId))
	}

	return nil
}

// checkServerMailCondition 检查玩家是否符合全服邮件条件
func (s *MailService) checkServerMailCondition(serverMail *ServerMailEntity, player logicCommon.PlayerInterface) bool {
	// 获取解锁ID列表
	unlockList, err := serverMail.GetUnlockList()
	if err != nil {
		logger.ErrorWithZapFields("[MailService] Failed to get unlock list", zap.Error(err))
		return false
	}

	// 如果没有解锁条件，则所有玩家都可以接收
	if len(unlockList) == 0 {
		return true
	}

	// 检查玩家是否满足所有解锁条件（所有unlockID都需要满足）
	// 使用类型断言获取 PlayerModel
	playerModel, ok := player.(*model.PlayerModel)
	if !ok {
		logger.ErrorWithZapFields("[MailService] Failed to convert player to PlayerModel")
		return false
	}

	for _, unlockID := range unlockList {
		if !s.unlockService.CheckUnlock(unlockID, playerModel) {
			return false
		}
	}

	return true
}

// ensureMailCapacity 废弃 确保邮件容量不超过上限
func (s *MailService) ensureMailCapacity(manager *MailManager) error {
	if len(manager.Mails) < MaxMailCount {
		return nil
	}

	// 按优先级删除：已领取 > 已读 > 未读，时间从旧到新
	var toDelete []int64
	for mailID, mail := range manager.Mails {
		if mail.Status == MailStatusClaimed {
			toDelete = append(toDelete, mailID)
		}
	}

	// 如果已领取邮件不够，删除已读邮件
	if len(toDelete) < len(manager.Mails)-MaxMailCount+1 {
		for mailID, mail := range manager.Mails {
			if mail.Status == MailStatusRead {
				toDelete = append(toDelete, mailID)
				if len(toDelete) >= len(manager.Mails)-MaxMailCount+1 {
					break
				}
			}
		}
	}

	// 删除邮件
	for _, mailID := range toDelete {
		manager.RemoveMail(mailID)
	}

	return nil
}

// SendMail 发送邮件（要求玩家在线，否则返回 player not found）
func (s *MailService) SendMail(userId int64, mail *Mail) (int64, error) {
	lock := s.getUserLock(userId)
	lock.Lock()
	defer lock.Unlock()

	manager, err := s.LoadMailManager(userId)
	if err != nil {
		return 0, err
	}
	if mail.MailID == 0 {
		mail.MailID = s.idGenerator.NextId()
	}
	if mail.SendTime == 0 {
		mail.SendTime = time.Now().Unix()
	}
	manager.AddMail(mail)
	if err := s.SaveMailManager(manager); err != nil {
		return 0, err
	}
	p := s.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return 0, errors.New("player not found")
	}
	if player, ok := p.(*model.PlayerModel); ok && player != nil {
		s.messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_MAIL_NEW, &pb.PushMailNew{UnreadCount: 1})
	}
	return mail.MailID, nil
}

// SendMailToUserId 向指定玩家发邮件（不要求在线；离线也写入成功，在线则推送红点）
func (s *MailService) SendMailToUserId(userId int64, mail *Mail) (int64, error) {
	lock := s.getUserLock(userId)
	lock.Lock()
	defer lock.Unlock()

	manager, err := s.LoadMailManager(userId)
	if err != nil {
		return 0, err
	}
	if mail.MailID == 0 {
		mail.MailID = s.idGenerator.NextId()
	}
	if mail.SendTime == 0 {
		mail.SendTime = time.Now().Unix()
	}
	manager.AddMail(mail)
	if err := s.SaveMailManager(manager); err != nil {
		return 0, err
	}
	if p := s.sessionManager.GetPlayerBasicInfoByUserId(userId); p != nil {
		if player, ok := p.(*model.PlayerModel); ok && player != nil {
			s.messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_MAIL_NEW, &pb.PushMailNew{UnreadCount: 1})
		}
	}
	return mail.MailID, nil
}

// hasMailByTemplate 检查玩家是否已经收到过指定模板ID的邮件（用于 mailSender 首次触发判定）
// 使用 Unscoped 包含软删除记录：玩家领取后一键删除的邮件仍视为“已收到过”，避免 GetMailList 时重复发送
func (s *MailService) hasMailByTemplate(userId int64, templateID int32) (bool, error) {
	if templateID == 0 {
		return false, nil
	}
	var count int64
	err := s.db.Unscoped().Model(&MailEntity{}).
		Where("user_id = ? AND template_id = ?", userId, templateID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// SendMailByTemplateID 根据邮件模板ID（mailContent 表 id）向指定玩家发送一封邮件。
// 模板中 mailTitle、mailWords、sendName 为文本ID，写入 Mail 的 Title/Content/SenderName 为 ID 的字符串形式，供客户端按 ID 查多语言表。
func (s *MailService) SendMailByTemplateID(userId int64, templateID int32) (int64, error) {
	cfg := gameConfig.GetMailContentCfg(templateID)
	if cfg == nil {
		return 0, fmt.Errorf("mail template not found: %d", templateID)
	}
	mail := buildMailFromTemplate(userId, 0, cfg)
	return s.SendMailToUserId(userId, mail)
}

func (s *MailService) SendRewardMailByTemplateID(userId int64, templateID int32, reward []*gameConfig.ItemConfig, titleParams []string, contentParams []string) (int64, error) {
	cfg := gameConfig.GetMailContentCfg(templateID)
	if cfg == nil {
		return 0, fmt.Errorf("mail template not found: %d", templateID)
	}
	mail := buildMailFromTemplate(userId, 0, cfg)
	mail.TitleParams = titleParams
	mail.ContentParams = contentParams
	// 使用传入的 reward 覆盖模板奖励
	items := make([]*logicCommon.MailAttachmentItem, 0, len(reward))
	for _, it := range reward {
		if it.ID <= 0 || it.Num <= 0 {
			continue
		}
		items = append(items, &logicCommon.MailAttachmentItem{
			Type: AttachmentItemTypeItem,
			ID:   it.ID,
			Num:  int32(it.Num),
		})
	}
	mail.Items = items
	return s.SendMailToUserId(userId, mail)
}

func (s *MailService) SendRewardMailByTemplateIDAndTime(userId int64, templateID int32, timestamp int64, reward []*gameConfig.ItemConfig, titleParams []string, contentParams []string) (int64, error) {
	cfg := gameConfig.GetMailContentCfg(templateID)
	if cfg == nil {
		return 0, fmt.Errorf("mail template not found: %d", templateID)
	}
	mail := buildMailFromTemplate(userId, timestamp, cfg)
	mail.TitleParams = titleParams
	mail.ContentParams = contentParams
	// 使用传入的 reward 覆盖模板奖励
	items := make([]*logicCommon.MailAttachmentItem, 0, len(reward))
	for _, it := range reward {
		if it.ID <= 0 || it.Num <= 0 {
			continue
		}
		items = append(items, &logicCommon.MailAttachmentItem{
			Type: AttachmentItemTypeItem,
			ID:   it.ID,
			Num:  int32(it.Num),
		})
	}
	mail.Items = items
	return s.SendMailToUserId(userId, mail)
}

// buildMailFromTemplate 根据邮件模板配置构建 Mail（不分配 MailID，由 SendMailToUserId 内分配）
func buildMailFromTemplate(userId int64, timestamp int64, cfg *gameConfig.MailContentCfg) *Mail {
	expireTime := int64(0)
	if cfg.MailExpTime > 0 {
		expireTime = time.Now().Unix() + int64(cfg.MailExpTime)*3600
	}
	items := make([]*MailAttachmentItem, 0, len(cfg.Item))
	for _, it := range cfg.Item {
		if it == nil || it.Num <= 0 {
			continue
		}
		items = append(items, &MailAttachmentItem{
			Type: AttachmentItemTypeItem,
			ID:   it.ID,
			Num:  int32(it.Num),
		})
	}
	if timestamp == 0 {
		timestamp = time.Now().Unix()
	}
	return &Mail{
		UserID:       userId,
		MailType:     cfg.MailType,
		Title:        strconv.FormatInt(int64(cfg.MailTitle), 10),
		Content:      strconv.FormatInt(int64(cfg.MailWords), 10),
		SenderID:     0,
		SenderName:   strconv.FormatInt(int64(cfg.SendName), 10),
		TemplateID:   cfg.ID,
		Status:       MailStatusUnread,
		IsConvenient: cfg.IsConvenient,
		Items:        items,
		ExpireTime:   expireTime,
		SendTime:     timestamp,
	}
}

// checkMailSenderForPlayer 根据 mailSender 配置检查并发送模板邮件
// 规则：BornDay/ServerDay/Date 任一达成，且 unlock 全部满足、unlockStop 全部不满足，且首次发送。
func (s *MailService) checkMailSenderForPlayer(player *model.PlayerModel) error {
	if player == nil {
		return nil
	}
	userId := player.GetUserId()
	now := tool.UnixNowMilli()

	cfgMap := gameConfig.GetAllMailSenderCfg()
	if len(cfgMap) == 0 {
		return nil
	}

	for _, cfg := range cfgMap {
		if cfg == nil {
			continue
		}
		// 时间触发条件：BornDay 或 ServerDay 或 Date 至少满足一个
		if !s.matchMailSenderTrigger(cfg, player, now) {
			continue
		}
		// 解锁条件：全部满足
		pass := true
		for _, unlockId := range cfg.Unlock {
			if unlockId == 0 {
				continue
			}
			if !s.unlockService.CheckUnlock(unlockId, player) {
				pass = false
				break
			}
		}
		if !pass {
			continue
		}
		// 结束条件：任一成立则不发送
		stop := false
		for _, stopId := range cfg.UnlockStop {
			if stopId == 0 {
				continue
			}
			if s.unlockService.CheckUnlock(stopId, player) {
				stop = true
				break
			}
		}
		if stop {
			continue
		}

		// 模板不存在则跳过（mailContent 未配置该 mailId）
		if gameConfig.GetMailContentCfg(cfg.MailId) == nil {
			logger.Warn("[MailService] mailSender skip: mailContent template not found",
				zap.Int32("mail_sender_id", cfg.Id), zap.Int32("mail_id", cfg.MailId))
			continue
		}
		// 首次达成：如果已存在该模板ID的邮件，则认为已经触发过
		has, err := s.hasMailByTemplate(userId, cfg.MailId)
		if err != nil {
			logger.ErrorWithZapFields("[MailService] hasMailByTemplate failed",
				zap.Error(err), zap.Int64("user_id", userId), zap.Int32("template_id", cfg.MailId))
			continue
		}
		if has {
			continue
		}

		if _, err := s.SendMailByTemplateID(userId, cfg.MailId); err != nil {
			logger.ErrorWithZapFields("[MailService] SendMailByTemplateID from mailSender failed",
				zap.Error(err), zap.Int64("user_id", userId), zap.Int32("template_id", cfg.MailId), zap.Int32("mail_sender_id", cfg.Id))
			continue
		}
	}

	return nil
}

// matchMailSenderTrigger 判断时间触发条件是否满足（BornDay/ServerDay/Date 任一即可）
func (s *MailService) matchMailSenderTrigger(cfg *gameConfig.MailSenderCfg, player *model.PlayerModel, now int64) bool {
	hasTrigger := cfg.BornDay > 0 || cfg.ServerDay > 0 || cfg.Date != ""
	if !hasTrigger {
		return false
	}
	// BornDay：注册第几天（1=首日，N=第N天）。GetNatureDayDistance(now, register)：同天=0，次日=1
	// 第1天=首日：distance>=0；第5天：distance>=4。故条件为 distance >= bornDay-1
	if cfg.BornDay > 0 {
		regDays := tool.GetNatureDayDistance(now, player.User.GetRegisterTime())
		if regDays >= cfg.BornDay-1 {
			return true
		}
	}
	// ServerDay：开服第几天（使用 tool.GetNatureDayDistance(now, serverInfo.GetServerOpenTime()) >= serverDay-1）
	if cfg.ServerDay > 0 && s.serverInfoService != nil {
		serverInfo := s.serverInfoService.GetServerInfo(player.GetUserServerId())
		if serverInfo != nil && tool.GetNatureDayDistance(now, serverInfo.GetServerOpenTime()) >= cfg.ServerDay-1 {
			return true
		}
	}
	// Date：具体时间，当前时间已过则触发
	if cfg.Date != "" {
		timeInterval, err := tool.ParseTime2TimeStamp(cfg.Date)
		timeInterval2, err := tool.ParseTime2TimeStamp(cfg.Date)
		if err == nil && now >= timeInterval && now <= timeInterval2 {
			return true
		}
	}
	return false
}

// GetMailList 获取邮件列表
func (s *MailService) GetMailList(userId int64, mailType int32, status int32, page int32, pageSize int32) ([]*Mail, int32, error) {
	// 在加锁前执行 checkMailSenderForPlayer，避免死锁（其内部会调用 SendMailToUserId 再次获取同一把锁）
	if p := s.sessionManager.GetPlayerBasicInfoByUserId(userId); p != nil {
		if player, ok := p.(*model.PlayerModel); ok && player != nil {
			if err := s.checkMailSenderForPlayer(player); err != nil {
				logger.ErrorWithZapFields("[MailService] checkMailSenderForPlayer in GetMailList failed",
					zap.Error(err), zap.Int64("user_id", userId))
			}
		}
	}

	lock := s.getUserLock(userId)
	lock.Lock()
	defer lock.Unlock()

	// 检查并创建全服邮件
	if err := s.checkAndCreateServerMails(userId); err != nil {
		logger.ErrorWithZapFields("[MailService] Failed to check and create server mails", zap.Error(err), zap.Int64("user_id", userId))
	}
	// 检查并创建联盟邮件
	if err := s.checkAndCreateAllianceMails(userId); err != nil {
		logger.ErrorWithZapFields("[MailService] Failed to check and create alliance mails", zap.Error(err), zap.Int64("user_id", userId))
	}

	// 强制从数据库重载（避免仅返回内存缓存导致“DB已有但列表为空”）
	manager, err := s.reloadMailManager(userId)
	if err != nil {
		return nil, 0, err
	}

	// 筛选邮件
	var filteredMails []*Mail
	for _, mail := range manager.Mails {
		// 过滤邮件类型
		if mailType > 0 && mail.MailType != mailType {
			continue
		}
		// 过滤状态
		if status > 0 && mail.Status != status {
			continue
		}
		filteredMails = append(filteredMails, mail)
	}

	// 排序：先未读、再已读（以及其他状态）；同组内按发送时间倒序
	sort.SliceStable(filteredMails, func(i, j int) bool {
		mi, mj := filteredMails[i], filteredMails[j]
		if mi == nil || mj == nil {
			return mi != nil
		}
		iUnread := mi.Status == MailStatusUnread
		jUnread := mj.Status == MailStatusUnread
		if iUnread != jUnread {
			return iUnread
		}
		if mi.SendTime != mj.SendTime {
			return mi.SendTime > mj.SendTime
		}
		return mi.MailID > mj.MailID
	})

	// 计算总数
	total := int32(len(filteredMails))

	// 分页
	start := (page - 1) * pageSize
	end := start + pageSize
	if start >= int32(len(filteredMails)) {
		return []*Mail{}, total, nil
	}
	if end > int32(len(filteredMails)) {
		end = int32(len(filteredMails))
	}

	return filteredMails[start:end], total, nil
}

// GetMailDetail 获取邮件详情
func (s *MailService) GetMailDetail(userId int64, mailId int64) (*Mail, error) {
	lock := s.getUserLock(userId)
	lock.Lock()
	defer lock.Unlock()

	manager, err := s.LoadMailManager(userId)
	if err != nil {
		return nil, err
	}

	mail := manager.GetMail(mailId)
	if mail == nil {
		return nil, errors.New("mail not found")
	}

	return mail, nil
}

// ReadMail 阅读邮件
func (s *MailService) ReadMail(userId int64, mailId int64) error {
	lock := s.getUserLock(userId)
	lock.Lock()
	defer lock.Unlock()

	manager, err := s.LoadMailManager(userId)
	if err != nil {
		return err
	}

	mail := manager.GetMail(mailId)
	if mail == nil {
		return errors.New("mail not found")
	}

	// 如果已经是已读或已领取状态，不需要更新
	if mail.Status == MailStatusRead || mail.Status == MailStatusClaimed {
		return nil
	}

	// 更新状态为已读
	mail.Status = MailStatusRead
	mail.ReadTime = time.Now().Unix()
	manager.ChangedMails[mailId] = mail
	manager.Changed = true

	// 保存到数据库
	return s.SaveMailManager(manager)
}

// ClaimMailAttachment 领取邮件附件
func (s *MailService) ClaimMailAttachment(userId int64, mailId int64) ([]*MailAttachmentItem, error) {
	lock := s.getUserLock(userId)
	lock.Lock()
	defer lock.Unlock()

	manager, err := s.LoadMailManager(userId)
	if err != nil {
		return nil, err
	}

	mail := manager.GetMail(mailId)
	if mail == nil {
		return nil, errors.New("mail not found")
	}

	// 检查邮件是否过期
	now := time.Now().Unix()
	if mail.ExpireTime > 0 && now > mail.ExpireTime {
		return nil, errors.New("mail expired")
	}

	if len(mail.Items) == 0 {
		return nil, errors.New("no attachment items")
	}
	if mail.Status == MailStatusClaimed {
		return nil, errors.New("already claimed")
	}

	// 获取玩家对象
	p := s.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return nil, errors.New("player not found")
	}
	player := p.(*model.PlayerModel)
	if player == nil {
		return nil, errors.New("player not found")
	}

	// 领取附件（一次性组装 items，调用 AddItemsCommon 批量发放）
	items := make([]*gameConfig.ItemConfig, 0, len(mail.Items))
	for _, it := range mail.Items {
		if it == nil || it.Num <= 0 || it.ID <= 0 {
			continue
		}
		// 全放
		items = append(items, &gameConfig.ItemConfig{ID: it.ID, Num: int64(it.Num)})

		//switch it.Type {
		//case AttachmentItemTypeItem, AttachmentItemTypeCurrency:
		//	items = append(items, &gameConfig.ItemConfig{ID: it.ID, Num: int64(it.Num)})
		//case AttachmentItemTypeExp:
		//	// TODO: 实现经验发放；目前项目里经验不走背包，这里先拒绝领取，避免“标记已领但没发放”的数据问题
		//	logger.ErrorWithZapFields("[MailService] Exp attachment not implemented", zap.Int64("mail_id", mailId))
		//	return nil, nil, errors.New("exp attachment not implemented")
		//default:
		//	logger.ErrorWithZapFields("[MailService] Unknown attachment item type", zap.Int64("mail_id", mailId), zap.Int32("type", it.Type))
		//	return nil, nil, errors.New("unknown attachment item type")
		//}
	}
	if len(items) == 0 {
		return nil, errors.New("attachment items is empty")
	}

	if err := itemService.AddItems(player, items, enum.ITEM_CHANGE_REASON_MAIL_ATTACHMENT); err != nil {
		logger.ErrorWithZapFields("[MailService] Failed to claim attachment", zap.Error(err), zap.Int64("mail_id", mailId))
		return nil, err
	}

	// 发放成功才标记已领取（状态/时间在邮件上）
	// 说明：当前业务没有“部分领取/多附件”场景，因此这里不需要再做二次判断
	mail.Status = MailStatusClaimed
	mail.ClaimTime = time.Now().Unix()
	claimedItems := mail.Items

	manager.ChangedMails[mailId] = mail
	manager.Changed = true

	// 保存到数据库
	if err := s.SaveMailManager(manager); err != nil {
		return nil, err
	}

	return claimedItems, nil
}

// ClaimAllMailAttachments 一键领取所有附件
// 所有附件合并后一次性添加，只触发一次物品推送
func (s *MailService) ClaimAllMailAttachments(userId int64) (int32, error) {
	lock := s.getUserLock(userId)
	lock.Lock()
	defer lock.Unlock()

	manager, err := s.LoadMailManager(userId)
	if err != nil {
		return 0, err
	}

	// 获取玩家对象
	p := s.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return 0, errors.New("player not found")
	}
	player := p.(*model.PlayerModel)
	if player == nil {
		return 0, errors.New("player not found")
	}

	now := time.Now().Unix()

	// 1. 收集所有可领取邮件的附件，并记录待标记的邮件
	var allItems []*gameConfig.ItemConfig
	var mailsToClaim []*struct {
		mailID int64
		mail   *Mail
	}

	for mailID, mail := range manager.Mails {
		if mail == nil || !mail.IsConvenient {
			continue
		}
		if mail.ExpireTime > 0 && now > mail.ExpireTime {
			continue
		}
		if mail.Status == MailStatusClaimed {
			continue
		}
		if len(mail.Items) == 0 {
			continue
		}

		for _, it := range mail.Items {
			if it == nil || it.Num <= 0 || it.ID <= 0 {
				continue
			}
			allItems = append(allItems, &gameConfig.ItemConfig{ID: it.ID, Num: int64(it.Num)})
		}
		mailsToClaim = append(mailsToClaim, &struct {
			mailID int64
			mail   *Mail
		}{mailID, mail})
	}

	if len(allItems) == 0 {
		return 0, nil
	}

	// 2. 一次性添加所有物品（只触发一次物品推送）
	if err := itemService.AddItems(player, allItems, enum.ITEM_CHANGE_REASON_MAIL_ATTACHMENT); err != nil {
		logger.ErrorWithZapFields("[MailService] Failed to claim all mail attachments", zap.Error(err), zap.Int64("user_id", userId))
		return 0, err
	}

	// 3. 标记所有已发放的邮件为已领取
	claimedCount := int32(0)
	for _, entry := range mailsToClaim {
		entry.mail.Status = MailStatusClaimed
		entry.mail.ClaimTime = time.Now().Unix()
		manager.ChangedMails[entry.mailID] = entry.mail
		claimedCount += int32(len(entry.mail.Items))
	}
	manager.Changed = true

	if err := s.SaveMailManager(manager); err != nil {
		return 0, err
	}

	return claimedCount, nil
}

// DeleteMail 删除邮件
func (s *MailService) DeleteMail(userId int64, mailId int64) error {
	lock := s.getUserLock(userId)
	lock.Lock()
	defer lock.Unlock()

	manager, err := s.LoadMailManager(userId)
	if err != nil {
		return err
	}

	mail := manager.GetMail(mailId)
	if mail == nil {
		return errors.New("mail not found")
	}

	// 只能删除操作过的邮件
	if mail.Status == MailStatusUnread {
		return errors.New("only delete read and claimed mails")
	}

	manager.RemoveMail(mailId)

	return s.SaveMailManager(manager)
}

// DeleteClaimedMails 一键删除已领取and已读邮件
func (s *MailService) DeleteClaimedMails(userId int64) (int32, error) {
	lock := s.getUserLock(userId)
	lock.Lock()
	defer lock.Unlock()

	manager, err := s.LoadMailManager(userId)
	if err != nil {
		return 0, err
	}

	deletedCount := int32(0)
	for mailID, mail := range manager.Mails {
		if mail.Status == MailStatusClaimed || (len(mail.Items) == 0 && mail.Status == MailStatusRead) {
			manager.RemoveMail(mailID)
			deletedCount++
		}
	}

	if err := s.SaveMailManager(manager); err != nil {
		return 0, err
	}

	return deletedCount, nil
}

// SendServerMail 发送全服邮件
// 注意：这里使用interface{}类型避免循环依赖，实际调用时需要类型断言
func (s *MailService) SendServerMail(req interface{}) (int64, error) {
	// 类型断言：将interface{}转换为*pb.SendServerMailRequest
	// 注意：需要proto代码生成后，pb包中会有SendServerMailRequest类型
	pbReq, ok := req.(interface {
		GetMailType() int32
		GetTitle() string
		GetContent() string
		GetTemplateId() int32
		GetServerId() int32
		GetUnlockList() []int32
		GetAttachment() interface {
			GetItems() []interface {
				GetType() int32
				GetId() int32
				GetNum() int32
			}
		}
		GetExpireDays() int32
		GetCreatedBy() string
	})
	if !ok {
		return 0, errors.New("invalid request type for SendServerMail")
	}

	// 创建全服邮件实体
	serverMailID := s.idGenerator.NextId()
	now := time.Now().Unix()

	// 计算过期时间
	expireTime := int64(0)
	if pbReq.GetExpireDays() > 0 {
		expireTime = now + int64(pbReq.GetExpireDays())*24*3600
	}

	// 转换附件（业务约定：只有一个附件，直接取 attachment.items）
	var items []*MailAttachmentItem
	if att := pbReq.GetAttachment(); att != nil {
		pbItems := att.GetItems()
		items = make([]*MailAttachmentItem, 0, len(pbItems))
		for _, it := range pbItems {
			items = append(items, &MailAttachmentItem{
				Type: it.GetType(),
				ID:   it.GetId(),
				Num:  it.GetNum(),
			})
		}
	}

	// 获取解锁ID列表
	unlockList := pbReq.GetUnlockList()
	if unlockList == nil {
		unlockList = []int32{}
	}

	serverMail := &ServerMail{
		ServerMailID: serverMailID,
		MailType:     pbReq.GetMailType(),
		Title:        pbReq.GetTitle(),
		Content:      pbReq.GetContent(),
		TemplateID:   pbReq.GetTemplateId(),
		ServerID:     pbReq.GetServerId(),
		AllianceID:   0,
		UnlockList:   unlockList,
		Items:        items,
		SendTime:     now,
		ExpireTime:   expireTime,
		Status:       ServerMailStatusSent,
		CreatedBy:    pbReq.GetCreatedBy(),
	}

	entity := ServerMailToEntity(serverMail)
	if err := s.db.Create(entity).Error; err != nil {
		return 0, err
	}

	// 通知在线玩家（异步推送红点）
	go s.notifyOnlinePlayersForServerMail(serverMailID)

	return serverMailID, nil
}

// notifyOnlinePlayersForServerMail todo 通知在线玩家有新全服邮件
func (s *MailService) notifyOnlinePlayersForServerMail(serverMailID int64) {
	// 加载全服邮件信息
	var serverMailEntity ServerMailEntity
	if err := s.db.Where("server_mail_id = ?", serverMailID).First(&serverMailEntity).Error; err != nil {
		logger.ErrorWithZapFields("[MailService] Failed to load server mail", zap.Error(err), zap.Int64("server_mail_id", serverMailID))
		return
	}

	// 在玩家登录时检查并推送
	logger.InfoWithZapFields("[MailService] Server mail created, will notify players on login", zap.Int64("server_mail_id", serverMailID))
}

// GetServerMailList 获取全服邮件列表
func (s *MailService) GetServerMailList(page int32, pageSize int32) ([]*ServerMail, int32, error) {
	var entities []ServerMailEntity

	// 查询总数
	var total int64
	if err := s.db.Model(&ServerMailEntity{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := s.db.Order("created_at DESC").Offset(int(offset)).Limit(int(pageSize)).Find(&entities).Error; err != nil {
		return nil, 0, err
	}

	// 转换为业务模型
	serverMails := make([]*ServerMail, 0, len(entities))
	for _, entity := range entities {
		serverMail, err := ServerMailFromEntity(&entity)
		if err != nil {
			logger.ErrorWithZapFields("[MailService] Failed to convert entity to server mail", zap.Error(err))
			continue
		}
		serverMails = append(serverMails, serverMail)
	}

	return serverMails, int32(total), nil
}

// OnPlayerLogin 玩家登录时处理
func (s *MailService) OnPlayerLogin(userId int64) error {
	p := s.sessionManager.GetPlayerBasicInfoByUserId(userId)
	if p == nil {
		return errors.New("player not found")
	}
	player, ok := p.(*model.PlayerModel)
	if !ok || player == nil {
		return errors.New("player not found")
	}
	registerTimeLowerBound := getRegisterTimeLowerBoundInSeconds(player.User.GetRegisterTime())

	// 登录时检查 mailSender 触发邮件
	if err := s.checkMailSenderForPlayer(player); err != nil {
		logger.ErrorWithZapFields("[MailService] checkMailSenderForPlayer in OnPlayerLogin failed",
			zap.Error(err), zap.Int64("user_id", userId))
	}

	// 查询符合条件的全服邮件（只查询，不创建）
	var serverMailEntities []ServerMailEntity
	query := s.db.Where("status = ? AND expire_time > ? AND alliance_id = 0", ServerMailStatusSent, time.Now().Unix())
	query = query.Where("send_time >= ?", registerTimeLowerBound)

	if player.GetUserServerId() > 0 {
		query = query.Where("(server_id = 0 OR server_id = ?)", player.GetUserServerId())
	}

	if err := query.Find(&serverMailEntities).Error; err != nil {
		return err
	}

	// 检查符合条件的全服邮件数量
	unreadCount := 0
	for _, serverMail := range serverMailEntities {
		if s.checkServerMailCondition(&serverMail, player) {
			unreadCount++
		}
	}

	// 累加联盟邮件数量（复用 server_mail 表）
	allianceInfo := logicCommon.GetPlayerAllianceInfoFromRedis(userId)
	if allianceInfo != nil && allianceInfo.AllianceId > 0 {
		var allianceUnread int64
		if err := s.db.Model(&ServerMailEntity{}).
			Where("alliance_id = ? AND status = ? AND expire_time > ? AND send_time >= ?", allianceInfo.AllianceId, ServerMailStatusSent, time.Now().Unix(), allianceInfo.JoinTime).
			Count(&allianceUnread).Error; err != nil {
			return err
		}
		unreadCount += int(allianceUnread)
	}

	// 如果有符合条件的全服邮件，推送红点通知
	if unreadCount > 0 {
		s.messageSender.SendMessage(player, pb.MESSAGE_ID_PUSH_MAIL_NEW, &pb.PushMailNew{
			UnreadCount: int32(unreadCount),
		})
	}

	return nil
}

// getRegisterTimeLowerBoundInSeconds converts register_time(ms) to a conservative send_time(sec) lower bound.
func getRegisterTimeLowerBoundInSeconds(registerTimeMs int64) int64 {
	if registerTimeMs <= 0 {
		return 0
	}
	return (registerTimeMs + 999) / 1000
}

// CleanExpiredMails 清理过期邮件
func (s *MailService) CleanExpiredMails() error {
	now := time.Now().Unix()

	// 删除已领取的过期邮件
	result := s.db.Where("expire_time > 0 AND expire_time < ? AND status = ?", now, MailStatusClaimed).
		Delete(&MailEntity{})
	if result.Error != nil {
		return result.Error
	}

	logger.InfoWithZapFields("[MailService] Cleaned expired mails", zap.Int64("count", result.RowsAffected))
	return nil
}

// ProcessPendingServerMails 处理待发送的全服邮件
func (s *MailService) ProcessPendingServerMails() error {
	// 查询待发送的全服邮件
	var entities []ServerMailEntity
	now := time.Now().Unix()

	if err := s.db.Where("status = ? AND send_time <= ?", ServerMailStatusPending, now).Find(&entities).Error; err != nil {
		return err
	}

	// 标记为已发送
	for _, entity := range entities {
		entity.Status = ServerMailStatusSent
		if err := s.db.Save(&entity).Error; err != nil {
			logger.ErrorWithZapFields("[MailService] Failed to update server mail status", zap.Error(err), zap.Int64("server_mail_id", entity.ServerMailID))
			continue
		}

		// 通知在线玩家
		go s.notifyOnlinePlayersForServerMail(entity.ServerMailID)
	}

	return nil
}
