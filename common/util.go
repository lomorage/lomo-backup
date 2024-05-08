package common

import (
	"encoding/json"
	"time"

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
