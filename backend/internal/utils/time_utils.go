package utils

import "time"

// GetCurrentPeriodStart возвращает начало текущего 15-секундного периода
func GetCurrentPeriodStart() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(),
		now.Hour(), now.Minute(), (now.Second()/15)*15, 0, now.Location())
}
