package dbx

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"strconv"
	"time"

	"github.com/lomorage/lomo-backup/common/types"
	//_ "github.com/mattn/go-sqlite3"
)

var listFilesNotInIsoOrCloudStmt = "select d.scan_root_dir_id, d.path, f.name, f.id, f.iso_id, f.size, f.mod_time from files as f" +
	" inner join dirs as d on f.dir_id=d.id where f.iso_id=0 or f.iso_id=" + strconv.Itoa(types.IsoIDCloud) +
	" order by f.dir_id, f.id"

const (
	listFilesNotInIsoAndCloudStmt = "select d.scan_root_dir_id, d.path, f.name, f.id, f.size, f.hash_local, f.mod_time from files as f" +
		" inner join dirs as d on f.dir_id=d.id where f.iso_id=0 order by f.dir_id, f.id"
	getTotalFileSizeNotInIsoStmt     = "select sum(size) from files where iso_id=0"
	getTotalFilesInIsoStmt           = "select sum(size), count(size) from files where iso_id=?"
	updateBatchFilesIsoIDStmt        = "update files set iso_id=%d where id in (%s)"
	updateFileIsoIDAndRemoteHashStmt = "update files set iso_id=?, hash_remote=? where id=?"

	deleteBatchFilesStmt = "delete from files where id in (%s)"

	getIsoByNameStmt = "select id, size, hash_local, hash_remote, region, bucket, upload_id, upload_key," +
		" create_time from isos where name=?"
	listIsosStmt  = "select id, name, size, status, region, bucket, hash_local, hash_remote, create_time from isos"
	insertIsoStmt = "insert into isos (name, size, status, hash_local, create_time) values (?, ?, ?, ?, ?)"

	resetISOFileInfo = "update isos set status=?, region='', bucket='', hash_remote='' where name=?"

	updateIsoStatusStmt           = "update isos set status=? where id=?"
	updateIsoStatusRemoteHashStmt = "update isos set status=?, hash_remote=? where id=?"
	updateIsoRegionBucketStmt     = "update isos set status=?, region=?, bucket=? where id=?"
	updateIsoRemoteHashStmt       = "update isos set hash_remote=? where id=?"
	updateIsoUploadInfoStmt       = "update isos set region=?,bucket=?, upload_key=?,upload_id=? where id=?"

	insertPartStmt = "insert into parts (iso_id, part_no, hash_local, hash_remote, size, status, create_time)" +
		" values (?, ?, ?, ?, ?, ?, ?)"
	getPartsByIsoIDStmt = "select part_no, hash_local, hash_remote, size, status, etag, create_time " +
		"from parts where iso_id=?"
	deletePartsByIsoIDStmt             = "delete from parts where iso_id=?"
	deletePartsByIsoNameStmt           = "delete from parts where iso_id=(select id from isos where name=?)"
	updatePartUploadStatusStmt         = "update parts set status=? where iso_id=? and part_no=?"
	updatePartUploadEtagStatusStmt     = "update parts set etag=?, status=? where iso_id=? and part_no=?"
	updatePartUploadEtagStatusHashStmt = "update parts set etag=?, status=?, hash_local=?, hash_remote=? where iso_id=? and part_no=?"
)

