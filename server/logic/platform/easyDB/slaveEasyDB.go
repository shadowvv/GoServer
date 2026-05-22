package easyDB

import "gorm.io/gorm"

var slaveGameDB *gorm.DB

func SetSlaveGameDB(db *gorm.DB) {
	slaveGameDB = db
}
