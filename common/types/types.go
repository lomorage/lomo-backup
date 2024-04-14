package types

import "time"

// DirInfo is structure for directory
type DirInfo struct {
	ID            int
	ScanRootDirID int
	Path          string
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
	Size       int
	CreateTime time.Time
}
