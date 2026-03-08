package chart

import (
	"fmt"
	"net/url"
	"strings"
)

const (
	quickChartBaseURL = "https://quickchart.io/chart"
	chartWidth        = 700
	chartHeight       = 450
	bgColor           = "rgb(15,23,42)"     // slate-900 (deeper, cleaner)
	textColor         = "rgb(226,232,240)"   // slate-300 (softer white)
	gridColor         = "rgba(100,116,139,0.15)" // subtle grid
)

// HPE-inspired tier palette — each tier clearly distinct
var tierColors = map[string]string{
	"platinum": "rgb(139,92,246)",  // violet-500
	"gold":     "rgb(251,191,36)", // amber-400
	"silver":   "rgb(100,116,139)", // slate-500
	"business": "rgb(51,65,85)",   // slate-700
}

var tierBorders = map[string]string{
	"platinum": "rgb(167,139,250)", // violet-400
	"gold":     "rgb(252,211,77)",  // amber-300
	"silver":   "rgb(148,163,184)", // slate-400
	"business": "rgb(71,85,105)",   // slate-600
}

// TierDoughnutURL builds a doughnut chart for tier distribution.
func TierDoughnutURL(dist map[string]int, centerName string) string {
	plat := dist["platinum"]
	gold := dist["gold"]
	silver := dist["silver"]
	biz := dist["business"]

	config := fmt.Sprintf(`{
		type:'doughnut',
		data:{
			labels:['Platinum','Gold','Silver','Business'],
			datasets:[{
				data:[%d,%d,%d,%d],
				backgroundColor:['%s','%s','%s','%s'],
				borderColor:['%s','%s','%s','%s'],
				borderWidth:2,
				spacing:3,
				hoverOffset:8
			}]
		},
		options:{
			cutout:'55%%',
			layout:{padding:{top:10,bottom:10}},
			plugins:{
				title:{display:true,text:'%s — Tier Distribution',color:'%s',
					font:{size:20,family:'Inter,system-ui,sans-serif',weight:'600'},
					padding:{bottom:20}},
				legend:{position:'bottom',
					labels:{color:'%s',font:{size:13,family:'Inter,system-ui,sans-serif'},
						padding:20,usePointStyle:true,pointStyle:'circle',boxWidth:10}},
				datalabels:{
					color:'white',font:{size:15,weight:'bold',family:'Inter,system-ui,sans-serif'},
					textShadowBlur:4,textShadowColor:'rgba(0,0,0,0.5)',
					formatter:function(v,ctx){if(v===0)return '';var t=ctx.dataset.data.reduce(function(a,b){return a+b},0);return v+' ('+Math.round(v/t*100)+'%%)'}}
			}
		}
	}`,
		plat, gold, silver, biz,
		tierColors["platinum"], tierColors["gold"], tierColors["silver"], tierColors["business"],
		tierBorders["platinum"], tierBorders["gold"], tierBorders["silver"], tierBorders["business"],
		centerName, textColor,
		textColor,
	)

	return buildURL(config)
}

// CountryStackedBarURL builds a horizontal stacked bar of countries by tier.
func CountryStackedBarURL(countries []string, plat, gold, silver, biz []int) string {
	config := fmt.Sprintf(`{
		type:'bar',
		data:{
			labels:%s,
			datasets:[
				{label:'Platinum',data:%s,backgroundColor:'%s',borderColor:'%s',borderWidth:1,borderRadius:2},
				{label:'Gold',data:%s,backgroundColor:'%s',borderColor:'%s',borderWidth:1,borderRadius:2},
				{label:'Silver',data:%s,backgroundColor:'%s',borderColor:'%s',borderWidth:1,borderRadius:2},
				{label:'Business',data:%s,backgroundColor:'%s',borderColor:'%s',borderWidth:1,borderRadius:2}
			]
		},
		options:{
			indexAxis:'y',
			layout:{padding:{left:5,right:20}},
			plugins:{
				title:{display:true,text:'Partners by Country & Tier',color:'%s',
					font:{size:20,family:'Inter,system-ui,sans-serif',weight:'600'},
					padding:{bottom:15}},
				legend:{position:'bottom',
					labels:{color:'%s',font:{size:12,family:'Inter,system-ui,sans-serif'},
						padding:18,usePointStyle:true,pointStyle:'circle',boxWidth:10}},
				datalabels:{display:false}
			},
			scales:{
				x:{stacked:true,grid:{color:'%s',drawBorder:false},
					ticks:{color:'%s',font:{size:11,family:'Inter,system-ui,sans-serif'}}},
				y:{stacked:true,grid:{display:false},
					ticks:{color:'%s',font:{size:13,family:'Inter,system-ui,sans-serif',weight:'500'}}}
			}
		}
	}`,
		toJSArray(countries),
		toIntArray(plat), tierColors["platinum"], tierBorders["platinum"],
		toIntArray(gold), tierColors["gold"], tierBorders["gold"],
		toIntArray(silver), tierColors["silver"], tierBorders["silver"],
		toIntArray(biz), tierColors["business"], tierBorders["business"],
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
				data:%s,
				backgroundColor:'rgba(99,102,241,0.75)',
				borderColor:'rgb(129,140,248)',
				borderWidth:1,
				borderRadius:4,
				barPercentage:0.7
			}]
		},
		options:{
			indexAxis:'y',
			layout:{padding:{left:5,right:30}},
			plugins:{
				title:{display:true,text:'Top Partners by Volume',color:'%s',
					font:{size:20,family:'Inter,system-ui,sans-serif',weight:'600'},
					padding:{bottom:15}},
				legend:{display:false},
				datalabels:{
					color:'%s',font:{size:12,weight:'600',family:'Inter,system-ui,sans-serif'},
					anchor:'end',align:'end',
					formatter:function(v){if(v>=1e6)return '$'+(Math.round(v/1e5)/10)+'M';if(v>=1e3)return '$'+Math.round(v/1e3)+'K';return '$'+v}}
			},
			scales:{
				x:{grid:{color:'%s',drawBorder:false},
					ticks:{color:'%s',font:{size:11,family:'Inter,system-ui,sans-serif'},
						callback:function(v){if(v>=1e6)return '$'+v/1e6+'M';if(v>=1e3)return '$'+v/1e3+'K';return '$'+v}}},
				y:{grid:{display:false},
					ticks:{color:'%s',font:{size:13,family:'Inter,system-ui,sans-serif',weight:'500'}}}
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
