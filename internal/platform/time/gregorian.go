package time

import gotime "time"

func init() {
	Register(gregorian{})
}

type gregorian struct{}

func (gregorian) ID() string   { return "gregorian" }
func (gregorian) Name() string { return "Gregorian" }

func (gregorian) Format(t gotime.Time) string {
	return t.Format("2006-01-02 15:04:05 MST")
}
