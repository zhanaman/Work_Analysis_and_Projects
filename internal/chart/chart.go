package chart

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	quickChartBaseURL = "https://quickchart.io/chart"
	chartWidth        = 600
	chartHeight       = 400
	bgColor           = "rgb(30,41,59)"    // slate-800
	textColor         = "rgb(248,250,252)" // slate-50
	gridColor         = "rgba(148,163,184,0.2)"
)

// Tier color palette
var tierColors = map[string]string{
	"platinum": "rgb(124,58,237)", // purple-600
	"gold":     "rgb(245,158,11)", // amber-500
	"silver":   "rgb(156,163,175)", // gray-400
	"business": "rgb(209,213,219)", // gray-300
}

// TierDoughnutURL builds a quickchart.io URL for a tier distribution doughnut chart.
func TierDoughnutURL(dist map[string]int, centerName string) string {
	plat := dist["platinum"]
	gold := dist["gold"]
	silver := dist["silver"]
	biz := dist["business"]

	config := fmt.Sprintf(`{
		type:'doughnut',
		data:{
			labels:['Platinum (%d)','Gold (%d)','Silver (%d)','Business (%d)'],
			datasets:[{
				data:[%d,%d,%d,%d],
				backgroundColor:['%s','%s','%s','%s'],
				borderWidth:2,
				borderColor:'%s'
			}]
		},
		options:{
			plugins:{
				title:{display:true,text:'%s — Tier Distribution',color:'%s',font:{size:18,weight:'bold'}},
				legend:{position:'bottom',labels:{color:'%s',font:{size:13},padding:15,usePointStyle:true}},
				datalabels:{color:'%s',font:{size:14,weight:'bold'},formatter:function(v){return v>0?v:''}}
			}
		}
	}`,
		plat, gold, silver, biz,
		plat, gold, silver, biz,
		tierColors["platinum"], tierColors["gold"], tierColors["silver"], tierColors["business"],
		bgColor,
		centerName, textColor,
		textColor,
		textColor,
	)

	return buildURL(config)
}

// CountryStackedBarURL builds a stacked bar chart of countries by tier.
func CountryStackedBarURL(countries []string, plat, gold, silver, biz []int) string {
	config := fmt.Sprintf(`{
		type:'bar',
		data:{
			labels:%s,
			datasets:[
				{label:'Platinum',data:%s,backgroundColor:'%s'},
				{label:'Gold',data:%s,backgroundColor:'%s'},
				{label:'Silver',data:%s,backgroundColor:'%s'},
				{label:'Business',data:%s,backgroundColor:'%s'}
			]
		},
		options:{
			indexAxis:'y',
			plugins:{
				title:{display:true,text:'Partners by Country',color:'%s',font:{size:18,weight:'bold'}},
				legend:{position:'bottom',labels:{color:'%s',font:{size:12},usePointStyle:true}},
				datalabels:{display:false}
			},
			scales:{
				x:{stacked:true,grid:{color:'%s'},ticks:{color:'%s'}},
				y:{stacked:true,grid:{display:false},ticks:{color:'%s',font:{size:13}}}
			}
		}
	}`,
		toJSArray(countries),
		toIntArray(plat), tierColors["platinum"],
		toIntArray(gold), tierColors["gold"],
		toIntArray(silver), tierColors["silver"],
		toIntArray(biz), tierColors["business"],
		textColor,
		textColor,
		gridColor, textColor,
		textColor,
	)

	return buildURL(config)
}

// VolumeTopBarURL builds a horizontal bar chart for top partners by volume.
func VolumeTopBarURL(names []string, volumes []float64) string {
	config := fmt.Sprintf(`{
		type:'bar',
		data:{
			labels:%s,
			datasets:[{
				label:'Volume ($)',
				data:%s,
				backgroundColor:'rgba(59,130,246,0.8)',
				borderColor:'rgb(59,130,246)',
				borderWidth:1,
				borderRadius:4
			}]
		},
		options:{
			indexAxis:'y',
			plugins:{
				title:{display:true,text:'Top Partners by Volume',color:'%s',font:{size:18,weight:'bold'}},
				legend:{display:false},
				datalabels:{color:'%s',font:{size:12,weight:'bold'},anchor:'end',align:'end',formatter:function(v){if(v>=1e6)return '$'+Math.round(v/1e6*10)/10+'M';if(v>=1e3)return '$'+Math.round(v/1e3)+'K';return '$'+v}}
			},
			scales:{
				x:{grid:{color:'%s'},ticks:{color:'%s',callback:function(v){if(v>=1e6)return '$'+v/1e6+'M';if(v>=1e3)return '$'+v/1e3+'K';return '$'+v}}},
				y:{grid:{display:false},ticks:{color:'%s',font:{size:12}}}
			}
		}
	}`,
		toJSArray(names),
		toFloatArray(volumes),
		textColor,
		textColor,
		gridColor, textColor,
		textColor,
	)

	return buildURL(config)
}

func buildURL(config string) string {
	params := url.Values{}
	params.Set("c", config)
	params.Set("w", fmt.Sprintf("%d", chartWidth))
	params.Set("h", fmt.Sprintf("%d", chartHeight))
	params.Set("bkg", bgColor)
	params.Set("f", "png")
	return quickChartBaseURL + "?" + params.Encode()
}

func toJSArray(items []string) string {
	quoted := make([]string, len(items))
	for i, s := range items {
		quoted[i] = fmt.Sprintf("'%s'", s)
	}
	return "[" + strings.Join(quoted, ",") + "]"
}

func toIntArray(items []int) string {
	strs := make([]string, len(items))
	for i, v := range items {
		strs[i] = fmt.Sprintf("%d", v)
	}
	return "[" + strings.Join(strs, ",") + "]"
}

func toFloatArray(items []float64) string {
	strs := make([]string, len(items))
	for i, v := range items {
		strs[i] = fmt.Sprintf("%.0f", v)
	}
	return "[" + strings.Join(strs, ",") + "]"
}
