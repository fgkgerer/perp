package funding

import (
	"strconv"
	"time"
)

// NextSettleUnix 返回下一期资金费结算时刻（Unix 秒）。
// intervalSec 为配置结算间隔；28800（8h）时对齐 UTC 00:00、08:00、16:00（与主流 CEX 一致）。
func NextSettleUnix(now time.Time, intervalSec int) int64 {
	if intervalSec <= 0 {
		intervalSec = 28800
	}
	if intervalSec == 28800 {
		return nextUtc8hSlot(now).Unix()
	}
	epoch := now.Unix()
	return ((epoch / int64(intervalSec)) + 1) * int64(intervalSec)
}

func nextUtc8hSlot(now time.Time) time.Time {
	t := now.UTC()
	y, m, d := t.Date()
	for _, hour := range []int{0, 8, 16} {
		cand := time.Date(y, m, d, hour, 0, 0, 0, time.UTC)
		if cand.After(t) {
			return cand
		}
	}
	return time.Date(y, m, d+1, 0, 0, 0, 0, time.UTC)
}

// ScheduleLabel 人类可读的结算周期说明。
func ScheduleLabel(intervalSec int) string {
	if intervalSec <= 0 {
		intervalSec = 28800
	}
	if intervalSec == 28800 {
		return "UTC 每 8 小时：00:00、08:00、16:00"
	}
	h := intervalSec / 3600
	if h > 0 && intervalSec%3600 == 0 {
		return "每 " + strconv.Itoa(h) + " 小时结算"
	}
	return "每 " + strconv.Itoa(intervalSec) + " 秒结算"
}
