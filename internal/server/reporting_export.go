package server

import (
	"bytes"
	"fmt"
	"mime"
	"net/http"
	"strings"

	"github.com/phpdave11/gofpdf"
	"github.com/xuri/excelize/v2"

	"github.com/Volkov-D-A/docs-register-and-track/internal/models"
	"github.com/Volkov-D-A/docs-register-and-track/internal/server/reportassets"
)

func (api *managementAPI) exportReport(w http.ResponseWriter, r *http.Request) {
	var req models.ReportRequest
	if err := decodeJSON(r, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	result, err := api.reporting.Run(r.Context(), req)
	if err != nil {
		writeUserError(w, err)
		return
	}
	var body []byte
	var contentType, extension string
	switch r.PathValue("format") {
	case "xlsx":
		body, err = reportXLSX(result)
		contentType, extension = "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "xlsx"
	case "pdf":
		body, err = reportPDF(result)
		contentType, extension = "application/pdf", "pdf"
	default:
		writeUserError(w, models.NewBadRequest("неизвестный формат отчёта"))
		return
	}
	if err != nil {
		writeUserError(w, err)
		return
	}
	if len(body) > 50<<20 {
		writeUserError(w, models.NewBadRequest("отчёт превышает 50 МБ"))
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": fmt.Sprintf("report-%s-%s.%s", result.Template, result.EndDate, extension)}))
	w.Header().Set("Content-Length", fmt.Sprint(len(body)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func reportTitle(template string) string {
	switch template {
	case "organizations":
		return "Письма по организациям"
	case "department_load":
		return "Нагрузка подразделений"
	case "execution_time":
		return "Среднее время исполнения"
	case "overdue_rate":
		return "Доля просрочки"
	default:
		return "Отчёт"
	}
}

func reportHeaders(template string) []string {
	switch template {
	case "organizations":
		return []string{"Организация", "Входящих", "Исходящих"}
	case "department_load":
		return []string{"Период", "Подразделение", "Поручений"}
	case "execution_time":
		return []string{"Подразделение", "Поручений", "Среднее, суток"}
	default:
		return []string{"Подразделение", "Поручений со сроком", "С просрочкой", "Доля, %"}
	}
}

func reportValues(template string, row models.ReportRow) []any {
	switch template {
	case "organizations":
		return []any{row.Name, row.Incoming, row.Outgoing}
	case "department_load":
		return []any{row.Period, row.Name, row.Count}
	case "execution_time":
		return []any{row.Name, row.Count, row.Metric}
	default:
		return []any{row.Name, row.Count, row.Overdue, row.Metric}
	}
}

func safeSpreadsheetText(value string) string {
	if value != "" && strings.ContainsRune("=+-@\t\r", rune(value[0])) {
		return "'" + value
	}
	return value
}

func reportXLSX(result *models.ReportResult) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()
	const sheet = "Отчёт"
	f.SetSheetName("Sheet1", sheet)
	metadata := []string{reportTitle(result.Template), "Период: " + result.StartDate + " — " + result.EndDate,
		"Сформировано: " + result.GeneratedAt + "; часовой пояс: " + result.Timezone, result.Definition + " (версия " + fmt.Sprint(result.Version) + ")", result.Filters}
	for i, value := range metadata {
		_ = f.SetCellStr(sheet, fmt.Sprintf("A%d", i+1), value)
	}
	_ = f.SetCellStr(sheet, "A6", reportSummaryText(result))
	for col, label := range reportHeaders(result.Template) {
		cell, _ := excelize.CoordinatesToCellName(col+1, 7)
		_ = f.SetCellStr(sheet, cell, label)
	}
	for i, row := range result.Rows {
		for col, value := range reportValues(result.Template, row) {
			cell, _ := excelize.CoordinatesToCellName(col+1, i+8)
			if label, ok := value.(string); ok {
				value = safeSpreadsheetText(label)
			}
			if err := f.SetCellValue(sheet, cell, value); err != nil {
				return nil, err
			}
		}
	}
	_ = f.SetColWidth(sheet, "A", "A", 34)
	_ = f.SetColWidth(sheet, "B", "D", 20)
	_ = f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 7, TopLeftCell: "A8", ActivePane: "bottomLeft"})
	var out bytes.Buffer
	if _, err := f.WriteTo(&out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func reportPDF(result *models.ReportResult) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(12, 12, 12)
	pdf.SetAutoPageBreak(true, 16)
	pdf.AddUTF8FontFromBytes("DejaVu", "", reportassets.DejaVuSans)
	pdf.AddPage()
	pdf.SetFont("DejaVu", "", 15)
	pdf.MultiCell(186, 9, reportTitle(result.Template), "", "L", false)
	pdf.SetFont("DejaVu", "", 9)
	pdf.MultiCell(186, 6, "Период: "+result.StartDate+" — "+result.EndDate+"  •  Сформировано: "+result.GeneratedAt+"; часовой пояс: "+result.Timezone, "", "L", false)
	pdf.MultiCell(186, 6, result.Definition+" (версия "+fmt.Sprint(result.Version)+")", "", "L", false)
	pdf.MultiCell(186, 6, result.Filters, "", "L", false)
	pdf.MultiCell(186, 6, reportSummaryText(result), "", "L", false)
	pdf.Ln(3)
	headers := reportHeaders(result.Template)
	widths := reportPDFWidths(result.Template)
	rowHeight := func(values []string) float64 {
		height := 8.0
		for i, value := range values {
			lines := pdf.SplitText(value, widths[i]-4)
			if h := float64(len(lines))*5 + 2; h > height {
				height = h
			}
		}
		return height
	}
	writeRow := func(values []string) {
		height := rowHeight(values)
		x, y := pdf.GetX(), pdf.GetY()
		for i, value := range values {
			pdf.Rect(x, y, widths[i], height, "D")
			pdf.SetXY(x+2, y+1)
			pdf.MultiCell(widths[i]-4, 5, value, "", "L", false)
			x += widths[i]
		}
		pdf.SetXY(12, y+height)
	}
	writeRow(headers)
	for _, row := range result.Rows {
		values := make([]string, 0, len(headers))
		for _, value := range reportValues(result.Template, row) {
			values = append(values, fmt.Sprint(value))
		}
		if pdf.GetY()+rowHeight(values) > 278 {
			pdf.AddPage()
			writeRow(headers)
		}
		writeRow(values)
	}
	if len(result.Rows) == 0 {
		pdf.CellFormat(186, 8, "Нет данных за выбранный период", "", 0, "L", false, 0, "")
	}
	var out bytes.Buffer
	if err := pdf.Output(&out); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func reportPDFWidths(template string) []float64 {
	switch template {
	case "organizations":
		return []float64{90, 48, 48}
	case "department_load":
		return []float64{32, 106, 48}
	case "execution_time":
		return []float64{90, 48, 48}
	default:
		return []float64{75, 42, 33, 36}
	}
}

func reportSummaryText(result *models.ReportResult) string {
	s := result.Summary
	switch result.Template {
	case "organizations":
		return fmt.Sprintf("Итого по организациям: входящих %d, исходящих %d; уникальных документов %d", s.Incoming, s.Outgoing, s.UniqueDocuments)
	case "department_load":
		return fmt.Sprintf("Всего поручений: %d", s.Count)
	case "execution_time":
		if s.Count == 0 {
			return "Поручений нет; среднее время: нет данных"
		}
		return fmt.Sprintf("Всего поручений: %d; среднее время: %.1f суток", s.Count, s.Metric)
	default:
		if s.Count == 0 {
			return fmt.Sprintf("Поручений со сроком нет; доля: нет данных; открытая просрочка: %d", s.OpenOverdue)
		}
		return fmt.Sprintf("Поручений со сроком: %d; просрочено: %d; доля: %.1f %%; открытая просрочка: %d", s.Count, s.Overdue, s.Metric, s.OpenOverdue)
	}
}
