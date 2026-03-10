package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/anonimouskz/pbm-partner-bot/internal/parser"
	"github.com/anonimouskz/pbm-partner-bot/internal/storage"
)

func main() {
	filePath := flag.String("file", "", "Path to HPE Excel file")
	dsn := flag.String("dsn", "", "PostgreSQL DSN (or set POSTGRES_DSN env)")
	filterCCA := flag.Bool("cca", true, "Filter to CCA region only")
	dryRun := flag.Bool("dry-run", false, "Parse only, don't write to DB")
	flag.Parse()

	if *filePath == "" {
		fmt.Fprintln(os.Stderr, "Usage: import -file <path.xlsx> [-dsn <postgres_dsn>] [-cca=true] [-dry-run]")
		os.Exit(1)
	}

	if *dsn == "" {
		*dsn = os.Getenv("POSTGRES_DSN")
	}
	if *dsn == "" && !*dryRun {
		*dsn = "postgres://pbm:pbm_secret@localhost:5433/pbm_partners?sslmode=disable"
	}

	// Setup logging
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	slog.SetDefault(slog.New(handler))

	slog.Info("starting import",
		"file", *filePath,
		"cca_filter", *filterCCA,
		"dry_run", *dryRun,
	)

	// Parse Excel
	p := parser.NewExcelParser(*filePath, *filterCCA)
	result, err := p.Parse()
	if err != nil {
		slog.Error("parse failed", "error", err)
		os.Exit(1)
	}

	slog.Info("parsing complete",
		"partners", len(result.Partners),
		"total_rows", result.TotalRows,
		"cca_rows", result.CCARows,
		"skipped_rows", result.SkippedRows,
		"sheets", result.SheetsParsed,
		"duration", result.Duration,
	)

	// Print diagnostics (missing columns/sheets)
	if len(result.Diagnostics) > 0 {
		fmt.Fprintf(os.Stderr, "\n╔══════════════════════════════════════════════════════╗\n")
		fmt.Fprintf(os.Stderr, "║  ⚠️  ДИАГНОСТИКА EXCEL: %d проблем(а)                ║\n", len(result.Diagnostics))
		fmt.Fprintf(os.Stderr, "╚══════════════════════════════════════════════════════╝\n\n")
		for i, d := range result.Diagnostics {
			fmt.Fprintf(os.Stderr, "%d. %s\n\n", i+1, d)
		}
		fmt.Fprintf(os.Stderr, "────────────────────────────────────────────────────────\n")
		fmt.Fprintf(os.Stderr, "Если колонки переименованы — обновите mapper.go\n")
		fmt.Fprintf(os.Stderr, "Если это новый FY — обновите кварталы в mapCompLevelsFromRow()\n\n")
	}

	// Critical check: no partners at all
	if len(result.Partners) == 0 {
		slog.Error("CRITICAL: 0 партнёров после парсинга — структура Excel вероятно изменилась")
		fmt.Fprintf(os.Stderr, "❌ КРИТИЧЕСКАЯ ОШИБКА: 0 партнёров. Импорт остановлен.\n")
		os.Exit(1)
	}

	// Print sample partners
	count := 0
	for _, partner := range result.Partners {
		if count >= 5 {
			break
		}
		tierInfo := ""
		for _, t := range partner.Tiers {
			if t.CriteriaMet {
				tierInfo += fmt.Sprintf(" %s:%s✅", t.Center, t.Tier)
			}
		}
		slog.Info("sample partner",
			"name", partner.Name,
			"country", partner.Country,
			"compute", partner.MembershipCompute,
			"hc", partner.MembershipHC,
			"networking", partner.MembershipNetworking,
			"tiers", len(partner.Tiers),
			"comps", len(partner.Competencies),
			"met", tierInfo,
		)
		count++
	}

	if *dryRun {
		slog.Info("dry run complete — no data written to DB")
		return
	}

	// Connect to DB
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	db, err := storage.NewPostgres(ctx, *dsn)
	if err != nil {
		slog.Error("DB connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// Upsert partners
	repo := storage.NewPartnerRepo(db)
	inserted, updated, err := repo.UpsertFromParsed(ctx, result.Partners)
	if err != nil {
		slog.Error("upsert failed", "error", err)
		os.Exit(1)
	}

	slog.Info("import complete",
		"inserted", inserted,
		"updated", updated,
	)

	// Log the import
	slog.Info("data refresh date", "date", result.RefreshDate)
	err = repo.LogImport(ctx, *filePath, result.RefreshDate, result.SheetsParsed,
		len(result.Partners), result.CCARows,
		int(result.Duration.Milliseconds()))
	if err != nil {
		slog.Warn("failed to log import", "error", err)
	}

	// Verify
	total, err := repo.CountAll(ctx)
	if err != nil {
		slog.Warn("count failed", "error", err)
	} else {
		slog.Info("verification", "total_partners_in_db", total)
	}
}
