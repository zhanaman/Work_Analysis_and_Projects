package parser

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"

	"github.com/anonimouskz/pbm-partner-bot/internal/domain"
	"github.com/xuri/excelize/v2"
)

// ExcelParser reads partner data from an Excel file using streaming.
type ExcelParser struct {
	filePath string
}

// NewExcelParser creates a new parser for the given file.
func NewExcelParser(filePath string) *ExcelParser {
	return &ExcelParser{filePath: filePath}
}

// Parse reads all sheets and extracts partner data.
// It uses streaming read to handle large files (96MB+) efficiently.
func (p *ExcelParser) Parse() ([]domain.Partner, error) {
	f, err := excelize.OpenFile(p.filePath, excelize.Options{
		UnzipSizeLimit: 512 << 20, // 512 MB unzip limit
	})
	if err != nil {
		return nil, fmt.Errorf("open excel file: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	slog.Info("found sheets", "count", len(sheets), "sheets", sheets)

	var allPartners []domain.Partner

	for _, sheet := range sheets {
		slog.Info("parsing sheet", "name", sheet)
		partners, err := p.parseSheet(f, sheet)
		if err != nil {
			slog.Warn("skipping sheet", "name", sheet, "error", err)
			continue
		}
		slog.Info("parsed sheet", "name", sheet, "partners", len(partners))
		allPartners = append(allPartners, partners...)
	}

	return allPartners, nil
}

// parseSheet parses a single sheet using streaming row iterator.
func (p *ExcelParser) parseSheet(f *excelize.File, sheet string) ([]domain.Partner, error) {
	rows, err := f.Rows(sheet)
	if err != nil {
		return nil, fmt.Errorf("create row iterator for sheet %q: %w", sheet, err)
	}
	defer rows.Close()

	// Read header row to determine column mapping
	if !rows.Next() {
		return nil, fmt.Errorf("sheet %q is empty", sheet)
	}

	headerRow, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("read header row: %w", err)
	}

	colMap := buildColumnMap(headerRow)
	slog.Debug("column mapping", "sheet", sheet, "columns", colMap)

	// Parse data rows
	var partners []domain.Partner
	rowNum := 1

	for rows.Next() {
		rowNum++
		cols, err := rows.Columns()
		if err != nil {
			slog.Warn("error reading row", "sheet", sheet, "row", rowNum, "error", err)
			continue
		}

		partner := mapRowToPartner(cols, colMap)
		if partner.Name == "" {
			continue // Skip rows without a partner name
		}

		partners = append(partners, partner)
	}

	return partners, nil
}

// columnMap maps semantic field names to column indices.
type columnMap map[string]int

// buildColumnMap creates a mapping from header names to column indices.
// This is flexible and handles multiple naming conventions.
func buildColumnMap(headers []string) columnMap {
	cm := make(columnMap)

	for i, h := range headers {
		h = strings.TrimSpace(strings.ToLower(h))

		switch {
		// Partner name
		case contains(h, "partner name", "company name", "partner", "name"):
			cm["name"] = i

		// Partner ID
		case contains(h, "partner id", "hpe id", "said", "account id"):
			cm["partner_id"] = i

		// Tier
		case contains(h, "tier", "level", "status"):
			cm["tier"] = i

		// Country
		case contains(h, "country"):
			cm["country"] = i

		// City
		case contains(h, "city"):
			cm["city"] = i

		// Certifications
		case contains(h, "compute"):
			cm["compute_cert"] = i
		case contains(h, "networking", "network"):
			cm["networking_cert"] = i
		case contains(h, "hybrid cloud", "hc cert", "hybrid"):
			cm["hybrid_cloud_cert"] = i
		case contains(h, "storage"):
			cm["storage_cert"] = i

		// Revenue
		case contains(h, "revenue", "ytd", "amount"):
			cm["revenue_ytd"] = i

		// Target
		case contains(h, "target", "goal"):
			cm["target"] = i

		// Contact
		case contains(h, "contact name", "contact person"):
			cm["contact_name"] = i
		case contains(h, "email", "e-mail"):
			cm["contact_email"] = i
		case contains(h, "phone", "tel"):
			cm["contact_phone"] = i
		}
	}

	return cm
}

// mapRowToPartner maps a row of strings to a Partner using the column map.
func mapRowToPartner(row []string, cm columnMap) domain.Partner {
	get := func(key string) string {
		idx, ok := cm[key]
		if !ok || idx >= len(row) {
			return ""
		}
		return strings.TrimSpace(row[idx])
	}

	getFloat := func(key string) float64 {
		s := get(key)
		if s == "" {
			return 0
		}
		// Remove currency symbols, commas, spaces
		s = strings.NewReplacer("$", "", ",", "", " ", "", "€", "").Replace(s)
		v, _ := strconv.ParseFloat(s, 64)
		return v
	}

	return domain.Partner{
		Name:           get("name"),
		PartnerID:      get("partner_id"),
		Tier:           get("tier"),
		Country:        get("country"),
		City:           get("city"),
		ComputeCert:    get("compute_cert"),
		NetworkingCert: get("networking_cert"),
		HybridCloud:    get("hybrid_cloud_cert"),
		StorageCert:    get("storage_cert"),
		RevenueYTD:     getFloat("revenue_ytd"),
		Target:         getFloat("target"),
		ContactName:    get("contact_name"),
		ContactEmail:   get("contact_email"),
		ContactPhone:   get("contact_phone"),
	}
}

func contains(s string, patterns ...string) bool {
	for _, p := range patterns {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}
