package common

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/djherbis/times"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestKeepDirsTime(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)

	tmpDir, err := os.MkdirTemp("", "lomotest")
	require.Nil(t, err)

	defer os.RemoveAll(tmpDir)

	srcRootDir := "../"
	dirs := []string{"cmd/lomob", "test", "test/scripts", "vendor/github.com/golang",
		"vendor/github.com/aws", "vendor/github.com/google"}
	testDirs := []string{"cmd", "cmd/lomob", "test", "test/scripts", "vendor", "vendor/github.com",
		"vendor/github.com/golang", "vendor/github.com/google", "vendor/github.com/aws"}

	dirsMap := map[string]string{}
	for _, d := range dirs {
		dst := filepath.Join(tmpDir, d)
		require.Nil(t, os.MkdirAll(dst, 0755))

		dirsMap[dst] = filepath.Join(srcRootDir, d)
	}

	KeepDirsTime(tmpDir, dirsMap)

	for _, d := range testDirs {
		dst := filepath.Join(tmpDir, d)
		src := filepath.Join(srcRootDir, d)

		dstStat, err := times.Stat(dst)
		require.Nil(t, err)
		srcStat, err := times.Stat(src)
		require.Nil(t, err)

		require.Equal(t, srcStat.AccessTime(), dstStat.AccessTime(), d)
		require.Equal(t, srcStat.ModTime(), dstStat.ModTime(), d)
	}

	require.Nil(t, os.RemoveAll(tmpDir))
}
