package types

import (
	"github.com/tencent-go/pkg/errx"
	"time"
)

type Date string

func (d Date) Validate() errx.Error {
	_, err := time.Parse("2006-01-02", string(d))
	if err != nil {
		return errx.Validation.WithMsgf("date %s format is incorrect", d).Err()
	}
	return nil
}

func (d Date) ToTime() (time.Time, errx.Error) {
	parse, err := time.Parse("2006-01-02", string(d))
	if err != nil {
		return time.Time{}, errx.Wrap(err).Err()
	}
	return parse, nil
}

func (d Date) AddXDays(x int) (*Date, errx.Error) {
	parse, err := d.ToTime()
	if err != nil {
		return nil, err
	}
	newDate := Date(parse.AddDate(0, 0, x).Format("2006-01-02"))
	return &newDate, nil
}

func NewTodayDate() *Date {
	today := Date(time.Now().Format("2006-01-02"))
	return &today
}

func ToDate(t time.Time) *Date {
	date := Date(t.Format("2006-01-02"))
	return &date
}

func UnixToDate(ut int64) *Date {
	var t time.Time
	t = time.Unix(ut, 0)
	if ut > 1e10 {
		t = time.Unix(ut/1000, 0)
	}
	return ToDate(t)
}
