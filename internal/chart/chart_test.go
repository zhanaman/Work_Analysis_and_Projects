package chart

import (
	"strings"
	"testing"
)

func TestTierDoughnutURL(t *testing.T) {
	dist := map[string]int{
		"platinum": 12,
		"gold":     28,
		"silver":   45,
		"business": 57,
	}

	url := TierDoughnutURL(dist, "Compute")

	if !strings.HasPrefix(url, "https://quickchart.io/chart?") {
		t.Errorf("URL should start with quickchart.io, got: %s", url[:50])
	}
	if !strings.Contains(url, "doughnut") {
		t.Error("URL should contain 'doughnut' chart type")
	}
	if !strings.Contains(url, "Compute") {
		t.Error("URL should contain center name 'Compute'")
	}
}

func TestCountryStackedBarURL(t *testing.T) {
	countries := []string{"Kazakhstan", "Uzbekistan", "Azerbaijan"}
	plat := []int{5, 2, 3}
	gold := []int{10, 5, 8}
	silver := []int{15, 10, 12}
	biz := []int{20, 15, 10}

	url := CountryStackedBarURL(countries, plat, gold, silver, biz)

	if !strings.HasPrefix(url, "https://quickchart.io/chart?") {
		t.Errorf("URL should start with quickchart.io, got: %s", url[:50])
	}
	if !strings.Contains(url, "bar") {
		t.Error("URL should contain 'bar' chart type")
	}
	if !strings.Contains(url, "Kazakhstan") {
		t.Error("URL should contain country name")
	}
}

func TestVolumeTopBarURL(t *testing.T) {
	names := []string{"Partner A", "Partner B"}
	volumes := []float64{1500000, 800000}

	url := VolumeTopBarURL(names, volumes)

	if !strings.HasPrefix(url, "https://quickchart.io/chart?") {
		t.Errorf("URL should start with quickchart.io, got: %s", url[:50])
	}
	if !strings.Contains(url, "bar") {
		t.Error("URL should contain 'bar' chart type")
	}
}

func TestToJSArray(t *testing.T) {
	got := toJSArray([]string{"a", "b", "c"})
	want := "['a','b','c']"
	if got != want {
		t.Errorf("toJSArray = %q, want %q", got, want)
	}
}

func TestToIntArray(t *testing.T) {
	got := toIntArray([]int{1, 2, 3})
	want := "[1,2,3]"
	if got != want {
		t.Errorf("toIntArray = %q, want %q", got, want)
	}
}

func TestEmptyDistribution(t *testing.T) {
	dist := map[string]int{}
	url := TierDoughnutURL(dist, "Compute")

	// Should still produce a valid URL even with empty data
	if !strings.HasPrefix(url, "https://quickchart.io/chart?") {
		t.Errorf("Empty dist should still produce valid URL, got: %s", url[:50])
	}
}
