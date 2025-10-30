package platform

import (
	"gorm.io/gorm"
)

type DBPool struct {
	workers []*DBWorker
	num     int32
}

func NewDBPool(workerNum, workerTaskSize int32, db *gorm.DB) *DBPool {
	p := &DBPool{num: workerNum}
	for i := int32(0); i < workerNum; i++ {
		p.workers = append(p.workers, NewDBWorker(db, workerTaskSize))
	}
	return p
}

func (p *DBPool) AddWorker(workerId int32, worker *DBWorker) {
	for _, w := range p.workers {
		w.run()
	}
}

func (p *DBPool) AddWorkers(workerId, workerNum int32) {
	for _, w := range p.workers {
		w.run()
	}
}

func (p *DBPool) Submit(playerID int64, task DBTask) {
	idx := int(playerID % int64(p.num))
	p.workers[idx].tasks <- task
}
