package chart

import (
	"fmt"
	"testing"
)

func TestPrintURLs(t *testing.T) {
	fmt.Println("URL_PIPELINE:", UpgradePipelineChart([]string{"Compute", "Networking", "Storage"}, []int{15, 8, 5}, []int{24, 12, 10}, []int{5, 2, 4}, []int{6, 1, 3}))
	fmt.Println("URL_FRUIT:", LowHangingFruitChart([]string{"Alpha Systems (Silver)", "Beta Tech (Gold)", "Gamma IT (Biz)"}, []float64{95000, 480000, 45000}, []float64{5000, 20000, 5000}))
	fmt.Println("URL_RISK:", RetentionRiskChart(25, 4, 12, 2))
	fmt.Println("URL_CONCENTRATION:", ConcentrationChart(8200000, 6500000, 4300000, "Compute"))
}
