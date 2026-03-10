package parser

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/anonimouskz/pbm-partner-bot/internal/domain"
	"github.com/xuri/excelize/v2"
)

// ParseResult holds the result of parsing the Excel file.
type ParseResult struct {
	Partners     map[string]*domain.Partner // keyed by party_id
	TotalRows    int
	CCARows      int
	SkippedRows  int
	SheetsParsed []string
	RefreshDate  string // extracted from "Refresh_date" column in Excel
	Duration     time.Duration
	Diagnostics  []string // warnings about missing columns/sheets
}

// ExcelParser reads partner data from an HPE Excel file.
type ExcelParser struct {
	filePath  string
	filterCCA bool // if true, only include CCA partners
	configs   []CenterConfig
}

// NewExcelParser creates a new parser for the given file.
func NewExcelParser(filePath string, filterCCA bool) *ExcelParser {
	return &ExcelParser{
		filePath:  filePath,
		filterCCA: filterCCA,
		configs:   CenterConfigs,
	}
}

// Parse reads all center sheets and extracts partner data.
func (p *ExcelParser) Parse() (*ParseResult, error) {
	start := time.Now()

	f, err := excelize.OpenFile(p.filePath, excelize.Options{
		UnzipSizeLimit:    2 << 30, // 2 GB (88MB file decompresses to ~1.5GB)
		UnzipXMLSizeLimit: 1 << 30, // 1 GB
	})
	if err != nil {
		return nil, fmt.Errorf("open excel file: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	slog.Info("opened excel file", "sheets", len(sheets), "file", p.filePath)

	result := &ParseResult{
		Partners: make(map[string]*domain.Partner),
	}

	for _, cfg := range p.configs {
		// Verify sheet exists
		found := false
		for _, s := range sheets {
			if s == cfg.SheetName {
				found = true
				break
			}
		}
		if !found {
			diag := fmt.Sprintf("⚠️  ЛИСТ НЕ НАЙДЕН: %q — ожидался, но отсутствует в файле. Имеющиеся листы: %v\n   FIX: проверьте что лист %q не переименован в новом файле", cfg.SheetName, sheets, cfg.SheetName)
			result.Diagnostics = append(result.Diagnostics, diag)
			slog.Warn("sheet not found, skipping", "sheet", cfg.SheetName, "available", sheets)
			continue
		}

		parsed, err := p.parseSheet(f, cfg)
		if err != nil {
			slog.Error("error parsing sheet", "sheet", cfg.SheetName, "error", err)
			continue
		}

		result.SheetsParsed = append(result.SheetsParsed, cfg.SheetName)
		result.TotalRows += parsed.totalRows
		result.CCARows += parsed.ccaRows
		result.SkippedRows += parsed.skippedRows
		result.Diagnostics = append(result.Diagnostics, parsed.diagnostics...)

		// Take refresh date from the first sheet that has it
		if result.RefreshDate == "" && parsed.refreshDate != "" {
			result.RefreshDate = parsed.refreshDate
		}

		// Merge parsed data into result
		for partyID, data := range parsed.partners {
			existing, ok := result.Partners[partyID]
			if !ok {
				result.Partners[partyID] = data
			} else {
				// Merge tier data from this center into existing partner
				existing.Tiers = append(existing.Tiers, data.Tiers...)
				if len(data.Competencies) > 0 {
					existing.Competencies = data.Competencies // competencies come from any sheet
				}
				if len(data.CompLevels) > 0 {
					existing.CompLevels = append(existing.CompLevels, data.CompLevels...)
				}
			}
		}

		slog.Info("parsed sheet",
			"sheet", cfg.SheetName,
			"total", parsed.totalRows,
			"cca", parsed.ccaRows,
			"skipped", parsed.skippedRows,
		)
	}

	result.Duration = time.Since(start)
	slog.Info("parsing complete",
		"partners", len(result.Partners),
		"total_rows", result.TotalRows,
		"cca_rows", result.CCARows,
		"duration", result.Duration,
	)

	return result, nil
}

// sheetParseResult holds per-sheet parsing results.
type sheetParseResult struct {
	partners    map[string]*domain.Partner
	totalRows   int
	ccaRows     int
	skippedRows int
	refreshDate string
	diagnostics []string
}

// parseSheet parses a single center sheet.
func (p *ExcelParser) parseSheet(f *excelize.File, cfg CenterConfig) (*sheetParseResult, error) {
	rows, err := f.Rows(cfg.SheetName)
	if err != nil {
		return nil, fmt.Errorf("create row iterator for sheet %q: %w", cfg.SheetName, err)
	}
	defer rows.Close()

	// Read header row (Row 1 or Row 2 depending on center)
	var headerRow []string
	rowNum := 0

	for rows.Next() {
		rowNum++
		cols, err := rows.Columns()
		if err != nil {
			return nil, fmt.Errorf("read row %d: %w", rowNum, err)
		}

		if rowNum == cfg.HeaderRow {
			headerRow = cols
			break
		}
	}

	if len(headerRow) == 0 {
		return nil, fmt.Errorf("no header row found at row %d in sheet %q", cfg.HeaderRow, cfg.SheetName)
	}

	cm := buildColumnMap(headerRow)
	slog.Debug("column map built", "sheet", cfg.SheetName, "columns", len(cm)/2) // /2 because we store both cases

	// Validate critical columns exist
	diags := validateColumns(cm, cfg)

	result := &sheetParseResult{
		partners:    make(map[string]*domain.Partner),
		diagnostics: diags,
	}

	// Extract refresh date from header columns
	for _, name := range []string{"Refresh_date", "Refresh date", "Refreshed date"} {
		if idx, ok := cm[name]; ok {
			_ = idx // we'll read it from first data row below
			result.refreshDate = "__pending:" + name
			break
		}
	}

	// Parse data rows
	for rows.Next() {
		rowNum++
		cols, err := rows.Columns()
		if err != nil {
			slog.Warn("error reading row", "sheet", cfg.SheetName, "row", rowNum, "error", err)
			continue
		}

		result.totalRows++

		// Extract partner identity
		partner := mapPartnerFromRow(cols, cm)
		if partner.PartyID == "" || partner.Name == "" {
			result.skippedRows++
			continue
		}

		// CCA filter
		if p.filterCCA {
			if !IsCCACountry(partner.Country) && !IsCCAByOrg(partner.HPEOrg) {
				result.skippedRows++
				continue
			}
		}

		result.ccaRows++

		// Extract refresh date from first data row
		if strings.HasPrefix(result.refreshDate, "__pending:") {
			colName := strings.TrimPrefix(result.refreshDate, "__pending:")
			raw := cm.get(cols, colName)
			result.refreshDate = parseRefreshDate(raw)
		}

		// Get or create partner in result map
		existing, ok := result.partners[partner.PartyID]
		if !ok {
			existing = &partner
			result.partners[partner.PartyID] = existing
		}

		// Extract tier data for all 4 tiers in this center
		tiers := []domain.Tier{
			domain.TierBusiness,
			domain.TierSilver,
			domain.TierGold,
			domain.TierPlatinum,
		}

		for _, tier := range tiers {
			tierData := mapTierFromRow(cols, cm, cfg.Prefix, tier)
			// Only add tier if it has meaningful data (at least threshold or criteria)
			if tierData.Threshold > 0 || tierData.CriteriaMet || tierData.VolumeActuals > 0 {
				existing.Tiers = append(existing.Tiers, tierData)
			}
		}

		// Extract competencies (same across centers, so only from Compute/HC sheets)
		if cfg.Center == domain.CenterCompute || cfg.Center == domain.CenterHybridCloud {
			comps := mapCompetenciesFromRow(cols, cm)
			if len(comps) > 0 {
				existing.Competencies = comps
			}
		}

		// Extract quarterly comp levels
		compLevels := mapCompLevelsFromRow(cols, cm, cfg.Prefix)
		if len(compLevels) > 0 {
			existing.CompLevels = append(existing.CompLevels, compLevels...)
		}
	}

	return result, nil
}

// GetCenterPrefixFromMembership maps a membership string to its center.
func GetCenterPrefixFromMembership(membership string) string {
	m := strings.ToLower(membership)
	switch {
	case strings.Contains(m, "compute"):
		return "Compute"
	case strings.Contains(m, "hybrid") || strings.Contains(m, "cloud"):
		return "Hybrid Cloud"
	case strings.Contains(m, "networking") || strings.Contains(m, "network"):
		return "Networking"
	default:
		return ""
	}
}
