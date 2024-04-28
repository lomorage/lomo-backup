package types

import "time"

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
	Path          string
	ModTime       time.Time
	CreateTime    time.Time
}

// FileInfo is structure for file
type FileInfo struct {
	ID      int
	DirID   int
	IsoID   int
	Name    string
	Hash    string
	Size    int
	ModTime time.Time
}

// ISOInfo is structure for one iso file
type ISOInfo struct {
	ID         int
	Name       string
	Region     string
	Bucket     string
	UploadKey  string
	UploadID   string
	HashHex    string
	HashBase64 string
	Size       int
	Status     IsoStatus
	CreateTime time.Time
}

// PartInfo is struct for one upload part of one iso file
type PartInfo struct {
	IsoID      int
	PartNo     int
	Size       int
	Status     PartStatus
	Etag       string
	HashHex    string
	HashBase64 string
	CreateTime time.Time
}
