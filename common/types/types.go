package types

import "time"

type IsoStatus int

const (
	Creating IsoStatus = iota
	Created
	Uploading
	Uploaded
)

func (s IsoStatus) String() string {
	switch s {
	case Creating:
		return "Creating"
	case Created:
		return "Created, not uploaded"
	case Uploading:
		return "Uploadinging"
	case Uploaded:
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
	Hash       string
	Size       int
	Status     IsoStatus
	CreateTime time.Time
}

// PartInfo is struct for one upload part of one iso file
type PartInfo struct {
	IsoID      int
	PartNo     int
	Size       int
	Bucket     string
	Hash       string
	UploadKey  string
	UploadID   string
	CreateTime time.Time
}
