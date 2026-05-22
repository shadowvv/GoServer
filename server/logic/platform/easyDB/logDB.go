package easyDB

import (
	"fmt"
	"github.com/drop/GoServer/server/service/logger"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

var logDB *gorm.DB

// logTableCreated 记录已确认存在的表名，避免重复执行 CREATE TABLE
var logTableCreated sync.Map

// ---------------------------------------------------------
// 异步写队列
// ---------------------------------------------------------

const (
	logChanSize   = 10000                  // channel 缓冲容量
	logBatchSize  = 50                     // 攒够 50 条触发一次批量写
	logFlushDelay = 500 * time.Millisecond // 最长等待 500ms 强制刷新

	ITEM_LOG int32 = 1
	OPER_LOG int32 = 2
)

type logTask struct {
	entity  map[string]interface{}
	logType int32
}

// logChan 日志写入缓冲队列，SetLogDB 时初始化并启动 worker
var logChan chan *logTask

// SetLogDB 设置日志库并启动异步 worker
func SetLogDB(db *gorm.DB) {
	logDB = db
	logChan = make(chan *logTask, logChanSize)
	go logWorker()
}

// logWorker 后台消费协程：攒够 logBatchSize 条或超过 logFlushDelay 则批量写库
func logWorker() {
	batch := make([]*logTask, 0, logBatchSize)
	ticker := time.NewTicker(logFlushDelay)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := writeEntities(batch); err != nil {
			logger.ErrorBySprintf("[logDB] 批量写入失败: %v", err)
		}
		batch = batch[:0]
	}

	for {
		select {
		case entity, ok := <-logChan:
			if !ok {
				// channel 已关闭，drain 剩余并退出
				flush()
				return
			}
			batch = append(batch, entity)
			if len(batch) >= logBatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

// CloseLogDB 优雅关闭：等待 channel 消费完毕（服务退出时调用）
func CloseLogDB() {
	if logChan != nil {
		close(logChan)
	}
}

// ---------------------------------------------------------
// 对外接口（异步投递，主逻辑不阻塞）
// ---------------------------------------------------------

// LogCreatEntity 异步投递单条日志，立即返回
func LogCreatEntity(entity map[string]interface{}, logType int32) error {
	if len(entity) == 0 {
		return fmt.Errorf("entity is null")
	}
	if _, ok := entity["add_time"]; !ok {
		return fmt.Errorf("addTime not found")
	}
	if logChan == nil {
		return fmt.Errorf("logDB is not initialized")
	}
	logEntity := &logTask{entity, logType}
	select {
	case logChan <- logEntity:
	default:
		// 队列满，丢弃并记录完整内容（不阻塞主逻辑）
		logger.ErrorBySprintf("[logDB] 写入队列已满，丢弃日志: %+v", entity)
	}
	return nil
}

// LogCreatEntities 异步批量投递，立即返回
func LogCreatEntities(entities []map[string]interface{}, logType int32) error {
	if len(entities) == 0 {
		return nil
	}
	if logChan == nil {
		return fmt.Errorf("logDB is not initialized")
	}
	for _, entity := range entities {
		logEntity := &logTask{entity, logType}
		select {
		case logChan <- logEntity:
		default:
			logger.ErrorBySprintf("[logDB] 写入队列已满，丢弃日志: %+v", entity)
		}
	}
	return nil
}

// ---------------------------------------------------------
// 内部同步写（仅 worker 调用）
// ---------------------------------------------------------

// ensureLogTableExists 在表不存在时，依据模板表 log_template 自动建表（含主键和索引）
func ensureLogTableExists(tableName string, logType int32) error {
	if logDB == nil {
		return fmt.Errorf("logDB is not initialized, skip create table %s", tableName)
	}
	if _, ok := logTableCreated.Load(tableName); ok {
		return nil
	}
	var sql string
	if logType == OPER_LOG {
		sql = fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` LIKE `operation_log_template`", tableName)
	} else {
		sql = fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` LIKE `log_template`", tableName)
	}
	if err := logDB.Exec(sql).Error; err != nil {
		return fmt.Errorf("create log table %s failed: %w", tableName, err)
	}
	logTableCreated.Store(tableName, struct{}{})
	return nil
}

// writeEntities 同步批量写库（worker 内部调用）
func writeEntities(entities []*logTask) error {
	// 按月份分组
	groupedByMonth := make(map[string][]map[string]interface{})
	for _, entity := range entities {
		addTime, err := getTime(entity.entity["add_time"])
		if err != nil {
			continue
		}
		tableName := generateLogTableName(addTime, entity.logType)
		if groupedByMonth[tableName] == nil {
			groupedByMonth[tableName] = make([]map[string]interface{}, 0)
		}
		groupedByMonth[tableName] = append(groupedByMonth[tableName], entity.entity)
	}

	for tableName, batch := range groupedByMonth {
		logType := ITEM_LOG
		if strings.Contains(tableName, "operation_log") {
			logType = OPER_LOG
		}
		if err := ensureLogTableExists(tableName, logType); err != nil {
			logger.ErrorBySprintf("[logDB] 创建日志表 %s 失败: %v", tableName, err)
			continue
		}
		if err := logDB.Table(tableName).Create(batch).Error; err != nil {
			logger.ErrorBySprintf("[logDB] 插入表 %s 失败: %v", tableName, err)
		}
	}
	return nil
}

func generateLogTableName(t time.Time, logType int32) string {
	// 格式: log_202503 (25年3月)
	year := t.Year()
	month := int(t.Month())
	// 确保月份是两位数
	monthStr := fmt.Sprintf("%02d", month)

	if logType == OPER_LOG {
		return fmt.Sprintf("operation_log_%d%s", year, monthStr)
	}
	return fmt.Sprintf("log_%d%s", year, monthStr)
}

func getTime(needTime interface{}) (time.Time, error) {

	// 根据不同类型转换
	v, ok := needTime.(int64)
	if !ok {
		return time.Now(), fmt.Errorf("time not int64")
	}
	seconds := v / 1000
	nanoseconds := (v % 1000) * 1_000_000
	return time.Unix(seconds, nanoseconds), nil
}

func LogGetEntitiesByWhere(where map[string]interface{}) (entities []map[string]interface{}, err error) {

	tables, stMs, etMs, err := getTablesFromWhere(where)
	if err != nil {
		return nil, err
	}

	all := make([]map[string]interface{}, 0)

	for _, table := range tables {
		var results []map[string]interface{}
		if logDB == nil {
			return nil, fmt.Errorf("logDB is nil")
		}
		q := logDB.Table(table).Where(where)
		// 把时间范围作为 SQL 条件补回（getTablesFromWhere 已从 where 中删除 st/et）
		if stMs > 0 {
			q = q.Where("t >= ?", stMs)
		}
		if etMs > 0 {
			q = q.Where("t <= ?", etMs)
		}
		q.Find(&results)
		all = append(all, results...)
	}
	return all, nil

}

// getTablesFromWhere 从 where 中取出 st/et（毫秒时间戳）计算需要查询的月表列表。
// 返回：表名列表、st 毫秒值、et 毫秒值（用于调用方补 SQL 时间范围条件）。
// 注意：会对传入的 where 做浅拷贝，不修改原始 map。
func getTablesFromWhere(where map[string]interface{}) ([]string, int64, int64, error) {
	st := time.Now().AddDate(0, -3, 0) // 默认查3个月
	et := time.Now()
	var stMs, etMs int64

	// 浅拷贝，避免修改调用方的 where map
	clean := make(map[string]interface{}, len(where))
	for k, v := range where {
		clean[k] = v
	}

	// 有st参数就用
	if s, ok := clean["st"]; ok {
		if t, err := getTime(s); err == nil {
			st = t
			if v, ok2 := s.(int64); ok2 {
				stMs = v
			}
		} else {
			return nil, 0, 0, err
		}
		delete(clean, "st")
		delete(where, "st") // 同步删除原始 map，保持一致性
	}

	// 有et参数就用
	if e, ok := clean["et"]; ok {
		if t, err := getTime(e); err == nil {
			et = t
			if v, ok2 := e.(int64); ok2 {
				etMs = v
			}
		} else {
			return nil, 0, 0, err
		}
		delete(clean, "et")
		delete(where, "et") // 同步删除原始 map，保持一致性
	}

	// 生成表名
	var tables []string
	current := time.Date(st.Year(), st.Month(), 1, 0, 0, 0, 0, time.UTC)
	endMonth := time.Date(et.Year(), et.Month(), 1, 0, 0, 0, 0, time.UTC)

	for !current.After(endMonth) {
		tables = append(tables, fmt.Sprintf("log_%04d%02d", current.Year(), current.Month()))
		current = current.AddDate(0, 1, 0)
	}

	return tables, stMs, etMs, nil
}

// OperLogGetEntitiesByWhere 查询操作日志表（operation_log_YYYYMM），支持时间范围和 limit
func OperLogGetEntitiesByWhere(where map[string]interface{}, limit int) (entities []map[string]interface{}, err error) {
	if logDB == nil {
		return nil, fmt.Errorf("logDB is nil")
	}

	// 获取时间范围
	st := time.Now().AddDate(0, -3, 0) // 默认查3个月
	et := time.Now()
	var stMs, etMs int64

	if s, ok := where["st"]; ok {
		if t, err := getTime(s); err == nil {
			st = t
			if v, ok2 := s.(int64); ok2 {
				stMs = v
			}
		}
	}
	if e, ok := where["et"]; ok {
		if t, err := getTime(e); err == nil {
			et = t
			if v, ok2 := e.(int64); ok2 {
				etMs = v
			}
		}
	}

	// 生成 operation_log 表名列表（从最近月份倒序）
	var tables []string
	current := time.Date(et.Year(), et.Month(), 1, 0, 0, 0, 0, time.UTC)
	startMonth := time.Date(st.Year(), st.Month(), 1, 0, 0, 0, 0, time.UTC)

	for !current.Before(startMonth) {
		tables = append(tables, fmt.Sprintf("operation_log_%04d%02d", current.Year(), current.Month()))
		current = current.AddDate(0, -1, 0)
	}

	// 复制 where 条件，移除 st/et
	cleanWhere := make(map[string]interface{}, len(where))
	for k, v := range where {
		if k == "st" || k == "et" {
			continue
		}
		cleanWhere[k] = v
	}

	all := make([]map[string]interface{}, 0)

	for _, table := range tables {
		var results []map[string]interface{}
		q := logDB.Table(table).Where(cleanWhere)
		if stMs > 0 {
			q = q.Where("add_time >= ?", stMs)
		}
		if etMs > 0 {
			q = q.Where("add_time <= ?", etMs)
		}
		// 按时间倒序查询
		q = q.Order("add_time DESC")
		if limit > 0 && len(all) < limit {
			q = q.Limit(limit - len(all))
		}
		if err := q.Find(&results).Error; err != nil {
			continue // 表不存在则跳过
		}
		all = append(all, results...)
		if limit > 0 && len(all) >= limit {
			break
		}
	}

	return all, nil
}
