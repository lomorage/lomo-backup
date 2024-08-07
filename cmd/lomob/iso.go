package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/diskfs/go-diskfs"
	"github.com/diskfs/go-diskfs/filesystem"
	"github.com/lomorage/lomo-backup/common"
	"github.com/lomorage/lomo-backup/common/datasize"
	lomohash "github.com/lomorage/lomo-backup/common/hash"
	"github.com/lomorage/lomo-backup/common/types"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/xlab/treeprint"
)

var futuretime = time.Date(3000, time.December, 31, 0, 0, 0, 0, time.Now().UTC().Location())

func mkISO(ctx *cli.Context) error {
	err := initLogLevel(ctx.GlobalInt("log-level"))
	if err != nil {
		return err
	}

	debug := ctx.Bool("debug")
	if debug {
		err = initLogLevel(int(logrus.DebugLevel))
		if err != nil {
			return err
		}
	}

	isoSize, err := datasize.ParseString(ctx.String("iso-size"))
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

	files, err := db.ListFilesNotInISOOrCloud()
	if err != nil {
		return err
	}

	var isoFilename string
	if len(ctx.Args()) > 0 {
		isoFilename = ctx.Args()[0]
	}

	logrus.Infof("Total %d files (%s)", len(files), datasize.ByteSize(currentSizeNotInISO).HR())

	for {
		if currentSizeNotInISO < isoSize.Bytes() {
			currSize := datasize.ByteSize(currentSizeNotInISO)
			fmt.Printf("Total size of un-backedup files is %s, less than %s, skip\n",
				currSize.HR(), isoSize.HR())

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

		size, filename, leftFiles, notExistFiles, err := createIso(isoSize.Bytes(), isoFilename, scanRootDirs, files, debug)
		if err != nil {
			return err
		}
		if len(notExistFiles) == 0 {
			logrus.Infof("%d files (%s) are added into %s, and %d files (%s) need to be added",
				len(files)-len(leftFiles), datasize.ByteSize(size).HR(), filename,
				len(leftFiles), datasize.ByteSize(currentSizeNotInISO-size).HR())
		} else {
			fileIDs := bytes.Buffer{}
			notExistSizes := 0
			for _, f := range notExistFiles {
				notExistSizes += f.Size
				fileIDs.WriteString(strconv.Itoa(f.ID))
				fileIDs.WriteString(",")
			}
			logrus.Infof("%d files (%s) are added into %s, %d files (%s) need to be added, %d files (%s) not exist",
				len(files)-len(leftFiles), datasize.ByteSize(size).HR(), filename,
				len(leftFiles), datasize.ByteSize(currentSizeNotInISO-size).HR(),
				len(notExistFiles), datasize.ByteSize(notExistSizes).HR())

			ids := fileIDs.String()
			_, err = db.DeleteBatchFiles(strings.Trim(ids, ","))
			if err != nil {
				return err
			}
			size += uint64(notExistSizes)
		}

		if len(leftFiles) == 0 {
			return nil
		}
		if len(ctx.Args()) > 0 {
			fmt.Println("Please supply another filename")
			return nil
		}
		files = leftFiles
		currentSizeNotInISO -= size
	}
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

func createIso(maxSize uint64, isoFilename string, scanRootDirs map[int]string, files []*types.FileInfo,
	debug bool) (uint64, string, []*types.FileInfo, []*types.FileInfo, error) {
	stagingDir, err := os.MkdirTemp("", "lomobackup-")
	if err != nil {
		return 0, "", nil, nil, err
	}
	if debug {
		logrus.Infof("Staging directory '%s' is kept for debugging", stagingDir)
	} else {
		defer os.RemoveAll(stagingDir)
	}

	const seperater = ','
	var (
		fileCount     int
		filesSize     uint64
		end           time.Time
		notExistFiles []*types.FileInfo
	)
	start := futuretime
	fileIDs := bytes.Buffer{}
	dirsMap := map[string]string{} // dstDir -> srcDir
	for idx, f := range files {
		scanRootDir, ok := scanRootDirs[f.DirID]
		if !ok {
			logrus.Warnf("%s not found root scan dir %d", f.Name, f.DirID)
			continue
		}
		srcFile := filepath.Join(scanRootDir, f.Name)
		dstFile := filepath.Join(filepath.Join(stagingDir, flattenScanRootDir(scanRootDir)), f.Name)

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
			if os.IsNotExist(err) {
				notExistFiles = append(notExistFiles, f)
				logrus.Warnf("'%s' not exist anymore", srcFile)
				continue
			}
			logrus.Warnf("Add %s into %s:%s: %s", srcFile, isoFilename, dstFile, err)
			continue
		}

		if f.ModTime.Before(start) {
			start = f.ModTime
		}
		if f.ModTime.After(end) {
			end = f.ModTime
		}

		err = common.KeepTime(srcFile, dstFile, false)
		if err != nil {
			logrus.Warnf("Keep file original timestamp %s: %s", srcFile, err)
		}

		fileIDs.WriteString(strconv.Itoa(f.ID))
		fileIDs.WriteRune(seperater)

		fileCount++
		filesSize += uint64(f.Size)
		if filesSize < maxSize {
			continue
		}

		// change all destination directory's last modify time and access time
		common.KeepDirsTime(stagingDir, dirsMap)

		name := fmt.Sprintf("%d-%02d-%02d--%d-%02d-%02d", start.Year(), start.Month(), start.Day(),
			end.Year(), end.Month(), end.Day())
		if isoFilename == "" {
			isoFilename = name + ".iso"
		}

		out, err := exec.Command("mkisofs", "-R", "-V", "lomorage: "+name, "-o", isoFilename,
			stagingDir).CombinedOutput()
		if err != nil {
			fmt.Println(string(out))
			return 0, "", nil, nil, err
		}

		fileInfo, err := os.Stat(isoFilename)
		if err != nil {
			return 0, "", nil, nil, err
		}
		isoInfo := &types.ISOInfo{Name: isoFilename, Size: int(fileInfo.Size())}

		hash, err := lomohash.CalculateHashFile(isoFilename)
		if err != nil {
			return 0, "", nil, nil, err
		}
		isoInfo.SetHashLocal(hash)
		// create db entry and update file info
		start := time.Now()
		_, count, err := db.CreateIsoWithFileIDs(isoInfo,
			strings.TrimSuffix(fileIDs.String(), string(seperater)))
		if err == nil && count != fileCount {
			logrus.Warnf("Expect to update %d files while updated %d files", fileCount, count)
		}

		logrus.Infof("Takes %s to update iso_id for %d files in DB", time.Since(start).Truncate(time.Second).String(), count)
		return filesSize, isoFilename, files[idx+1:], notExistFiles, err
	}

	return filesSize, isoFilename, nil, nil, nil
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

	fmt.Fprint(writer, "ID\tName\tSize\tStatus\tRegion\tBucket\tFiles Count\tCreate Time\tLocal Hash\n")
	for _, iso := range isos {
		_, count, err := db.GetTotalFilesInIso(iso.ID)
		if err != nil {
			return err
		}
		fmt.Fprintf(writer, "%d\t%s\t%s\t%s\t%s\t%s\t%d\t%s\t%s\n", iso.ID, iso.Name,
			datasize.ByteSize(iso.Size).HR(), iso.Status, iso.Region, iso.Bucket, count,
			common.FormatTime(iso.CreateTime.Local()), iso.HashLocal)
	}
	return nil
}

func dumpISO(ctx *cli.Context) error {
	if len(ctx.Args()) == 0 {
		return errors.New("please provide one iso filename")
	}

	tree, err := genTreeInIso(ctx.Args()[0])
	if err != nil {
		return err
	}
	fmt.Println(tree)
	return nil
}

func genTreeInIso(isoFilename string) (string, error) {
	const root = "/"
	rootNode := treeprint.NewWithRoot(root)

	disk, err := diskfs.Open(isoFilename)
	if err != nil {
		return "", err
	}
	fs, err := disk.GetFilesystem(0)
	if err != nil {
		return "", err
	}

	err = fileInfoFor(root, fs, rootNode)
	if err != nil {
		return "", err
	}

	return rootNode.String(), nil
}

func fileInfoFor(path string, fs filesystem.FileSystem, currNode treeprint.Tree) error {
	files, err := fs.ReadDir(path)
	if err != nil {
		return err
	}

	slices.SortStableFunc(files, func(a, b os.FileInfo) int {
		return strings.Compare(a.Name(), b.Name())
	})

	for _, file := range files {
		t := file.ModTime()
		fullPath := filepath.Join(path, file.Name())
		if file.IsDir() {
			childNode := currNode.AddMetaBranch(
				fmt.Sprintf("\t%02d/%02d/%d", t.Month(), t.Day(), t.Year()), file.Name())
			err = fileInfoFor(fullPath, fs, childNode)
			if err != nil {
				return err
			}
			continue
		}
		currNode.AddMetaNode(fmt.Sprintf("\t%12s\t%02d/%02d/%d", strconv.Itoa(int(file.Size())),
			t.Month(), t.Day(), t.Year()), file.Name())
	}
	return nil
}
