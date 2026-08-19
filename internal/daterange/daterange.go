// Package daterange は、検索条件で使う日付範囲を検証して境界を計算する
package daterange

import (
	"fmt"
	"strings"
	"time"
)

// Range は、検証済みの開始日と終了日を表す
type Range struct {
	From      string
	To        string
	Precision int
	fromTime  time.Time
	toTime    time.Time
}

// Parse は、YYYY、YYYY-MM、YYYY-MM-DD形式の日付範囲を検証する
func Parse(from, to string) (Range, error) {
	parsedFrom, err := parse(from, "date_from")
	if err != nil {
		return Range{}, err
	}
	parsedTo, err := parse(to, "date_to")
	if err != nil {
		return Range{}, err
	}
	if parsedFrom.precision != 0 && parsedTo.precision != 0 && parsedFrom.precision != parsedTo.precision {
		return Range{}, fmt.Errorf("date_from and date_to must use the same date precision")
	}
	if !parsedFrom.start.IsZero() && !parsedTo.start.IsZero() && parsedFrom.start.After(parsedTo.start) {
		return Range{}, fmt.Errorf("date_from must not be after date_to")
	}
	precision := parsedFrom.precision
	if precision == 0 {
		precision = parsedTo.precision
	}
	return Range{From: parsedFrom.value, To: parsedTo.value, Precision: precision, fromTime: parsedFrom.start, toTime: parsedTo.start}, nil
}

// FromStart は、開始日を指定精度の期間開始として返す
func (rangeValue Range) FromStart() time.Time { return rangeValue.fromTime }

// ToEnd は、終了日を指定精度の期間終了として返す
func (rangeValue Range) ToEnd() time.Time {
	if rangeValue.toTime.IsZero() {
		return time.Time{}
	}
	return next(rangeValue.toTime, rangeValue.Precision).Add(-time.Second)
}

// ToExclusive は、終了日の直後の半開区間境界を返す
func (rangeValue Range) ToExclusive() time.Time {
	if rangeValue.toTime.IsZero() {
		return time.Time{}
	}
	return next(rangeValue.toTime, rangeValue.Precision)
}

type parsedDate struct {
	value     string
	precision int
	start     time.Time
}

// parse は、1つの日付を検証して期間開始へ変換する
func parse(value, name string) (parsedDate, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return parsedDate{}, nil
	}
	for _, format := range []struct {
		precision int
		layout    string
	}{{4, "2006"}, {7, "2006-01"}, {10, "2006-01-02"}} {
		if len(value) != format.precision {
			continue
		}
		parsed, err := time.Parse(format.layout, value)
		if err == nil {
			return parsedDate{value: value, precision: format.precision, start: parsed}, nil
		}
	}
	return parsedDate{}, fmt.Errorf("%s must use YYYY, YYYY-MM, or YYYY-MM-DD with a valid calendar date", name)
}

// next は、指定精度の次の期間開始を返す
func next(value time.Time, precision int) time.Time {
	switch precision {
	case 4:
		return value.AddDate(1, 0, 0)
	case 7:
		return value.AddDate(0, 1, 0)
	default:
		return value.AddDate(0, 0, 1)
	}
}
