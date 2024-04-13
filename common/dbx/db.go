package dbx

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/lomorage/lomo-backup/common/types"
	"github.com/mattn/go-sqlite3"

	//_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

const (
	maxRetry                    = 100000
	insertDirStmt               = "insert into dirs (path, scan_root_dir_id) values (?, ?)"
	getDirIDByPathAndRootIDStmt = "select id from dirs where path = ? and scan_root_dir_id = ?"

	insertFileStmt          = "insert into files (dir_id, name, ext, size, hash, mod_time) values (?, ?, ?, ?, ?, ?)"
	getFileByNameAndDirStmt = "select iso_id, size, hash, mod_time from files where name=? and dir_id=?"
)

const (
	// SuperScanRootDirID is scan root dir id for scan root dir entry in DB
	SuperScanRootDirID = 0
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

func (db *DB) GetDirIDByPathAndRootID(path string, scanRootDirID int) (*int, error) {
	var id *int
	err := db.retryIfLocked(fmt.Sprintf("get dir id %d/%s", scanRootDirID, path),
		func(tx *sql.Tx) error {
			err := tx.QueryRow(getDirIDByPathAndRootIDStmt, path, scanRootDirID).Scan(&id)
			if err != nil {
				if IsErrNoRow(err) {
					return nil
				}
			}
			return err
		},
	)
	return id, err
}

func (db *DB) InsertDir(path string, scanRootDirID int) (int, error) {
	var id int64
	err := db.retryIfLocked(fmt.Sprintf("insert dir %d/%s", scanRootDirID, path),
		func(tx *sql.Tx) error {
			res, err := tx.Exec(insertDirStmt, path, scanRootDirID)
			if err != nil {
				return err
			}
			id, err = res.LastInsertId()
			return err
		},
	)
	return int(id), err
}

func (db *DB) GetFileByNameAndDirID(name string, dirID int) (*types.FileInfo, error) {
	var f *types.FileInfo
	err := db.retryIfLocked(fmt.Sprintf("get file id %d/%s", dirID, name),
		func(tx *sql.Tx) error {
			var isoID, size int
			var hash string
			var modTime time.Time
			err := tx.QueryRow(getFileByNameAndDirStmt, name, dirID).Scan(&isoID, &size, &hash, &modTime)
			if err != nil {
				if IsErrNoRow(err) {
					return nil
				}
				return err
			}
			f = &types.FileInfo{Name: name, DirID: dirID, IsoID: isoID, Size: size,
				Hash: hash, ModTime: modTime}
			return nil
		},
	)
	return f, err
}

func (db *DB) InsertFile(f *types.FileInfo) (int, error) {
	var id int64
	err := db.retryIfLocked(fmt.Sprintf("insert file %d/%s", f.DirID, f.Name),
		func(tx *sql.Tx) error {
			res, err := tx.Exec(insertFileStmt, f.DirID, f.Name,
				strings.ToLower(strings.TrimPrefix(filepath.Ext(f.Name), ".")),
				f.Size, f.Hash, f.ModTime.UTC())
			if err != nil {
				return err
			}
			id, err = res.LastInsertId()
			return err
		},
	)
	return int(id), err
}
