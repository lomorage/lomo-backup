package types

import (
	"time"

	"github.com/lomorage/lomo-backup/common/hash"
)

// IsoIDCloud is to flag the file is uploaded into cloud and not packed in ISO yet
const IsoIDCloud = -1

const (
	MetadataKeyHashOrig    = "hash_orig"
	MetadataKeyHashEncrypt = "hash_enc"
)

type IsoStatus int

const (
	IsoCreating IsoStatus = iota
	IsoCreated
	IsoUploading
	IsoUploaded
)

func (s IsoStatus) String() string {
	switch s {
	case IsoCreating:
		return "Creating"
	case IsoCreated:
		return "Created, not uploaded"
	case IsoUploading:
		return "Uploadinging"
	case IsoUploaded:
		return "Uploaded"
	}
	return "Unknown"
}

type PartStatus int

const (
	PartUploading PartStatus = iota
	PartUploaded
	PartUploadFailed
)

func (p PartStatus) String() string {
	switch p {
	case PartUploading:
		return "Uploadinging"
	case PartUploaded:
		return "Uploaded"
	}
	return "Unknown"
}

// DirInfo is structure for directory
type DirInfo struct {
	ID            int
	ScanRootDirID int
	NumberOfFiles int
	NumberOfDirs  int
	TotalFileSize int
	RefID         string // ID in cloud
	Path          string
	ModTime       time.Time
	CreateTime    time.Time
}

// FileInfo is structure for file
type FileInfo struct {
	ID    int
	DirID int
	IsoID int
	RefID string // ID in cloud
	Name  string
	// HashLocal use hex encoding method as command line sha256sum output is hex, so as to easy compare
	HashLocal string
	// HashRemote uses base64 encoding as it is required by AWS
	HashRemote string
	Size       int
	ModTime    time.Time
}

// SetHashLocal
func (fi *FileInfo) SetHashLocal(data []byte) {
	fi.HashLocal = hash.CalculateHashHex(data)
}

// ISOInfo is structure for one iso file
type ISOInfo struct {
	ID         int
	Name       string
	Region     string
	Bucket     string
	UploadKey  string
	UploadID   string
	HashLocal  string
	HashRemote string
	Size       int
	Status     IsoStatus
	CreateTime time.Time
}

func (ii *ISOInfo) SetHashLocal(data []byte) {
	ii.HashLocal = hash.CalculateHashHex(data)
}

func (ii *ISOInfo) SetHashRemote(data []byte) {
	ii.HashRemote = hash.CalculateHashBase64(data)
}

// PartInfo is struct for one upload part of one iso file
type PartInfo struct {
	IsoID      int
	PartNo     int
	Size       int
	Status     PartStatus
	Etag       string
	HashLocal  string
	HashRemote string
	CreateTime time.Time
}

func (pi *PartInfo) SetHashLocal(data []byte) {
	pi.HashLocal = hash.CalculateHashHex(data)
}

func (pi *PartInfo) SetHashRemote(data []byte) {
	pi.HashRemote = hash.CalculateHashBase64(data)
}
