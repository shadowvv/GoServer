package platform

import "gorm.io/gorm"

type DBTask func(db *gorm.DB)

type DBWorker struct {
	tasks chan DBTask
	db    *gorm.DB
}

func NewDBWorker(db *gorm.DB) *DBWorker {
	w := &DBWorker{
		tasks: make(chan DBTask, 1000),
		db:    db,
	}
	go w.run()
	return w
}

func (w *DBWorker) run() {
	for task := range w.tasks {
		task(w.db)
	}
}

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
