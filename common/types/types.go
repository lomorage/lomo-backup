package types

import "time"

// DirInfo is structure for directory
type DirInfo struct {
	ID   int
	Path string
}

// FileInfo is structure for file
type FileInfo struct {
	DirID   int
	IsoID   int
	Name    string
	Hash    string
	Size    int
	ModTime time.Time
}
