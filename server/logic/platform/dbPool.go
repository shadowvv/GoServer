package platform

import (
	"gorm.io/gorm"
)

type DBPool struct {
	workers []*DBWorker
	num     int
}

func NewDBPool(num int, db *gorm.DB) *DBPool {
	p := &DBPool{num: num}
	for i := 0; i < num; i++ {
		p.workers = append(p.workers, NewDBWorker(db))
	}
	return p
}

func (p *DBPool) Submit(playerID int64, task DBTask) {
	idx := int(playerID % int64(p.num))
	p.workers[idx].tasks <- task
}
