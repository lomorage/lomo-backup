package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/lomorage/lomo-backup/common/datasize"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
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

	idx := 0
	dirsToPrint := make([]string, len(dirs))
	for _, dir := range dirs {
		scanRootDir, ok := dirs[dir.ScanRootDirID]
		if !ok {
			logrus.Warnf("%s's scan root dir %d is not found", dir.Path, dir.ScanRootDirID)
			continue
		}
		dirsToPrint[idx] = filepath.Join(scanRootDir.Path, dir.Path)
		idx++
	}

	sort.Slice(dirsToPrint, func(i, j int) bool {
		return strings.Compare(dirsToPrint[i], dirsToPrint[j]) < 0
	})

	for _, d := range dirsToPrint {
		fmt.Println(d)
	}

	return nil
}
