package persian

import (
	"fmt"
	gotime "time"

	"kaizengo/internal/platform/i18n"
	ptime "kaizengo/internal/platform/time"
)

func init() {
	ptime.Register(calendar{})
	i18n.Register("fa", map[string]string{
		"clock.title":       "ساعت",
		"clock.subtitle":    "زمان محلی، به‌صورت زنده.",
		"clock.calendar":    "تقویم",
		"settings.title":    "تنظیمات",
		"settings.locale":   "زبان",
		"settings.calendar": "تقویم پیش‌فرض",
		"settings.shell":    "عنوان پوسته",
		"settings.save":     "ذخیره",
		"settings.saved":    "ذخیره شد.",
	})
}

type calendar struct{}

func (calendar) ID() string   { return "persian" }
func (calendar) Name() string { return "Persian (Jalali)" }

func (calendar) Format(t gotime.Time) string {
	y, m, d := toJalali(t.Year(), int(t.Month()), t.Day())
	return fmt.Sprintf("%04d/%02d/%02d %02d:%02d:%02d %s",
		y, m, d, t.Hour(), t.Minute(), t.Second(), t.Format("MST"))
}

// toJalali converts Gregorian y/m/d to Jalali (algorithm from common public domain sources).
func toJalali(gy, gm, gd int) (jy, jm, jd int) {
	gy2 := gy - 1600
	gm2 := gm - 1
	gd2 := gd - 1

	gDayNo := 365*gy2 + (gy2+3)/4 - (gy2+99)/100 + (gy2+399)/400
	for i := 0; i < gm2; i++ {
		gDayNo += gregorianMonthDays[i]
	}
	if gm2 > 1 && ((gy%4 == 0 && gy%100 != 0) || gy%400 == 0) {
		gDayNo++
	}
	gDayNo += gd2

	jDayNo := gDayNo - 79
	jNp := jDayNo / 12053
	jDayNo %= 12053
	jy = 979 + 33*jNp + 4*(jDayNo/1461)
	jDayNo %= 1461

	if jDayNo >= 366 {
		jy += (jDayNo - 1) / 365
		jDayNo = (jDayNo - 1) % 365
	}

	var i int
	for i = 0; i < 11 && jDayNo >= jalaliMonthDays[i]; i++ {
		jDayNo -= jalaliMonthDays[i]
	}
	jm = i + 1
	jd = jDayNo + 1
	return jy, jm, jd
}

var gregorianMonthDays = [...]int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
var jalaliMonthDays = [...]int{31, 31, 31, 31, 31, 31, 30, 30, 30, 30, 30, 29}
