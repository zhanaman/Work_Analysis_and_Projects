package chart

import (
	"strings"
	"testing"
)

func TestUpgradePipelineChart(t *testing.T) {
	centers := []string{"Compute", "Networking"}
	ready := []int{5, 2}
	cert := []int{10, 8}
	vol := []int{3, 1}
	deep := []int{2, 0}

	url := UpgradePipelineChart(centers, ready, cert, vol, deep)

	if !strings.HasPrefix(url, quickChartBaseURL) {
		t.Errorf("Expected URL to start with %s, got %s", quickChartBaseURL, url)
	}

	if !strings.Contains(url, "Compute") || !strings.Contains(url, "Networking") {
		t.Errorf("Expected URL to contain center labels, got %s", url)
	}
	
	if !strings.Contains(url, "10") || !strings.Contains(url, "8") {
		t.Errorf("Expected URL to contain data points, got %s", url)
	}
}

func TestLowHangingFruitChart(t *testing.T) {
	names := []string{"Partner A", "Partner B"}
	volumes := []float64{950000, 480000}
	gaps := []float64{50000, 20000}

	url := LowHangingFruitChart(names, volumes, gaps)

	if !strings.HasPrefix(url, quickChartBaseURL) {
		t.Errorf("Expected URL to start with %s, got %s", quickChartBaseURL, url)
	}

	if !strings.Contains(url, "Partner+A") || !strings.Contains(url, "Partner+B") {
		t.Errorf("Expected URL to contain partner names, got %s", url)
	}
	
	if !strings.Contains(url, "950000") || !strings.Contains(url, "50000") {
		t.Errorf("Expected URL to contain volume/gap data, got %s", url)
	}
}

func TestRetentionRiskChart(t *testing.T) {
	url := RetentionRiskChart(20, 5, 10, 2)

	if !strings.HasPrefix(url, quickChartBaseURL) {
		t.Errorf("Expected URL to start with %s, got %s", quickChartBaseURL, url)
	}

	if !strings.Contains(url, "20") || !strings.Contains(url, "10") {
		t.Errorf("Expected URL to contain risk data, got %s", url)
	}

	if !strings.Contains(url, "Retention+Risk") {
		t.Errorf("Expected URL to contain title, got %s", url)
	}
}

func TestConcentrationChart(t *testing.T) {
	url := ConcentrationChart(1000000, 2500000, 5000000, "Storage")

	if !strings.HasPrefix(url, quickChartBaseURL) {
		t.Errorf("Expected URL to start with %s, got %s", quickChartBaseURL, url)
	}

	if !strings.Contains(url, "Storage") {
		t.Errorf("Expected URL to contain center name, got %s", url)
	}

	if !strings.Contains(url, "1000000.000000") && !strings.Contains(url, "1000000") {
		t.Errorf("Expected URL to contain volume data, got %s", url)
	}
}
