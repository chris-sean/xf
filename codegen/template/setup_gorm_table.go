package template

const SETUP_GORM_TABLE_GO = `package dao

import (
	"#ModuleName#/errs"
	"#ModuleName#/model"

	"github.com/go-sql-driver/mysql"
)

func setupTables() xf.ErrorType {
	if err := gdb.AutoMigrate(
		//DO_NOT_TOUCH_THIS_COMMENT:SETUP_TABLE
	); err != nil {
		return errs.ErrCannotInitGORM(err)
	}
	return nil
}

func setupTriggers() xf.ErrorType {
	var triggerDDLs []string

	//DO_NOT_TOUCH_THIS_COMMENT:APPEND_TRIGGER_DDL

	for _, ddl := range triggerDDLs {
		err := gdb.Exec(ddl).Error

		if me, ok := err.(*mysql.MySQLError); ok {
			if me.Number == 1359 {
				// Trigger already exists
				continue
			}
		}

		if err != nil {
			return errs.ErrCannotInitGORM(err)
		}
	}

	return nil
}
`