func (db *DB) ListFilesNotInISOAndCloud() ([]*types.FileInfo, error) {
	files := []*types.FileInfo{}

	err := db.retryIfLocked("list files not in ISO and cloud",
		func(tx *sql.Tx) error {
			rows, err := tx.Query(listFilesNotInIsoAndCloudStmt)
			if err != nil {
				return nil
			}
			for rows.Next() {
				var path, name string
				f := &types.FileInfo{}
				err = rows.Scan(&f.DirID, &path, &name, &f.ID, &f.Size, &f.HashLocal, &f.ModTime)
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

func (db *DB) ListFilesNotInISOOrCloud() ([]*types.FileInfo, error) {
	files := []*types.FileInfo{}

	err := db.retryIfLocked("list files not in ISO or cloud",
		func(tx *sql.Tx) error {
			rows, err := tx.Query(listFilesNotInIsoOrCloudStmt)
			if err != nil {
				return nil
			}
			for rows.Next() {
				var path, name string
				f := &types.FileInfo{}
				err = rows.Scan(&f.DirID, &path, &name, &f.ID, &f.IsoID, &f.Size, &f.ModTime)
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

func (db *DB) GetTotalFilesInIso(isoID int) (uint64, uint64, error) {
	var totalSize, totalCount uint64
	err := db.retryIfLocked("get total file info in ISO "+strconv.Itoa(isoID),
		func(tx *sql.Tx) error {
			return tx.QueryRow(getTotalFilesInIsoStmt, isoID).Scan(&totalSize, &totalCount)
		},
	)
	return totalSize, totalCount, err
}

func (db *DB) GetIsoByName(name string) (*types.ISOInfo, error) {
	iso := &types.ISOInfo{Name: name}
	err := db.retryIfLocked(fmt.Sprintf("get ISO %s", name),
		func(tx *sql.Tx) error {
			err := tx.QueryRow(getIsoByNameStmt, name).Scan(&iso.ID, &iso.Size, &iso.HashLocal,
				&iso.HashRemote, &iso.Region, &iso.Bucket,
				&iso.UploadID, &iso.UploadKey, &iso.CreateTime)
			return err
		},
	)
	if err != nil {
		if IsErrNoRow(err) {
			return nil, nil
		}
		return nil, err
	}
	return iso, nil
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
				err = rows.Scan(&iso.ID, &iso.Name, &iso.Size, &iso.Status, &iso.Region, &iso.Bucket,
					&iso.HashLocal, &iso.HashRemote, &iso.CreateTime)
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
			res, err := tx.Exec(insertIsoStmt, iso.Name, iso.Size, types.IsoCreated, iso.HashLocal,
				time.Now().UTC())
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
			res, err := tx.Exec(insertIsoStmt, iso.Name, iso.Size, types.IsoCreated, iso.HashLocal,
				time.Now().UTC())
			if err != nil {
				return err
			}
			isoID, err = res.LastInsertId()
			if err != nil {
				return err
			}

			res, err = tx.Exec(fmt.Sprintf(updateBatchFilesIsoIDStmt, isoID, fileIDs))
			if err != nil {
				return err
			}
			updatedFiles, err = res.RowsAffected()
			return err
		},
	)
	return int(isoID), int(updatedFiles), err
}

func (db *DB) DeleteBatchFiles(fileIDs string) (int, error) {
	var updatedFiles int64
	err := db.retryIfLocked(fmt.Sprintf("delete files %s", fileIDs),
		func(tx *sql.Tx) error {
			res, err := tx.Exec(fmt.Sprintf(deleteBatchFilesStmt, fileIDs))
			if err != nil {
				return err
			}
			updatedFiles, err = res.RowsAffected()
			return err
		},
	)
	return int(updatedFiles), err
}

func (db *DB) ResetISOUploadInfo(isoFilename string) error {
	return db.retryIfLocked(fmt.Sprintf("reset iso %s upload info", isoFilename),
		func(tx *sql.Tx) error {
			_, err := tx.Exec(resetISOFileInfo, types.IsoUploading, isoFilename)
			if err != nil {
				return err
			}
			_, err = tx.Exec(deletePartsByIsoNameStmt, isoFilename)
			return err
		},
	)
}

func (db *DB) UpdateFileIsoIDAndRemoteHash(isoID, fileID int, remoteHash string) error {
	return db.retryIfLocked(fmt.Sprintf("file %d's iso ID %d", fileID, isoID),
		func(tx *sql.Tx) error {
			_, err := tx.Exec(updateFileIsoIDAndRemoteHashStmt, isoID, remoteHash, fileID)
			return err
		},
	)
}

func (db *DB) UpdateIsoRemoteHash(isoID int, hash string) error {
	return db.retryIfLocked(fmt.Sprintf("update iso %d remote hash %s", isoID, hash),
		func(tx *sql.Tx) error {
			_, err := tx.Exec(updateIsoRemoteHashStmt, hash, isoID)
			return err
		},
	)
}

func (db *DB) UpdateIsoStatus(isoID int, status types.IsoStatus) error {
	return db.retryIfLocked(fmt.Sprintf("update iso %d status %s", isoID, status),
		func(tx *sql.Tx) error {
			_, err := tx.Exec(updateIsoStatusStmt, status, isoID)
			return err
		},
	)
}

func (db *DB) UpdateIsoStatusRemoteHash(isoID int, remoteHash string, status types.IsoStatus) error {
	return db.retryIfLocked(fmt.Sprintf("update iso %d status %s remote hash %s", isoID, status, remoteHash),
		func(tx *sql.Tx) error {
			_, err := tx.Exec(updateIsoStatusRemoteHashStmt, status, remoteHash, isoID)
			return err
		},
	)
}

func (db *DB) UpdateIsoUploadInfo(info *types.ISOInfo) error {
	return db.retryIfLocked(fmt.Sprintf("update ISO %d upload info", info.ID),
		func(tx *sql.Tx) error {
			_, err := tx.Exec(updateIsoUploadInfoStmt, info.Region, info.Bucket,
				info.UploadKey, info.UploadID, info.ID)
			return err
		},
	)
}

func (db *DB) InsertIsoParts(isoID int, parts []*types.PartInfo) error {
	createTime := time.Now().UTC()
	return db.retryIfLocked(fmt.Sprintf("insert iso %d parts", isoID),
		func(tx *sql.Tx) error {
			for _, p := range parts {
				_, err := tx.Exec(insertPartStmt, isoID, p.PartNo, p.HashLocal,
					p.HashRemote, p.Size, types.PartUploading, createTime)
				if err != nil {
					return err
				}
			}
			return nil
		},
	)
}

func (db *DB) GetPartsByIsoID(isoID int) (parts []*types.PartInfo, err error) {
	err = db.retryIfLocked(fmt.Sprintf("get parts of iso %d", isoID),
		func(tx *sql.Tx) error {
			rows, err := tx.Query(getPartsByIsoIDStmt, isoID)
			if err != nil {
				return err
			}
			for rows.Next() {
				p := &types.PartInfo{IsoID: isoID}
				err = rows.Scan(&p.PartNo, &p.HashLocal, &p.HashRemote, &p.Size,
					&p.Status, &p.Etag, &p.CreateTime)
				if err != nil {
					return err
				}
				parts = append(parts, p)
			}
			return rows.Err()
		},
	)
	return
}

func (db *DB) DeletePartsByIsoID(isoID int) error {
	return db.retryIfLocked(fmt.Sprintf("delete iso %d parts", isoID),
		func(tx *sql.Tx) error {
			_, err := tx.Exec(deletePartsByIsoIDStmt, isoID)
			return err
		},
	)
}

func (db *DB) UpdatePartStatus(isoID, partNo int, status types.PartStatus) error {
	return db.retryIfLocked(fmt.Sprintf("update ISO %d part %d status %d", isoID, partNo, status),
		func(tx *sql.Tx) error {
			_, err := tx.Exec(updatePartUploadStatusStmt, status, isoID, partNo)
			return err
		},
	)
}

func (db *DB) UpdatePartEtagAndStatus(isoID, partNo int, etag string, status types.PartStatus) error {
	return db.retryIfLocked(fmt.Sprintf("update ISO %d part %d etag %s status %s",
		isoID, partNo, etag, status),
		func(tx *sql.Tx) error {
			_, err := tx.Exec(updatePartUploadEtagStatusStmt, etag, status, isoID, partNo)
			return err
		},
	)
}

func (db *DB) UpdatePartEtagAndStatusHash(isoID, partNo int, etag, localHash, remoteHash string, status types.PartStatus) error {
	return db.retryIfLocked(fmt.Sprintf("update ISO %d part %d etag %s status %s, local hash %s, remote hash %s",
		isoID, partNo, etag, status, localHash, remoteHash),
		func(tx *sql.Tx) error {
			_, err := tx.Exec(updatePartUploadEtagStatusHashStmt, etag, status, localHash, remoteHash, isoID, partNo)
			return err
		},
	)
}
