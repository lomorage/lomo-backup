package common

import (
	"cmp"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/djherbis/times"
	"github.com/sirupsen/logrus"
)

func FormatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func FormatTimeDateOnly(t time.Time) string {
	return t.Format("2006-01-02")
}

func LogDebugObject(key string, obj any) {
	content, _ := json.Marshal(obj)
	logrus.Debugf("%s: %s", key, string(content))
}

func KeepDirsTime(dstRootDir string, dirs map[string]string) {
	parentDirs := map[string]string{}
	for dst, src := range dirs {
		if dst == dstRootDir {
			continue
		}
		for {
			src = filepath.Dir(src)
			dst = filepath.Dir(dst)
			if dst == dstRootDir {
				break
			}

			_, ok := dirs[dst]
			if ok {
				// already in the map, continue
				continue
			}

			_, ok = parentDirs[dst]
			if ok {
				// already in the newly created map, continue
				continue
			}
			parentDirs[dst] = src
		}
	}

	logrus.Debugf("Add parent directories %v", parentDirs)

	for dst, src := range parentDirs {
		dirs[dst] = src
	}

	names := make([]string, len(dirs))
	idx := 0
	for dst := range dirs {
		names[idx] = dst
		idx++
	}

	// sort from the longest to shortest
	slices.SortFunc(names, func(a, b string) int {
		return cmp.Compare(strings.ToLower(b), strings.ToLower(a))
	})

	for _, dst := range names {
		src := dirs[dst]
		err := KeepTime(src, dst, true)
		if err != nil {
			logrus.Warnf("Keep dir original timestamp %s: %s", src, err)
		}
	}
}

func KeepTime(src, dst string, l bool) error {
	ts, err := times.Stat(src)
	if err != nil {
		return err
	}
	if l {
		logrus.Debugf("Keep %s access time %s, mod time %s with original %s", dst,
			FormatTimeDateOnly(ts.AccessTime()),
			FormatTimeDateOnly(ts.ModTime()),
			src,
		)
	}
	return os.Chtimes(dst, ts.AccessTime(), ts.ModTime())
}
