package parser

import (
	"testing"
)

func TestParseBool(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"Yes", true},
		{"No", false},
		{"yes", true},
		{"no", false},
		{"", false},
		{"True", true},
		{"False", false},
		{"1", true},
		{"0", false},
	}

	for _, tt := range tests {
		got := parseBool(tt.input)
		if got != tt.want {
			t.Errorf("parseBool(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseMoney(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"$1,234,567", 1234567},
		{"$90,000", 90000},
		{"$0", 0},
		{"", 0},
		{"(empty)", 0},
		{"N/A", 0},
		{"$555,000", 555000},
		{"$2,500,000", 2500000},
	}

	for _, tt := range tests {
		got := parseMoney(tt.input)
		if got != tt.want {
			t.Errorf("parseMoney(%q) = %f, want %f", tt.input, got, tt.want)
		}
	}
}

func TestParsePct(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"91%", 91},
		{"0%", 0},
		{"100%", 100},
		{"2508%", 2508},
		{"", 0},
		{"N/A", 0},
	}

	for _, tt := range tests {
		got := parsePct(tt.input)
		if got != tt.want {
			t.Errorf("parsePct(%q) = %f, want %f", tt.input, got, tt.want)
		}
	}
}

func TestIsCCACountry(t *testing.T) {
	tests := []struct {
		country string
		want    bool
	}{
		{"Kazakhstan", true},
		{"Azerbaijan", true},
		{"Uzbekistan", true},
		{"Georgia", true},
		{"Armenia", true},
		{"Czech Republic", false},
		{"South Africa", false},
		{"", false},
	}

	for _, tt := range tests {
		got := IsCCACountry(tt.country)
		if got != tt.want {
			t.Errorf("IsCCACountry(%q) = %v, want %v", tt.country, got, tt.want)
		}
	}
}

func TestBuildColumnMap(t *testing.T) {
	headers := []string{"Theater", "Sub-Region", "", "Country", "Party Country Code"}
	cm := buildColumnMap(headers)

	if cm["Theater"] != 0 {
		t.Errorf("Theater should be at index 0, got %d", cm["Theater"])
	}
	if cm["Country"] != 3 {
		t.Errorf("Country should be at index 3, got %d", cm["Country"])
	}

	// Case-insensitive lookup
	if cm["theater"] != 0 {
		t.Errorf("theater (lowercase) should be at index 0, got %d", cm["theater"])
	}

	// Empty column should not be in map
	if _, ok := cm[""]; ok {
		t.Error("empty column should not be in map")
	}
}

func TestColumnMapGet(t *testing.T) {
	headers := []string{"Theater", "Sub-Region", "HPE Organization", "Country"}
	cm := buildColumnMap(headers)

	row := []string{"EMEA", "East Central", "CCA", "Kazakhstan"}
	
	if v := cm.get(row, "Theater"); v != "EMEA" {
		t.Errorf("get Theater = %q, want EMEA", v)
	}
	if v := cm.get(row, "Country"); v != "Kazakhstan" {
		t.Errorf("get Country = %q, want Kazakhstan", v)
	}
	if v := cm.get(row, "NonExistent"); v != "" {
		t.Errorf("get NonExistent = %q, want empty", v)
	}
}
