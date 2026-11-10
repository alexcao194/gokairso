package performance

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

type kpiMemberRecords struct {
	feedbacks   []FeedbackResponse
	initiatives []InitiativeResponse
	lateLogs    []LateLogResponse
}

const (
	kpiFormCode  = "Mẫu số: IT.KPI.001"
	kpiNoteWidth = 50
)

func kpiFmtDay(t time.Time) string     { return t.In(vietnamLocation()).Format("02/01") }
func kpiFmtDayTime(t time.Time) string { return t.In(vietnamLocation()).Format("02/01 15:04") }

func kpiSigned(n float64) string {
	r := math.Round(n*10) / 10
	if r > 0 {
		return fmt.Sprintf("+%g", r)
	}
	return fmt.Sprintf("%g", r)
}

func feedbackNoteLines(list []FeedbackResponse) []string {
	lines := make([]string, 0, len(list))
	for _, f := range list {
		quote := ""
		if f.FeedbackText != "" {
			quote = fmt.Sprintf(": “%s”", f.FeedbackText)
		}
		lines = append(lines, fmt.Sprintf("%s — %s%s (%s)", kpiFmtDay(f.CreatedAt), f.Content, quote, kpiSigned(f.Estimate)))
	}
	return lines
}

func initiativeNoteLines(list []InitiativeResponse) []string {
	lines := make([]string, 0, len(list))
	for _, i := range list {
		lines = append(lines, fmt.Sprintf("%s — %s (%s)", kpiFmtDay(derefTime(i.ApprovedAt)), i.Content, kpiSigned(i.Estimate)))
	}
	return lines
}

func lateLogNoteLines(list []LateLogResponse) []string {
	lines := make([]string, 0, len(list))
	for _, l := range list {
		lines = append(lines, fmt.Sprintf("%s — %s (%s)", kpiFmtDayTime(l.LateTime), l.Content, kpiSigned(-l.Estimate)))
	}
	return lines
}

func safeSheetName(name string) string {
	repl := strings.NewReplacer("\\", " ", "/", " ", "*", " ", "?", " ", ":", " ", "[", " ", "]", " ")
	out := strings.TrimSpace(repl.Replace(name))
	if out == "" {
		out = "KPI"
	}
	if r := []rune(out); len(r) > 31 {
		out = string(r[:31])
	}
	return out
}

