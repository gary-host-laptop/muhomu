package widgets

import (
	"fmt"
	"html/template"
	"strings"
	"time"
)

type CalendarWidget struct{}

func (w *CalendarWidget) ID() string { return "calendar" }

func (w *CalendarWidget) Render(ctx RenderContext) (template.HTML, error) {
	now   := time.Now()
	year  := now.Year()
	month := now.Month()
	today := now.Day()

	daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, time.Local).Day()
	firstDay    := int(time.Date(year, month, 1, 0, 0, 0, 0, time.Local).Weekday())
	firstDayMon := (firstDay + 6) % 7

	weekdays := []string{"m", "t", "w", "t", "f", "s", "s"}
	monthNames := []string{
		"january", "february", "march", "april", "may", "june",
		"july", "august", "september", "october", "november", "december",
	}

	// week number (iso)
	_, weekNum := now.ISOWeek()

	// year progress — days elapsed / days in year
	dayOfYear  := now.YearDay()
	daysInYear := 365
	if isLeap(year) {
		daysInYear = 366
	}
	yearPct := int(float64(dayOfYear) / float64(daysInYear) * 100)

	leftCount   := 16
	rightCount  := daysInMonth - leftCount
	rightOffset := 3
	totalRows   := leftCount
	if rightCount+rightOffset > totalRows {
		totalRows = rightCount + rightOffset
	}

	var sb strings.Builder
	sb.WriteString(`<div class="widget-body"><table id="calendarTable">`)

	// upper spacer rows — year and month name live here
	fmt.Fprintf(&sb, `
<tr>
  <td colspan="4" class="cal-header-year">%d · %s</td>
</tr>
<tr>
  <td class="spacer-col1"></td>
  <td class="spacer-col2"></td>
  <td class="spacer-col3"></td>
  <td class="spacer-col4"></td>
</tr>`, year, monthNames[month-1])

	// connector row
	sb.WriteString(`<tr>
<td class="connector-col1"></td>
<td class="connector-col2"></td>
<td class="connector-col3"></td>
<td class="connector-col4"></td>
</tr>`)

	// day rows
	for i := 0; i < totalRows; i++ {
		leftDay  := 0
		rightDay := 0
		if i < leftCount {
			leftDay = i + 1
		}
		if i >= rightOffset {
			idx := i - rightOffset
			if idx < rightCount {
				rightDay = leftCount + idx + 1
			}
		}

		isToday := leftDay == today || rightDay == today
		todayClass := ""
		if isToday {
			todayClass = ` class="today"`
		}

		fmt.Fprintf(&sb, "<tr%s>", todayClass)

		if leftDay > 0 {
			wdL := weekdays[(firstDayMon+leftDay-1)%7]
			fmt.Fprintf(&sb, `<td class="col-left-num">%02d</td><td class="col-left-wk">%s</td>`, leftDay, wdL)
		} else {
			sb.WriteString(`<td class="col-left-num"></td><td class="col-left-wk"></td>`)
		}

		if rightDay > 0 {
			wdR := weekdays[(firstDayMon+rightDay-1)%7]
			fmt.Fprintf(&sb, `<td class="col-right-num">%02d</td><td class="col-right-wk">%s</td>`, rightDay, wdR)
		} else {
			sb.WriteString(`<td class="col-right-num"></td><td class="col-right-wk"></td>`)
		}

		sb.WriteString("</tr>")
	}

	// footer — week number and year percentage
	// ids preserved so js can update year-progress if needed
	fmt.Fprintf(&sb, `
<tr><td colspan="3" class="footer-sep-line"></td><td class="footer-sep-spacer"></td></tr>
<tr><td colspan="3" class="footer-text-col1-3"><span id="week-num">W%02d</span></td><td class="footer-spacer"></td></tr>
<tr><td colspan="3" class="footer-sep-line"></td><td class="footer-sep-spacer"></td></tr>
<tr><td colspan="3" class="footer-text-col1-3"><span id="year-progress">y · %d%%</span></td><td class="footer-spacer"></td></tr>
<tr><td colspan="3" class="footer-sep-line"></td><td class="footer-sep-spacer"></td></tr>`,
		weekNum, yearPct)

	sb.WriteString(`</table></div>`)

	return wrap("calendar", "blue", "カレンダー", "", sb.String()), nil
}

func isLeap(year int) bool {
	return year%400 == 0 || (year%4 == 0 && year%100 != 0)
}
