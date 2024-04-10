package types

import "time"

// DirInfo is structure for directory
type DirInfo struct {
	ID   int
	Path string
}

// FileInfo is structure for file
type FileInfo struct {
	IsoID      int
	Name       string
	Dir        *DirInfo
	Ext        string
	Size       int
	ModifyTime time.Time
}
