package parser

import (
	"strconv"
	"strings"
	"time"
)

// parseBool parses "Yes"/"No" and other boolean representations.
func parseBool(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	return s == "yes" || s == "true" || s == "1"
}

// parseMoney parses "$1,234,567" into 1234567.0.
func parseMoney(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "(empty)" || s == "N/A" {
		return 0
	}
	s = strings.NewReplacer("$", "", ",", "", " ", "", "€", "").Replace(s)
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// parsePct parses "91%" into 91.0.
func parsePct(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "(empty)" || s == "N/A" {
		return 0
	}
	s = strings.TrimSuffix(s, "%")
	s = strings.ReplaceAll(s, ",", "")
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// parseFloat parses a generic float string.
func parseFloat(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "(empty)" || s == "N/A" {
		return 0
	}
	s = strings.ReplaceAll(s, ",", "")
	v, _ := strconv.ParseFloat(s, 64)
	return v
}

// parseInt parses an integer string.
func parseInt(s string) int {
	s = strings.TrimSpace(s)
	if s == "" || s == "(empty)" || s == "N/A" {
		return 0
	}
	v, _ := strconv.Atoi(s)
	return v
}

// IsCCACountry returns true if the country is in the CCA region.
func IsCCACountry(country string) bool {
	country = strings.TrimSpace(country)
	switch country {
	case "Kazakhstan",
		"Azerbaijan",
		"Uzbekistan",
		"Kyrgyzstan",
		"Turkmenistan",
		"Georgia",
		"Armenia",
		"Tajikistan":
		return true
	}
	return false
}

// IsCCAByOrg checks if the partner belongs to CCA by HPE Organization.
func IsCCAByOrg(hpeOrg string) bool {
	hpeOrg = strings.TrimSpace(hpeOrg)
	return hpeOrg == "RMC" || hpeOrg == "CCA"
}

// parseRefreshDate normalizes various date formats from Excel "Refresh_date" columns.
// Known formats: "02-23-26", "2/23/26 06:18", "2026-02-23"
// Returns YYYY-MM-DD string, or empty string if unparseable.
func parseRefreshDate(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	// Try multiple formats
	layouts := []string{
		"01-02-06",       // MM-DD-YY
		"1/2/06 15:04",   // M/D/YY HH:MM
		"1/2/06",         // M/D/YY
		"2006-01-02",     // YYYY-MM-DD (ISO)
		"01/02/2006",     // MM/DD/YYYY
		"01-02-2006",     // MM-DD-YYYY
	}

	for _, layout := range layouts {
		t, err := time.Parse(layout, raw)
		if err == nil {
			// Normalize 2-digit year: 00-79 -> 2000s, 80-99 -> 1900s (Go default)
			return t.Format("2006-01-02")
		}
	}
	return ""
}