func safeFilenamePart(s string) string {
	out := strings.NewReplacer("/", "_", "\\", "_", " ", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_").Replace(s)
	if out == "" {
		return "member"
	}
	return out
}

func kpiNoteHeight(lines []string) float64 {
	visual := 0
	for _, l := range lines {
		rows := (len([]rune(l)) + (kpiNoteWidth - 4) - 1) / (kpiNoteWidth - 4)
		if rows < 1 {
			rows = 1
		}
		visual += rows
	}
	return float64(visual*15 + 6)
}

type kpiLeaf struct {
	stt    int
	label  string
	how    string
	weight float64
	result float64
	notes  []string
}

type kpiGroup struct {
	stt      string
	label    string
	children []kpiLeaf
}

func buildKPIWorkbook(members []KPIMemberResponse, settings KPISettingsResponse, records map[string]*kpiMemberRecords, monthLabel, controllerName string) ([]byte, error) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	used := map[string]bool{}
	firstSheet := ""
	for idx, m := range members {
		name := safeSheetName(firstNonEmpty(m.MemberAlias, m.MemberName))
		base := name
		for n := 1; used[name]; n++ {
			name = safeSheetName(fmt.Sprintf("%s_%d", base, n))
		}
		used[name] = true

		if idx == 0 {
			if err := f.SetSheetName("Sheet1", name); err != nil {
				return nil, err
			}
			firstSheet = name
		} else if _, err := f.NewSheet(name); err != nil {
			return nil, err
		}
		if err := fillKPISheet(f, name, m, settings, records[m.MemberID], monthLabel, controllerName); err != nil {
			return nil, err
		}
	}
	if firstSheet != "" {
		if i, err := f.GetSheetIndex(firstSheet); err == nil {
			f.SetActiveSheet(i)
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func firstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

func cellName(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}

func fillKPISheet(f *excelize.File, sheet string, k KPIMemberResponse, settings KPISettingsResponse, rec *kpiMemberRecords, monthLabel, controllerName string) error {
	st, err := newKPIStyles(f)
	if err != nil {
		return err
	}

	widths := []float64{6, 15, 24, 30, 28, 12, 12, kpiNoteWidth}
	for i, w := range widths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		if err := f.SetColWidth(sheet, col, col, w); err != nil {
			return err
		}
	}

	_ = f.MergeCell(sheet, "F1", "H1")
	_ = f.SetCellValue(sheet, "F1", kpiFormCode)
	_ = f.SetCellStyle(sheet, "F1", "H1", st.formCode)
	_ = f.SetRowHeight(sheet, 1, 16)

	_ = f.MergeCell(sheet, "A2", "H2")
	_ = f.SetCellValue(sheet, "A2", "BẢNG ĐÁNH GIÁ HIỆU QUẢ CÔNG VIỆC HÀNG THÁNG (KPI)")
	_ = f.SetCellStyle(sheet, "A2", "H2", st.title)
	_ = f.SetRowHeight(sheet, 2, 26)

	setInfo := func(row int, labelA, valA, labelB, valB string) {
		_ = f.SetCellValue(sheet, cellName(1, row), labelA)
		_ = f.SetCellStyle(sheet, cellName(1, row), cellName(1, row), st.infoLabel)
		_ = f.SetCellValue(sheet, cellName(3, row), valA)
		_ = f.SetCellStyle(sheet, cellName(3, row), cellName(3, row), st.infoValue)
		_ = f.SetCellValue(sheet, cellName(5, row), labelB)
		_ = f.SetCellStyle(sheet, cellName(5, row), cellName(5, row), st.infoLabel)
		_ = f.SetCellValue(sheet, cellName(7, row), valB)
		_ = f.SetCellStyle(sheet, cellName(7, row), cellName(7, row), st.infoValue)
	}
	setInfo(4, "KPI Tháng:", monthLabel, "Người duyệt:", "")
	setInfo(5, "Người thực hiện:", k.MemberName, "Người kiểm soát:", controllerName)

	_ = f.SetCellValue(sheet, "A6", "Kết quả đánh giá:")
	_ = f.SetCellStyle(sheet, "A6", "A6", st.infoLabel)
	_ = f.MergeCell(sheet, "C6", "H6")
	_ = f.SetCellValue(sheet, "C6", "85 điểm coi như hoàn thành công việc — từ 86 điểm trở lên coi như vượt chỉ tiêu")

	const h1, h2 = 8, 9
	_ = f.MergeCell(sheet, cellName(1, h1), cellName(1, h2))
	_ = f.MergeCell(sheet, cellName(2, h1), cellName(3, h2))
	_ = f.MergeCell(sheet, cellName(4, h1), cellName(4, h2))
	_ = f.MergeCell(sheet, cellName(5, h1), cellName(5, h2))
	_ = f.MergeCell(sheet, cellName(6, h1), cellName(7, h1))
	_ = f.MergeCell(sheet, cellName(8, h1), cellName(8, h2))
	_ = f.SetCellValue(sheet, cellName(1, h1), "STT")
	_ = f.SetCellValue(sheet, cellName(2, h1), "TIÊU CHÍ ĐÁNH GIÁ")
	_ = f.SetCellValue(sheet, cellName(4, h1), "CÁCH THỰC HIỆN")
	_ = f.SetCellValue(sheet, cellName(5, h1), "MÔ TẢ THỰC TẾ")
	_ = f.SetCellValue(sheet, cellName(6, h1), "MỨC ĐỘ HOÀN THÀNH (%)")
	_ = f.SetCellValue(sheet, cellName(8, h1), "GHI CHÚ")
	_ = f.SetCellValue(sheet, cellName(6, h2), "Hệ số (%)")
	_ = f.SetCellValue(sheet, cellName(7, h2), "Kết quả (%)")
	_ = f.SetCellStyle(sheet, cellName(1, h1), cellName(8, h2), st.header)
	_ = f.SetRowHeight(sheet, h1, 20)
	_ = f.SetRowHeight(sheet, h2, 20)

	groups := kpiGroupsFor(k, settings, rec)

	r := h2 + 1
	var sectionRows []int
	for _, g := range groups {
		sectionRow := r
		sectionRows = append(sectionRows, sectionRow)
		childFirst := r + 1
		childLast := r + len(g.children)

		_ = f.SetCellValue(sheet, cellName(1, sectionRow), g.stt)
		_ = f.MergeCell(sheet, cellName(2, sectionRow), cellName(3, sectionRow))
		_ = f.SetCellValue(sheet, cellName(2, sectionRow), g.label)
		_ = f.SetCellFormula(sheet, cellName(6, sectionRow), fmt.Sprintf("=SUM(%s:%s)", cellName(6, childFirst), cellName(6, childLast)))
		_ = f.SetCellFormula(sheet, cellName(7, sectionRow), fmt.Sprintf("=SUM(%s:%s)", cellName(7, childFirst), cellName(7, childLast)))
		styleKPIRow(f, sheet, sectionRow, true, st)
		_ = f.SetRowHeight(sheet, sectionRow, 18)
		r++

		for _, leaf := range g.children {
			_ = f.SetCellValue(sheet, cellName(1, r), leaf.stt)
			_ = f.MergeCell(sheet, cellName(2, r), cellName(3, r))
			_ = f.SetCellValue(sheet, cellName(2, r), leaf.label)
			if leaf.how != "" {
				_ = f.SetCellValue(sheet, cellName(4, r), leaf.how)
			}
			_ = f.SetCellValue(sheet, cellName(6, r), leaf.weight)
			_ = f.SetCellValue(sheet, cellName(7, r), leaf.result)
			if len(leaf.notes) > 0 {
				_ = f.SetCellValue(sheet, cellName(8, r), strings.Join(leaf.notes, "\n"))
			}
			styleKPIRow(f, sheet, r, false, st)

			height := 18.0
			if leaf.how != "" {
				height = 32
			}
			if nh := kpiNoteHeight(leaf.notes); len(leaf.notes) > 0 && nh > height {
				height = nh
			}
			_ = f.SetRowHeight(sheet, r, height)
			r++
		}
	}

	totalRow := r
	_ = f.MergeCell(sheet, cellName(1, totalRow), cellName(5, totalRow))
	_ = f.SetCellValue(sheet, cellName(1, totalRow), "TỔNG KPI")
	_ = f.SetCellFormula(sheet, cellName(6, totalRow), sumOfRows(6, sectionRows))
	_ = f.SetCellFormula(sheet, cellName(7, totalRow), sumOfRows(7, sectionRows))
	_ = f.SetCellStyle(sheet, cellName(1, totalRow), cellName(5, totalRow), st.totalLabel)
	_ = f.SetCellStyle(sheet, cellName(6, totalRow), cellName(6, totalRow), st.totalPctWeight)
	_ = f.SetCellStyle(sheet, cellName(7, totalRow), cellName(7, totalRow), st.totalPctResult)
	_ = f.SetCellStyle(sheet, cellName(8, totalRow), cellName(8, totalRow), st.totalEmpty)
	_ = f.SetRowHeight(sheet, totalRow, 20)

	now := time.Now().In(vietnamLocation())
	dateRow := totalRow + 2
	_ = f.MergeCell(sheet, cellName(6, dateRow), cellName(8, dateRow))
	_ = f.SetCellValue(sheet, cellName(6, dateRow), fmt.Sprintf("Ngày %d Tháng %d Năm %d", now.Day(), int(now.Month()), now.Year()))
	_ = f.SetCellStyle(sheet, cellName(6, dateRow), cellName(8, dateRow), st.dateCell)

	signRow := dateRow + 2
	signs := []struct {
		col   int
		label string
	}{{3, "Người thực hiện"}, {5, "Người kiểm soát"}, {7, "Người phê duyệt"}}
	for _, s := range signs {
		_ = f.MergeCell(sheet, cellName(s.col, signRow), cellName(s.col+1, signRow))
		_ = f.SetCellValue(sheet, cellName(s.col, signRow), s.label)
		_ = f.SetCellStyle(sheet, cellName(s.col, signRow), cellName(s.col+1, signRow), st.signLabel)
	}

	landscape := "landscape"
	fitWide, fitTall := 1, 0
	fit := true
	_ = f.SetSheetProps(sheet, &excelize.SheetPropsOptions{FitToPage: &fit})
	_ = f.SetPageLayout(sheet, &excelize.PageLayoutOptions{Orientation: &landscape, FitToWidth: &fitWide, FitToHeight: &fitTall})
	return nil
}

func kpiGroupsFor(k KPIMemberResponse, s KPISettingsResponse, rec *kpiMemberRecords) []kpiGroup {
	var fb, iv, ll []string
	if rec != nil {
		fb = feedbackNoteLines(rec.feedbacks)
		iv = initiativeNoteLines(rec.initiatives)
		ll = lateLogNoteLines(rec.lateLogs)
	}
	var attNotes []string
	if strings.TrimSpace(k.Attitude.Comment) != "" {
		attNotes = []string{k.Attitude.Comment}
	}
	return []kpiGroup{
		{"A", "ĐÁNH GIÁ TỪ KHÁCH HÀNG", []kpiLeaf{
			{1, "Khách hàng có phản hồi tích cực", "Từ cấp độ trưởng phòng, trong biên bản nghiệm thu", s.CustomerWeight * 0.7 / 100, k.Customer.KPI / 100, fb},
			{2, "Không phản hồi", "", s.CustomerWeight * 0.3 / 100, 0, nil},
			{3, "Claim", "", 0, 0, nil},
		}},
		{"B", "CÔNG VIỆC", []kpiLeaf{
			{1, "Tiến độ (Đúng hạn)", "Đo lường mức độ hoàn thành dựa trên khối lượng công việc.", s.ProgressWeight / 100, k.Progress.KPI / 100, nil},
			{2, "Chất lượng (Rework)", "Đo lường số khối lượng bị QA/Tester trả về.", s.QualityWeight / 100, k.Quality.KPI / 100, nil},
			{3, "Sáng kiến / Giải pháp", "Đếm số giải pháp tối ưu/tái cấu trúc được áp dụng.", s.InitiativeWeight / 100, k.Initiative.KPI / 100, iv},
		}},
		{"C", "KỶ LUẬT LAO ĐỘNG", []kpiLeaf{
			{1, "Đi sớm về muộn", "Theo bảng theo dõi hằng ngày.", s.PunctualityWeight / 100, k.Punctuality.KPI / 100, ll},
			{2, "Thái độ công việc, cố gắng", "Theo đánh giá của trưởng bộ phận.", s.AttitudeWeight / 100, k.Attitude.KPI / 100, attNotes},
		}},
	}
}

func sumOfRows(col int, rows []int) string {
	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		parts = append(parts, cellName(col, row))
	}
	return "=" + strings.Join(parts, "+")
}

type kpiStyles struct {
	title, formCode, infoLabel, infoValue int
	header                                int
	cellLeft, cellCenter                  int
	pctWeight, pctResult                  int
	secLeft, secCenter                    int
	secPctWeight, secPctResult            int
	totalLabel, totalPctWeight            int
	totalPctResult, totalEmpty            int
	dateCell, signLabel                   int
}

func newKPIStyles(f *excelize.File) (kpiStyles, error) {
	var st kpiStyles
	var firstErr error
	mk := func(s *excelize.Style) int {
		if firstErr != nil {
			return 0
		}
		id, err := f.NewStyle(s)
		if err != nil {
			firstErr = err
		}
		return id
	}

	border := kpiBorders()
	headerFill := excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"E9ECF6"}}
	sectionFill := excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F1F1F2"}}
	pct0, pct2 := "0%", "0.00%"

	alignLeft := &excelize.Alignment{Vertical: "center", Horizontal: "left", WrapText: true}
	alignCenter := &excelize.Alignment{Vertical: "center", Horizontal: "center", WrapText: true}

	st.title = mk(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 15}, Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"}})
	st.formCode = mk(&excelize.Style{Font: &excelize.Font{Italic: true, Size: 10, Color: "4B5563"}, Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"}})
	st.infoLabel = mk(&excelize.Style{Alignment: &excelize.Alignment{Vertical: "center"}})
	st.infoValue = mk(&excelize.Style{Font: &excelize.Font{Bold: true}, Alignment: &excelize.Alignment{Vertical: "center"}})

	st.header = mk(&excelize.Style{Font: &excelize.Font{Bold: true}, Alignment: alignCenter, Fill: headerFill, Border: border})

	st.cellLeft = mk(&excelize.Style{Alignment: alignLeft, Border: border})
	st.cellCenter = mk(&excelize.Style{Alignment: alignCenter, Border: border})
	st.pctWeight = mk(&excelize.Style{Alignment: alignCenter, Border: border, CustomNumFmt: &pct0})
	st.pctResult = mk(&excelize.Style{Alignment: alignCenter, Border: border, CustomNumFmt: &pct2})

	st.secLeft = mk(&excelize.Style{Font: &excelize.Font{Bold: true}, Alignment: alignLeft, Fill: sectionFill, Border: border})
	st.secCenter = mk(&excelize.Style{Font: &excelize.Font{Bold: true}, Alignment: alignCenter, Fill: sectionFill, Border: border})
	st.secPctWeight = mk(&excelize.Style{Font: &excelize.Font{Bold: true}, Alignment: alignCenter, Fill: sectionFill, Border: border, CustomNumFmt: &pct0})
	st.secPctResult = mk(&excelize.Style{Font: &excelize.Font{Bold: true}, Alignment: alignCenter, Fill: sectionFill, Border: border, CustomNumFmt: &pct2})

	st.totalLabel = mk(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 12}, Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"}, Fill: headerFill, Border: border})
	st.totalPctWeight = mk(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 12}, Alignment: alignCenter, Fill: headerFill, Border: border, CustomNumFmt: &pct0})
	st.totalPctResult = mk(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 12}, Alignment: alignCenter, Fill: headerFill, Border: border, CustomNumFmt: &pct2})
	st.totalEmpty = mk(&excelize.Style{Fill: headerFill, Border: border})

	st.dateCell = mk(&excelize.Style{Alignment: &excelize.Alignment{Horizontal: "center"}})
	st.signLabel = mk(&excelize.Style{Font: &excelize.Font{Bold: true}, Alignment: &excelize.Alignment{Horizontal: "center"}})

	return st, firstErr
}

func styleKPIRow(f *excelize.File, sheet string, row int, section bool, st kpiStyles) {
	a, left, pctW, pctR, h := st.cellCenter, st.cellLeft, st.pctWeight, st.pctResult, st.cellLeft
	if section {
		a, left, pctW, pctR, h = st.secCenter, st.secLeft, st.secPctWeight, st.secPctResult, st.secLeft
	}
	_ = f.SetCellStyle(sheet, cellName(1, row), cellName(1, row), a)
	_ = f.SetCellStyle(sheet, cellName(2, row), cellName(5, row), left)
	_ = f.SetCellStyle(sheet, cellName(6, row), cellName(6, row), pctW)
	_ = f.SetCellStyle(sheet, cellName(7, row), cellName(7, row), pctR)
	_ = f.SetCellStyle(sheet, cellName(8, row), cellName(8, row), h)
}

func kpiBorders() []excelize.Border {
	c := "9CA3AF"
	return []excelize.Border{
		{Type: "left", Color: c, Style: 1},
		{Type: "top", Color: c, Style: 1},
		{Type: "right", Color: c, Style: 1},
		{Type: "bottom", Color: c, Style: 1},
	}
}
