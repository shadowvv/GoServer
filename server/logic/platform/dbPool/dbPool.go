package dbPool

import (
	"github.com/drop/GoServer/server/service/logger"
	"gorm.io/gorm"
)

type DBPool struct {
	workers []*DBWorker
	num     int32
}

// NewDBPool 创建数据库线程池
func NewDBPool(workerNum int32, queueSize int32, db *gorm.DB) *DBPool {
	if workerNum <= 0 {
		workerNum = 1
	}
	p := &DBPool{
		num:     workerNum,
		workers: make([]*DBWorker, workerNum),
	}
	for i := int32(0); i < workerNum; i++ {
		p.workers[i] = NewDBWorker(i, db, queueSize)
	}
	logger.InfoWithSprintf("[DBPool] started with %d workers", workerNum)
	return p
}

// Submit 根据 playerID 分配任务到指定 worker
func (p *DBPool) Submit(playerID int64, task DBTask) {
	idx := int(playerID % int64(p.num))
	p.workers[idx].Submit(task)
}

// Stop 关闭整个线程池
func (p *DBPool) Stop() {
	for _, w := range p.workers {
		w.Stop()
	}
}
