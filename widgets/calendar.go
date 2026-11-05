package widgets

import (
	"fmt"
	"html/template"
	"time"
)

type CalendarWidget struct{}

func (w *CalendarWidget) ID() string { return "calendar" }

type calendarRow struct {
	LeftDay      string
	LeftToday    bool
	LeftWeekday  string
	Year         int
	Month        string
	RightDay     string
	RightToday   bool
	RightWeekday string
	EmptyRight   bool
}

func (w *CalendarWidget) Render(ctx RenderContext) (template.HTML, error) {
	now := time.Now()
	month := now.Month()
	today := now.Day()

	daysInMonth := time.Date(now.Year(), month+1, 0, 0, 0, 0, 0, time.Local).Day()
	firstDay := int(time.Date(now.Year(), month, 1, 0, 0, 0, 0, time.Local).Weekday())
	firstDayMon := (firstDay + 6) % 7

	weekdays := []string{"m", "t", "w", "t", "f", "s", "s"}
	monthNames := []string{
		"january", "february", "march", "april", "may", "june",
		"july", "august", "september", "october", "november", "december",
	}

	leftCount := 16
	rightCount := daysInMonth - leftCount
	// The right strip starts directly BELOW the month name, which spans
	// the first two rows: rows 0-1 are the header block, so 17 lines up
	// with 03 (row 2).
	rightOffset := 2
	// The year block is appended after the left strip: its 16 days already
	// fill rows 0-15, so a two-row year (mirroring the two-row month name)
	// needs two rows of its own. The right strip never needs more than
	// rightCount+rightOffset = 17 rows, which the 18 below always covers.
	totalRows := leftCount + 2

	rows := make([]calendarRow, totalRows)
	for i := 0; i < totalRows; i++ {
		var row calendarRow
		leftDay, rightDay := 0, 0
		if i < leftCount {
			leftDay = i + 1
		}

		// Row 0 carries the left strip's first date in cols 1-2 and the month
		// name in cols 3-4; the header cell rowspans the next row too, so the
		// name is a two-row-tall block beside days 01-02. Row 1 therefore emits
		// only its two left cells — cols 3-4 belong to the rowspan, and adding
		// cells there would push them into new columns. The right strip begins
		// at row 2, i.e. immediately below the month name.
		switch {
		case leftDay > 0:
			row.LeftDay = fmt.Sprintf("%02d", leftDay)
			row.LeftToday = leftDay == today
			row.LeftWeekday = weekdays[(firstDayMon+leftDay-1)%7]
		case i == leftCount:
			// Year block, two rows tall at the bottom of the left strip.
			row.Year = now.Year()
		case i == leftCount+1:
			// Cols 1-2 are covered by the year's rowspan; emit nothing.
		}

		if i >= rightOffset {
			if idx := i - rightOffset; idx < rightCount {
				rightDay = leftCount + idx + 1
			}
		}

		switch {
		case i == 0:
			// Month header over the right strip's columns, two rows tall;
			// its inline border-bottom IS the underline. Alignment lives in
			// the stylesheet (the cell is left-aligned, not centred).
			row.Month = monthNames[month-1]
		case i == 1:
			// Covered by the header's rowspan — no right cells here.
		case rightDay > 0:
			row.RightDay = fmt.Sprintf("%02d", rightDay)
			row.RightToday = rightDay == today
			row.RightWeekday = weekdays[(firstDayMon+rightDay-1)%7]
		default:
			row.EmptyRight = true
		}

		rows[i] = row
	}

	inner, err := renderWidgetTemplate("calendar.tmpl", rows)
	if err != nil {
		return "", err
	}
	return wrap(ctx, "calendar", "カレンダー", "", string(inner)), nil
}
