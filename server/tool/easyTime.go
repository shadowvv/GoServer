package tool

import (
	"time"

	"github.com/robfig/cron/v3"
)

const (
	DAY_MILLI    int64 = 24 * 60 * 60 * 1000
	HOUR_MILLI   int64 = 60 * 60 * 1000
	MINUTE_MILLI int64 = 60 * 1000
)

// Now 返回当前时间
func Now() time.Time {
	return time.Now()
}

// UnixNow 返回当前时间戳（秒）
func UnixNow() int64 {
	return time.Now().Unix()
}

// UnixNowMilli 获取当前时间戳（毫秒）
func UnixNowMilli() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// ParseTime2TimeStamp 将字符串解析为时间戳
func ParseTime2TimeStamp(timeStr string) (int64, error) {
	t, err := time.Parse("2006-01-02 15:04:05", timeStr)
	if err != nil {
		return 0, err
	}
	return t.Unix(), nil
}

// TodayZero 返回今天零点时间
func TodayZero() time.Time {
	now := time.Now()
	year, month, day := now.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, now.Location())
}

func GetTodayZeroByTimeStamp(timeStamp int64) int64 {
	// 毫秒 → time.Time
	t := time.UnixMilli(timeStamp)

	// 取所在时区的当天 00:00:00
	y, m, d := t.Date()
	loc := t.Location()

	zero := time.Date(y, m, d, 0, 0, 0, 0, loc)

	// 返回毫秒时间戳
	return zero.UnixMilli()
}

// IsSameDay 判断两个时间是否是同一天
func IsSameDay(t1, t2 time.Time) bool {
	y1, m1, d1 := t1.Date()
	y2, m2, d2 := t2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

// GetNatureDayDistance 获取两个自然日之间的距离
func GetNatureDayDistance(now, target int64) int32 {
	nowTime := time.UnixMilli(now)
	targetTime := time.UnixMilli(target)

	ny, nm, nd := nowTime.Date()
	ty, tm, td := targetTime.Date()

	nowZero := time.Date(ny, nm, nd, 0, 0, 0, 0, nowTime.Location())
	targetZero := time.Date(ty, tm, td, 0, 0, 0, 0, targetTime.Location())

	days := int(targetZero.Sub(nowZero) / (24 * time.Hour))
	if days < 0 {
		days = -days
	}
	return int32(days)
}

// WeekStart 获取指定时间所在周的周一
func WeekStart(t time.Time) time.Time {
	y, m, d := t.Date()
	wd := int(t.Weekday())
	if wd == 0 { // Sunday -> 7
		wd = 7
	}
	// 回退到周一
	return time.Date(y, m, d-wd+1, 0, 0, 0, 0, t.Location())
}

// weekStart 获取指定时间所在周的周一
func WeekStartByMilli(ts int64) time.Time {
	t := time.UnixMilli(ts)
	y, m, d := t.Date()
	wd := int(t.Weekday())
	if wd == 0 {
		wd = 7
	}
	return time.Date(y, m, d-wd+1, 0, 0, 0, 0, t.Location())
}

// GetNatureWeekDistance 获取两个自然周之间的距离
func GetNatureWeekDistance(now, target int64) int32 {
	nowStart := WeekStart(time.UnixMilli(now))
	targetStart := WeekStart(time.UnixMilli(target))

	weeks := int(targetStart.Sub(nowStart) / (7 * 24 * time.Hour))
	if weeks < 0 {
		weeks = -weeks
	}
	return int32(weeks)
}

// GetNatureMonthDistance 获取两个自然月之间的距离
func GetNatureMonthDistance(now, target int64) int32 {
	n := time.UnixMilli(now)
	t := time.UnixMilli(target)

	ny, nm, _ := n.Date()
	ty, tm, _ := t.Date()

	months := (ty-int(ny))*12 + int(tm-nm)
	if months < 0 {
		months = -months
	}
	return int32(months)
}

func WeekDayWithTimeStamp(timestamp int64) int {
	t := time.Unix(timestamp, 0)
	return int(t.Weekday())
}

func MonthDayWithTimeStamp(timestamp int64) int {
	t := time.Unix(timestamp, 0)
	return t.Day()
}

// MilliToTime 毫秒时间戳转换为时间
func MilliToTime(milli int64) time.Time {
	sec := milli / 1000
	nsec := (milli % 1000) * int64(time.Millisecond)
	return time.Unix(sec, nsec)
}

// ValidateCron 验证 cron 表达式
func ValidateCron(expr string) bool {
	_, err := cron.ParseStandard(expr)
	return err == nil
}

// CheckCronMatch 判断时间戳是否匹配 5 字段 cron 表达式（精度：分钟）
func CheckCronMatchWithExpression(cronExpr string, ts int64) (bool, error) {
	t := time.Unix(ts, 0).Truncate(time.Minute)

	// 使用标准 5 字段解析器（无秒）
	parser := cron.NewParser(
		cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow,
	)

	schedule, err := parser.Parse(cronExpr)
	if err != nil {
		return false, err
	}

	// 获取 t 的前一分钟作为参考时间
	prevTime := t.Add(-time.Minute)

	// 找到 prevTime 后下一个应该触发的时间
	next := schedule.Next(prevTime)

	// 如果 next 正好等于当前时间 t，则表示匹配成功
	return next.Equal(t), nil
}

// CheckCronMatch 验证 cron 表达式是否匹配给定的时间戳
func CheckCronMatch(cron cron.Schedule, ts int64) (bool, error) {
	t := time.Unix(ts, 0).Truncate(time.Minute)
	// 获取 t 的前一分钟作为参考时间
	prevTime := t.Add(-time.Minute)

	// 找到 prevTime 后下一个应该触发的时间
	next := cron.Next(prevTime)

	// 如果 next 正好等于当前时间 t，则表示匹配成功
	return next.Equal(t), nil
}

func GetTodayDataIntByTimeStamp(currentTime int64) int32 {
	t := time.UnixMilli(currentTime)

	y, m, d := t.Date()

	return int32(y*10000 + int(m)*100 + d)
}

// GetNatureDayDistanceByDateInt 获取两个 YYYYMMDD 之间的自然日差
func GetNatureDayDistanceByDateInt(currentDay int32, targetDay int32) int32 {
	if currentDay <= 0 || targetDay <= 0 {
		return 0
	}
	cy := int(currentDay / 10000)
	cm := time.Month((currentDay / 100) % 100)
	cd := int(currentDay % 100)
	ty := int(targetDay / 10000)
	tm := time.Month((targetDay / 100) % 100)
	td := int(targetDay % 100)

	current := time.Date(cy, cm, cd, 0, 0, 0, 0, time.Local)
	target := time.Date(ty, tm, td, 0, 0, 0, 0, time.Local)
	return GetNatureDayDistance(current.UnixMilli(), target.UnixMilli())
}

func GetTodayDataStringByTimeStamp(currentTime int64) string {
	t := time.UnixMilli(currentTime)
	return t.Format("20060102")
}

func GetMondayDataStringByTimeStamp(currentTime int64) string {
	t := time.UnixMilli(currentTime)
	// 获取本周周一的日期
	weekday := int(t.Weekday())
	if weekday == 0 { // Sunday
		weekday = 7
	}
	monday := t.AddDate(0, 0, -(weekday - 1))
	return monday.Format("20060102")
}

func CheckIsMondayByTimeStamp(currentTime int64) bool {
	return time.UnixMilli(currentTime).Weekday() == time.Monday
}
