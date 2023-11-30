package utils

import (
	"strconv"
	"time"
)

func TimestampToString(time time.Time) string {
	return strconv.Itoa(int(time.UnixNano()))
}
