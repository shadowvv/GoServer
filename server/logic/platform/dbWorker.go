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
