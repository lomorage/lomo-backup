package dbx

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/lomorage/lomo-backup/common/types"
	//_ "github.com/mattn/go-sqlite3"
)

const (
	listDirsStmt                = "select id, path, scan_root_dir_id from dirs"
	insertDirStmt               = "insert into dirs (path, scan_root_dir_id, create_time) values (?, ?, ?)"
	getDirIDByPathAndRootIDStmt = "select id from dirs where path = ? and scan_root_dir_id = ?"

	listFilesBySizeStmt = "select d.scan_root_dir_id, d.path, f.name, f.id, f.size from files as f" +
		" inner join dirs as d on f.dir_id=d.id where f.size >= ? order by f.size DESC"
	insertFileStmt = "insert into files (dir_id, name, ext, size, hash, mod_time, create_time)" +
		" values (?, ?, ?, ?, ?, ?, ?)"
	getFileByNameAndDirStmt = "select iso_id, size, hash, mod_time from files where name=? and dir_id=?"
)

const (
	// SuperScanRootDirID is scan root dir id for scan root dir entry in DB
	SuperScanRootDirID = 0
)

var (
	listScanRootDirsStmt = fmt.Sprintf("select id, path from dirs where scan_root_dir_id = %d", SuperScanRootDirID)
)

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
				f.Size, f.Hash, f.ModTime, time.Now().UTC())
			if err != nil {
				return err
			}
			id, err = res.LastInsertId()
			return err
		},
	)
	return int(id), err
}

func (db *DB) ListFilesBySize(minFileSize int) ([]*types.FileInfo, error) {
	files := []*types.FileInfo{}

	err := db.retryIfLocked("list files by size",
		func(tx *sql.Tx) error {
			rows, err := tx.Query(listFilesBySizeStmt, minFileSize)
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
