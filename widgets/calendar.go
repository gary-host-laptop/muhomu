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
	month := now.Month()
	today := now.Day()

	daysInMonth := time.Date(now.Year(), month+1, 0, 0, 0, 0, 0, time.Local).Day()
	firstDay    := int(time.Date(now.Year(), month, 1, 0, 0, 0, 0, time.Local).Weekday())
	firstDayMon := (firstDay + 6) % 7

	weekdays := []string{"m", "t", "w", "t", "f", "s", "s"}
	monthNames := []string{
		"january", "february", "march", "april", "may", "june",
		"july", "august", "september", "october", "november", "december",
	}

	leftCount   := 16
	rightCount  := daysInMonth - leftCount
	rightOffset := 3
	totalRows   := leftCount
	if rightCount+rightOffset > totalRows {
		totalRows = rightCount + rightOffset
	}

	var sb strings.Builder
	sb.WriteString(`<div class="widget-body"><table id="calendarTable">`)

	// 1 empty spacer row + connector row with month name spanning middle two columns
	fmt.Fprintf(&sb, `
<tr>
  <td class="spacer-col1"></td>
  <td class="spacer-col2"></td>
  <td class="spacer-col3"></td>
  <td class="spacer-col4"></td>
</tr>
<tr>
  <td class="connector-col1"></td>
  <td colspan="2" class="cal-header-year cal-header-today" style="text-align:center;border-bottom:1px solid var(--border);">%s</td>
  <td class="connector-col4"></td>
</tr>`, monthNames[month-1])

	// day rows
	for i := 0; i < totalRows; i++ {
		leftDay, rightDay := 0, 0
		if i < leftCount {
			leftDay = i + 1
		}
		if i >= rightOffset {
			if idx := i - rightOffset; idx < rightCount {
				rightDay = leftCount + idx + 1
			}
		}

		fmt.Fprintf(&sb, "<tr>")

		if leftDay > 0 {
			todayL := ""
			if leftDay == today {
				todayL = " today"
			}
			fmt.Fprintf(&sb, `<td class="col-left-num%s">%02d</td><td class="col-left-wk%s">%s</td>`,
				todayL, leftDay, todayL, weekdays[(firstDayMon+leftDay-1)%7])
		} else {
			sb.WriteString(`<td class="col-left-num"></td><td class="col-left-wk"></td>`)
		}
		if rightDay > 0 {
			todayR := ""
			if rightDay == today {
				todayR = " today"
			}
			fmt.Fprintf(&sb, `<td class="col-right-num%s">%02d</td><td class="col-right-wk%s">%s</td>`,
				todayR, rightDay, todayR, weekdays[(firstDayMon+rightDay-1)%7])
		} else {
			sb.WriteString(`<td class="col-right-num"></td><td class="col-right-wk"></td>`)
		}
		sb.WriteString("</tr>")
	}

	sb.WriteString(`</table></div>`)
	return wrap("calendar", "blue", "カレンダー", "", sb.String()), nil
}
