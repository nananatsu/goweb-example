package dao

import (
	"database/sql/driver"
	"fmt"
	"time"
)

const timeFormat = "2006-01-02 15:04:05"

type LocalTime struct {
	time.Time
}

// Scan implements sql.Scanner interface
func (t *LocalTime) Scan(value interface{}) error {
	timestr, ok := value.([]byte)
	if ok {
		time, err := time.Parse(timeFormat, string(timestr))
		if err != nil {
			return err
		}
		t.Time = time
		return nil
	}
	return fmt.Errorf("解析时间戳失败")
}

// Value implements driver.Valuer interface
func (t *LocalTime) Value() (driver.Value, error) {
	return t.Time.Format(timeFormat), nil
}
