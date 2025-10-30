package platform

import "gorm.io/gorm"

type DBTask func(db *gorm.DB)

type DBWorker struct {
	tasks chan DBTask
	db    *gorm.DB
}

func NewDBWorker(db *gorm.DB, workerTaskSize int32) *DBWorker {
	w := &DBWorker{
		tasks: make(chan DBTask, workerTaskSize),
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
