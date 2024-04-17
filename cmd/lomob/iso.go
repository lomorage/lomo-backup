package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/djherbis/times"
	"github.com/lomorage/lomo-backup/common/datasize"
	"github.com/lomorage/lomo-backup/common/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

const volumePrefix = "lomobackup: "

func mkISO(ctx *cli.Context) error {
	isoSize, err := datasize.ParseString(ctx.String("iso-size"))
	if err != nil {
		return err
	}

	err = initLogLevel(ctx.GlobalInt("log-level"))
	if err != nil {
		return err
	}

	err = initDB(ctx.GlobalString("db"))
	if err != nil {
		return err
	}

	currentSizeNotInISO, err := db.TotalFileSizeNotInISO()
	if err != nil {
		return err
	}

	scanRootDirs, err := db.ListScanRootDirs()
	if err != nil {
		return err
	}

	files, err := db.ListFilesNotInISO()
	if err != nil {
		return err
	}

	var isoFilename string
	if len(ctx.Args()) > 0 {
		isoFilename = ctx.Args()[0]
	} else {
		isoFilename = time.Now().Format("2006-01-02T15-04-05") + ".iso"
	}

	for {
		if currentSizeNotInISO < isoSize.Bytes() {
			currSize := datasize.ByteSize(currentSizeNotInISO)
			expSize := datasize.ByteSize(isoSize)
			fmt.Printf("Total size of un-backedup files is %s, less than %s, skip\n",
				currSize.HR(), expSize.HR())

			return nil
		}

		iso, err := db.GetIsoByName(isoFilename)
		if err != nil {
			return err
		}
		if iso != nil {
			return errors.Errorf("%s was created at %s, and its size is %s", isoFilename,
				iso.CreateTime.Truncate(time.Second).Local(),
				datasize.ByteSize(iso.Size).HR())
		}

		volumeIdentifier := volumePrefix + strings.TrimSuffix(isoFilename, filepath.Ext(isoFilename))

		size, leftFiles, err := createIso(isoSize.Bytes(), isoFilename, volumeIdentifier, scanRootDirs, files)
		if err != nil {
			return err
		}
		logrus.Infof("%d files (%s) are added into %s, and %d files (%s) need to be added",
			len(files)-len(leftFiles), datasize.ByteSize(size).HR(), isoFilename,
			len(leftFiles), datasize.ByteSize(currentSizeNotInISO-size).HR())

		if len(ctx.Args()) > 0 {
			fmt.Println("Please supply another filename")
			return nil
		}
		files = leftFiles
		currentSizeNotInISO -= size
		isoFilename = time.Now().Format("2006-01-02T15-04-05") + ".iso"
	}
}

func keepTime(src, dst string) error {
	ts, err := times.Stat(src)
	if err != nil {
		return err
	}
	return os.Chtimes(dst, ts.AccessTime(), ts.ModTime())
}

func createFileInStaging(srcFile, dstFile string) error {
	src, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstFile)
	if err != nil {
		return err
	}

	_, err = io.Copy(dst, src)

	return err
}

func createIso(maxSize uint64, isoFilename, volumeIdentifier string, scanRootDirs map[int]string,
	files []*types.FileInfo) (uint64, []*types.FileInfo, error) {
	stagingDir, err := os.MkdirTemp("", "lomobackup-")
	if err != nil {
		return 0, nil, err
	}
	defer os.RemoveAll(stagingDir)

	var (
		fileCount int
		isoSize   uint64
	)
	const seperater = ','
	fileIDs := bytes.Buffer{}
	dirsMap := map[string]string{} // dstDir -> srcDir
	for idx, f := range files {
		scanRootDir, ok := scanRootDirs[f.DirID]
		if !ok {
			logrus.Warnf("%s not found root scan dir %d", f.Name, f.DirID)
			continue
		}
		srcFile := filepath.Join(scanRootDir, f.Name)
		dstFile := filepath.Join(stagingDir, f.Name)

		// create dir
		dstDir := filepath.Dir(dstFile)
		_, ok = dirsMap[dstDir]
		if !ok {
			err = os.MkdirAll(dstDir, 0744)
			if err != nil {
				logrus.Warnf("Create staging dir %s: %s", filepath.Dir(dstFile), err)
				continue
			}
			dirsMap[dstDir] = filepath.Dir(srcFile)
		}

		err = createFileInStaging(srcFile, dstFile)
		if err != nil {
			logrus.Warnf("Add %s into %s:%s: %s", srcFile, isoFilename, dstFile, err)
			continue
		}

		err = keepTime(srcFile, dstFile)
		if err != nil {
			logrus.Warnf("Keep file original timestamp %s: %s", srcFile, err)
		}

		fileIDs.WriteString(strconv.Itoa(f.ID))
		fileIDs.WriteRune(seperater)

		fileCount++
		isoSize += uint64(f.Size)
		if isoSize < maxSize {
			continue
		}

		// change all destination directory's last modify time and access time
		for dst, src := range dirsMap {
			err = keepTime(src, dst)
			if err != nil {
				logrus.Warnf("Keep dir original timestamp %s: %s", src, err)
			}
		}

		out, err := exec.Command("mkisofs", "-R", "-V", volumeIdentifier, "-o", isoFilename,
			stagingDir).CombinedOutput()
		if err != nil {
			fmt.Println(string(out))
			return 0, nil, err
		}

		// create db entry and update file info
		start := time.Now()
		_, count, err := db.CreateIsoWithFileIDs(&types.ISOInfo{Name: isoFilename, Size: int(isoSize)},
			strings.TrimSuffix(fileIDs.String(), string(seperater)))
		if err == nil && count != fileCount {
			logrus.Warnf("Expect to update %d files while updated %d files", fileCount, count)
		}

		logrus.Infof("Takes %s to update iso_id for %d files in DB", time.Since(start).Truncate(time.Second).String(), count)
		return isoSize, files[idx+1:], err
	}

	return isoSize, nil, nil
}

func listISO(ctx *cli.Context) error {
	err := initDB(ctx.GlobalString("db"))
	if err != nil {
		return err
	}

	isos, err := db.ListISOs()
	if err != nil {
		return err
	}
	writer := tabwriter.NewWriter(os.Stdout, 0, 0, 4, ' ', tabwriter.TabIndent)
	defer writer.Flush()

	fmt.Fprint(writer, "ID\tName\tSize\tCreate Time\n")
	for _, iso := range isos {
		fmt.Fprintf(writer, "%d\t%s\t%s\t%s\n", iso.ID, iso.Name,
			datasize.ByteSize(iso.Size).HR(),
			iso.CreateTime.Truncate(time.Second).Local())
	}
	return nil
}
