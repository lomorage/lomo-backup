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
	maxRetry = 100000

	listDirsStmt                = "select id, path, scan_root_dir_id from dirs"
	insertDirStmt               = "insert into dirs (path, scan_root_dir_id, create_time) values (?, ?, ?)"
	getDirIDByPathAndRootIDStmt = "select id from dirs where path = ? and scan_root_dir_id = ?"

	listFilesNotInIsoStmt = "select d.scan_root_dir_id, d.path, f.name, f.id, f.size from files as f" +
		" inner join dirs as d on f.dir_id=d.id where f.iso_id=0 order by f.dir_id, f.id"
	insertFileStmt = "insert into files (dir_id, name, ext, size, hash, mod_time, create_time)" +
		" values (?, ?, ?, ?, ?, ?, ?)"
	getFileByNameAndDirStmt      = "select iso_id, size, hash, mod_time from files where name=? and dir_id=?"
	getTotalFileSizeNotInIsoStmt = "select sum(size) from files where iso_id=0"
	updateIsoIDStmt              = "update files set iso_id=%d where id in (%s)"

	listIsosStmt  = "select id, name, size from isos"
	insertIsoStmt = "insert into isos (name, size, create_time) values (?, ?, ?)"
)

const (
	// SuperScanRootDirID is scan root dir id for scan root dir entry in DB
	SuperScanRootDirID = 0
)

var (
	listScanRootDirsStmt = fmt.Sprintf("select id, path from dirs where scan_root_dir_id = %d", SuperScanRootDirID)
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
			res, err := tx.Exec(insertDirStmt, path, scanRootDirID, time.Now().UTC())
			if err != nil {
				return err
			}
			id, err = res.LastInsertId()
			return err
		},
	)
	return int(id), err
}

func (db *DB) ListDirs() (map[int]*types.DirInfo, error) {
	dirs := make(map[int]*types.DirInfo)
	err := db.retryIfLocked("list dirs",
		func(tx *sql.Tx) error {
			rows, err := tx.Query(listDirsStmt)
			if err != nil {
				return nil
			}
			for rows.Next() {
				dir := &types.DirInfo{}
				err = rows.Scan(&dir.ID, &dir.Path, &dir.ScanRootDirID)
				if err != nil {
					return err
				}
				dirs[dir.ID] = dir
			}
			return rows.Err()
		},
	)
	return dirs, err
}

func (db *DB) ListScanRootDirs() (map[int]string, error) {
	dirs := make(map[int]string)
	err := db.retryIfLocked("list scan root dirs",
		func(tx *sql.Tx) error {
			rows, err := tx.Query(listScanRootDirsStmt)
			if err != nil {
				return nil
			}
			for rows.Next() {
				var (
					id   int
					path string
				)
				err = rows.Scan(&id, &path)
				if err != nil {
					return err
				}
				dirs[id] = path
			}
			return rows.Err()
		},
	)
	return dirs, err
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
				f.Size, f.Hash, f.ModTime.UTC(), time.Now().UTC())
			if err != nil {
				return err
			}
			id, err = res.LastInsertId()
			return err
		},
	)
	return int(id), err
}

func (db *DB) ListFilesNotInISO() ([]*types.FileInfo, error) {
	files := []*types.FileInfo{}

	err := db.retryIfLocked("list files not in ISO",
		func(tx *sql.Tx) error {
			rows, err := tx.Query(listFilesNotInIsoStmt)
			if err != nil {
				return nil
			}
			for rows.Next() {
				var path, name string
				f := &types.FileInfo{}
				err = rows.Scan(&f.DirID, &path, &name, &f.ID, &f.Size)
				if err != nil {
					return err
				}
				f.Name = filepath.Join(path, name)

				files = append(files, f)
			}
			return rows.Err()
		},
	)
	return files, err
}

func (db *DB) TotalFileSizeNotInISO() (uint64, error) {
	var totalSize uint64
	err := db.retryIfLocked("total file size not in ISO",
		func(tx *sql.Tx) error {
			return tx.QueryRow(getTotalFileSizeNotInIsoStmt).Scan(&totalSize)
		},
	)
	return totalSize, err
}

func (db *DB) ListISOs() ([]*types.ISOInfo, error) {
	isos := []*types.ISOInfo{}
	err := db.retryIfLocked("list ISOs",
		func(tx *sql.Tx) error {
			rows, err := tx.Query(listIsosStmt)
			if err != nil {
				return nil
			}
			for rows.Next() {
				iso := &types.ISOInfo{}
				err = rows.Scan(&iso.ID, &iso.Name, &iso.Size)
				if err != nil {
					return err
				}
				isos = append(isos, iso)
			}
			return rows.Err()
		},
	)
	return isos, err
}

func (db *DB) InsertISO(iso *types.ISOInfo) (int, error) {
	var id int64
	err := db.retryIfLocked(fmt.Sprintf("insert iso %s", iso.Name),
		func(tx *sql.Tx) error {
			res, err := tx.Exec(insertIsoStmt, iso.Name, iso.Size, time.Now().UTC())
			if err != nil {
				return err
			}
			id, err = res.LastInsertId()
			return err
		},
	)
	return int(id), err
}

func (db *DB) CreateIsoWithFileIDs(iso *types.ISOInfo, fileIDs string) (int, int, error) {
	var isoID, updatedFiles int64
	err := db.retryIfLocked(fmt.Sprintf("insert iso %s", iso.Name),
		func(tx *sql.Tx) error {
			res, err := tx.Exec(insertIsoStmt, iso.Name, iso.Size, time.Now().UTC())
			if err != nil {
				return err
			}
			isoID, err = res.LastInsertId()
			if err != nil {
				return err
			}

			res, err = tx.Exec(fmt.Sprintf(updateIsoIDStmt, isoID, fileIDs))
			if err != nil {
				return err
			}
			updatedFiles, err = res.RowsAffected()
			return err
		},
	)
	return int(isoID), int(updatedFiles), err
}
