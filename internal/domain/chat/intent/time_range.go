package intent

import (
	"strings"
	"time"
)

// BuildSelectTimeRange 根据口语中的日期/时段推算 start/end（供 fallback 使用）。
func BuildSelectTimeRange(dayLabel, dayPeriod string, now time.Time) (start, end string) {
	loc := now.Location()
	day := resolveDayBase(dayLabel, now, loc)
	startHour, endHour := dayPeriodHours(dayPeriod)
	if startHour < 0 {
		start = formatIntentTime(day)
		end = formatIntentTime(day.Add(24*time.Hour - time.Nanosecond))
		return start, end
	}
	startAt := time.Date(day.Year(), day.Month(), day.Day(), startHour, 0, 0, 0, loc)
	endAt := time.Date(day.Year(), day.Month(), day.Day(), endHour, 0, 0, 0, loc)
	if endHour <= startHour {
		endAt = endAt.Add(24 * time.Hour)
	}
	return formatIntentTime(startAt), formatIntentTime(endAt)
}

func resolveDayBase(dayLabel string, now time.Time, loc *time.Location) time.Time {
	dayLabel = strings.TrimSpace(dayLabel)
	nowDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	switch dayLabel {
	case "昨天":
		return nowDate.Add(-24 * time.Hour)
	case "前天":
		return nowDate.Add(-48 * time.Hour)
	default:
		return nowDate
	}
}

func dayPeriodHours(period string) (startHour, endHour int) {
	switch strings.TrimSpace(period) {
	case "早上":
		return 5, 11
	case "上午":
		return 5, 12
	case "中午":
		return 11, 14
	case "下午":
		return 12, 19
	case "傍晚":
		return 14, 19
	case "晚上":
		return 19, 24
	default:
		return -1, -1
	}
}

func formatIntentTime(t time.Time) string {
	return t.Format("2006-01-02T15:04:05")
}
