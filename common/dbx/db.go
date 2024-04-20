package dbx

import (
	"database/sql"
	"regexp"

	"github.com/mattn/go-sqlite3"

	//_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

const (
	maxRetry = 100000
)

var (
	noRowRegex  = regexp.MustCompile(sql.ErrNoRows.Error())
	dbLockRegex = regexp.MustCompile(sqlite3.ErrBusy.Error())
)

var (
	ErrMaxRetry = errors.New("Beyond max retry")
)

type DB struct {
	db *sql.DB
}

// OpenDB opens db with given filename.
func OpenDB(filename string) (*DB, error) {
	db := &DB{}
	var err error
	db.db, err = sql.Open("sqlite3", filename)
	return db, err
}

// IsNoRow check the error is no row or not
func IsErrNoRow(err error) bool {
	return err == sql.ErrNoRows || noRowRegex.MatchString(err.Error())
}

func (db *DB) retryIfLocked(log string, run func(tx *sql.Tx) error) error {
	tx, err := db.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for retry := 0; retry < maxRetry; retry++ {
		err := run(tx)
		if err != nil {
			if dbLockRegex.MatchString(err.Error()) {
				continue
			}
			return err
		}
		return tx.Commit()
	}
	return errors.Wrap(sqlite3.ErrBusy, log)
}
