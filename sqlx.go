package xf

import (
	"database/sql"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

var SlowSQLDuration = time.Millisecond * 100
var VerySlowSQLDuration = time.Second

func NewDBXWithLogger(dbx *sqlx.DB, traceID string, file string) *DBXWithLogger {
	return &DBXWithLogger{DB: dbx, traceID: traceID, file: file}
}

type DBXWithLogger struct {
	*sqlx.DB
	traceID string
	file    string
}

func (o *DBXWithLogger) Query(query string, args ...interface{}) (*sql.Rows, error) {
	begin := time.Now()
	defer SQLTrace(o.traceID, o.file, begin, query, args...)
	return o.DB.Query(query, args...)
}

func (o *DBXWithLogger) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	begin := time.Now()
	defer SQLTrace(o.traceID, o.file, begin, query, args...)
	return o.DB.Queryx(query, args...)
}

func (o *DBXWithLogger) QueryRowx(query string, args ...interface{}) *sqlx.Row {
	begin := time.Now()
	defer SQLTrace(o.traceID, o.file, begin, query, args...)
	return o.DB.QueryRowx(query, args...)
}

func (o *DBXWithLogger) Exec(query string, args ...interface{}) (sql.Result, error) {
	begin := time.Now()
	defer SQLTrace(o.traceID, o.file, begin, query, args...)
	return o.DB.Exec(query, args...)
}

func NewTXXWithLogger(txx *sqlx.Tx, traceID string, file string) *TXXWithLogger {
	return &TXXWithLogger{Tx: txx, traceID: traceID, file: file}
}

type TXXWithLogger struct {
	*sqlx.Tx
	traceID string
	file    string
}

func (o *TXXWithLogger) Query(query string, args ...interface{}) (*sql.Rows, error) {
	begin := time.Now()
	defer SQLTrace(o.traceID, o.file, begin, query, args...)
	return o.Tx.Query(query, args...)
}

func (o *TXXWithLogger) Queryx(query string, args ...interface{}) (*sqlx.Rows, error) {
	begin := time.Now()
	defer SQLTrace(o.traceID, o.file, begin, query, args...)
	return o.Tx.Queryx(query, args...)
}

func (o *TXXWithLogger) QueryRowx(query string, args ...interface{}) *sqlx.Row {
	begin := time.Now()
	defer SQLTrace(o.traceID, o.file, begin, query, args...)
	return o.Tx.QueryRowx(query, args...)
}

func (o *TXXWithLogger) Exec(query string, args ...interface{}) (sql.Result, error) {
	begin := time.Now()
	defer SQLTrace(o.traceID, o.file, begin, query, args...)
	return o.Tx.Exec(query, args...)
}

func SQLTrace(traceID, file string, begin time.Time, sql string, args ...interface{}) {
	elapsed := time.Since(begin)
	slowTag := ""
	if elapsed >= VerySlowSQLDuration {
		slowTag = " [VERY SLOW SQL]"
		Warnf("[%s] %s\n[%vms]%s %v; args=%v", traceID, file, float64(elapsed.Nanoseconds())/1e6, slowTag, sql, args)
	} else {
		if elapsed >= SlowSQLDuration {
			slowTag = " [SLOW SQL]"
		}
		Debugf("[%s] %s\n[%vms]%s %v; args=%v", traceID, file, float64(elapsed.Nanoseconds())/1e6, slowTag, sql, args)
	}
}

// SecureSQLName 过滤表/库/字段名，防止SQL注入
func SecureSQLName(name string) string {
	s := strings.ReplaceAll(name, ";", "")
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, "(", "")
	s = strings.ReplaceAll(s, ")", "")
	s = strings.ReplaceAll(s, `"`, "")
	return s
}

var sourceFile string

func init() {
	_, file, _, _ := runtime.Caller(0)
	sourceFile = file
}

func CreateDSNWith(driver, hostname string, port int32, user, password string) string {
	switch driver {
	case "mysql":
		return fmt.Sprintf("%v:%v@tcp(%v:%v)/", user, password, hostname, port)
	}
	return ""
}

func MustQueryRow(db *sqlx.DB, query string, args ...interface{}) map[string]interface{} {
	row := db.QueryRowx(query, args...)

	err := row.Err()

	if err != nil {
		panic(ErrDBQueryError(query, err))
	}

	cols, err := row.Columns()

	if err != nil {
		panic(ErrDBQueryError(query, err))
	}

	colCnt := len(cols)
	cache := make([]interface{}, colCnt)

	for i := range cache {
		var any interface{}
		cache[i] = &any
	}

	err = row.Scan(cache...)

	if err == sql.ErrNoRows {
		return nil
	}

	if err != nil {
		panic(ErrDBQueryError(query, err))
	}

	rowValue := map[string]interface{}{}

	for i, data := range cache {
		rowValue[cols[i]] = *data.(*interface{})
	}

	return rowValue
}
