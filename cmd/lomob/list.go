package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/lomorage/lomo-backup/common"
	"github.com/lomorage/lomo-backup/common/datasize"
	"github.com/lomorage/lomo-backup/common/dbx"
	"github.com/lomorage/lomo-backup/common/types"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/xlab/treeprint"
)

func listBigfiles(ctx *cli.Context) error {
	fileSize, err := datasize.ParseString(ctx.String("file-size"))
	if err != nil {
		return err
	}

	err = initDB(ctx.GlobalString("db"))
	if err != nil {
		return err
	}

	files, err := db.ListFilesBySize(int(fileSize))
	if err != nil {
		return err
	}

	scanRootDirs, err := db.ListScanRootDirs()
	if err != nil {
		return err
	}

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', tabwriter.TabIndent)
	defer writer.Flush()

	fmt.Fprint(writer, "Name\tSize\n")
	for _, f := range files {
		scanRootDir, ok := scanRootDirs[f.DirID]
		if !ok {
			logrus.Warnf("%s's scan root dir %d is not found", f.Name, f.DirID)
			continue
		}
		fmt.Fprintf(writer, "%s\t%s\n", filepath.Join(scanRootDir, f.Name),
			datasize.ByteSize(f.Size).HR(),
		)
	}
	return nil
}

func listScanedDirs(ctx *cli.Context) error {
	err := initDB(ctx.GlobalString("db"))
	if err != nil {
		return err
	}

	dirs, err := db.ListDirs()
	if err != nil {
		return err
	}

	if ctx.Bool("table-view") {
		printDirsByTable(dirs)
	} else {
		printDirsByTree(dirs)
	}
	return nil
}

func printDirsByTree(dirs map[int]*types.DirInfo) {
	scanRootTree := map[int]treeprint.Tree{}
	dirsToPrint := []*types.DirInfo{}
	for _, dir := range dirs {
		if dir.ScanRootDirID == dbx.SuperScanRootDirID {
			scanRootTree[dir.ID] = treeprint.NewWithRoot(dir.Path)
			continue
		}
		// skip root scan dir. note that table view won't check later as it may have some files
		if dir.Path == "" {
			continue
		}
		scanRootDir, ok := dirs[dir.ScanRootDirID]
		if !ok {
			logrus.Warnf("%s's scan root dir %d is not found", dir.Path, dir.ScanRootDirID)
			continue
		}

		// use the fullpath so as to sort easily
		dir.Path = filepath.Join(scanRootDir.Path, dir.Path)
		dirsToPrint = append(dirsToPrint, dir)
	}

	sort.Slice(dirsToPrint, func(i, j int) bool {
		return strings.Compare(dirsToPrint[i].Path, dirsToPrint[j].Path) < 0
	})

	nodes := map[string]treeprint.Tree{} // store all parents
	// create tree view
	for _, d := range dirsToPrint {
		scanRootDir, ok := dirs[d.ScanRootDirID]
		if !ok {
			logrus.Warnf("%s's scan root dir %d is not found", d.Path, d.ScanRootDirID)
			continue
		}

		var parentNode treeprint.Tree
		parentDir, name := filepath.Split(d.Path)
		parentDir = strings.TrimSuffix(parentDir, string(os.PathSeparator))
		if parentDir == scanRootDir.Path {
			// add the node into scan root tree
			parentNode, ok = scanRootTree[scanRootDir.ID]
		} else {
			parentNode, ok = nodes[parentDir]
		}
		if !ok {
			logrus.Warnf("%s's parent node is not found", d.Path)
		}
		// TODO: pretty output with meta value
		if d.NumberOfDirs > 0 {
			//newNode := parentNode.AddMetaBranch("\t\t"+common.FormatTimeDateOnly(d.ModTime), name)
			newNode := parentNode.AddBranch(name)
			nodes[d.Path] = newNode
		} else {
			//parentNode.AddMetaNode(common.FormatTimeDateOnly(d.ModTime), name)
			parentNode.AddNode(name)
		}
	}

	for _, tree := range scanRootTree {
		fmt.Println(tree)
	}

}

func printDirsByTable(dirs map[int]*types.DirInfo) {
	dirsToPrint := []*types.DirInfo{}
	for _, dir := range dirs {
		if dir.ScanRootDirID == dbx.SuperScanRootDirID {
			continue
		}
		scanRootDir, ok := dirs[dir.ScanRootDirID]
		if !ok {
			logrus.Warnf("%s's scan root dir %d is not found", dir.Path, dir.ScanRootDirID)
			continue
		}

		dir.Path = filepath.Join(scanRootDir.Path, dir.Path)
		dirsToPrint = append(dirsToPrint, dir)
	}

	sort.Slice(dirsToPrint, func(i, j int) bool {
		return strings.Compare(dirsToPrint[i].Path, dirsToPrint[j].Path) < 0
	})

	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', tabwriter.TabIndent)
	defer writer.Flush()

	fmt.Fprint(writer, "File Counts\tTotal File Size\tMod Time\tPath\n")

	for _, d := range dirsToPrint {
		fmt.Fprintf(writer, "%d\t%s\t%s\t%s\n", d.NumberOfFiles, datasize.ByteSize(d.TotalFileSize).HR(),
			common.FormatTime(d.ModTime), d.Path)
	}
}
