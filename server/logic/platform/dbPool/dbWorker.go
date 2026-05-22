package dbPool

import (
	"gorm.io/gorm"
	"log"
)

type DBWorker struct {
	id    int32
	db    *gorm.DB
	tasks chan DBTask
	quit  chan struct{}
}

// NewDBWorker 创建一个新的数据库 worker
func NewDBWorker(id int32, db *gorm.DB, queueSize int32) *DBWorker {
	w := &DBWorker{
		id:    id,
		db:    db,
		tasks: make(chan DBTask, queueSize),
		quit:  make(chan struct{}),
	}
	go w.run()
	return w
}

// run 持续监听任务
func (w *DBWorker) run() {
	for {
		select {
		case task := <-w.tasks:
			if err := task(w.db); err != nil {
				log.Printf("[DBWorker %d] task error: %v", w.id, err)
			}
		case <-w.quit:
			log.Printf("[DBWorker %d] stopped", w.id)
			return
		}
	}
}

// Submit 向 worker 提交任务
func (w *DBWorker) Submit(task DBTask) {
	select {
	case w.tasks <- task:
	default:
		log.Printf("[DBWorker %d] task queue full, task dropped!", w.id)
	}
}

// Stop 停止 worker
func (w *DBWorker) Stop() {
	close(w.quit)
}
