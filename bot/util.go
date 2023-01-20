package bot

import (
	"strconv"
	"strings"
	"time"
)

func TrimChannelString(chStr string) string {
	chStr = strings.TrimPrefix(chStr, "<#")
	chStr = strings.TrimSuffix(chStr, ">")
	return chStr
}

func ParseSnowflake(id string) (time.Time, error) {
	n, err := strconv.ParseInt(id, 0, 63)
	if err != nil {
		return time.Now(), err
	}
	return time.Unix(((n>>22)+1420070400000)/1000, 0), nil
}
