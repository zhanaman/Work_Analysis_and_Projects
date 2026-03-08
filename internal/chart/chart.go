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
	bgColor           = "rgb(15,23,42)"            // slate-900 (deeper, cleaner)
	textColor         = "rgb(226,232,240)"          // slate-300 (softer white)
	gridColor         = "rgba(100,116,139,0.15)"    // subtle grid
)

// PBM Risk & Action Colors
var riskColors = map[string]string{
	"safe":      "rgb(34,197,94)",   // green-500
	"certBlock": "rgb(234,179,8)",   // yellow-500 (needs workshop)
	"volBlock":  "rgb(249,115,22)",  // orange-500 (needs sales push)
	"deepGap":   "rgb(239,68,68)",   // red-500 (high risk)
}

var concentrationColors = []string{
	"rgb(168,85,247)", // Top 3 - purple-500
	"rgb(56,189,248)", // Next 7 - sky-400
	"rgb(100,116,139)", // Rest - slate-500
}

// UpgradePipelineChart builds a stacked bar chart showing upgrade blockers.
func UpgradePipelineChart(centers []string, ready, certBlocked, volBlocked, deepGap []int) string {
	config := fmt.Sprintf(`{
		type:'bar',
		data:{
			labels:%s,
			datasets:[
				{label:'Ready to Upgrade',data:%s,backgroundColor:'%s',borderRadius:2},
				{label:'Cert Blocked',data:%s,backgroundColor:'%s',borderRadius:2},
				{label:'Volume Blocked',data:%s,backgroundColor:'%s',borderRadius:2},
				{label:'Deep Gap',data:%s,backgroundColor:'%s',borderRadius:2}
			]
		},
		options:{
			indexAxis:'y',
			layout:{padding:{left:5,right:20}},
			plugins:{
				title:{display:true,text:'Upgrade Pipeline (Blockers)',color:'%s',font:{size:20,family:'Inter,system-ui,sans-serif',weight:'600'},padding:{bottom:15}},
				legend:{position:'bottom',labels:{color:'%s',font:{size:12,family:'Inter,system-ui,sans-serif'},padding:18,usePointStyle:true}},
				datalabels:{display:false}
			},
			scales:{
				x:{stacked:true,grid:{color:'%s',drawBorder:false},ticks:{color:'%s',font:{size:12}}},
				y:{stacked:true,grid:{display:false},ticks:{color:'%s',font:{size:13,weight:'500'}}}
			}
		}
	}`,
		toJSArray(centers),
		toIntArray(ready), riskColors["safe"],
		toIntArray(certBlocked), riskColors["certBlock"],
		toIntArray(volBlocked), riskColors["volBlock"],
		toIntArray(deepGap), riskColors["deepGap"],
		textColor, textColor, gridColor, textColor, textColor,
	)
	return buildURL(config)
}

// LowHangingFruitChart builds a stacked horizontal bar showing Volume vs Gap to threshold.
func LowHangingFruitChart(names []string, volumes []float64, gaps []float64) string {
	config := fmt.Sprintf(`{
		type:'bar',
		data:{
			labels:%s,
			datasets:[
				{label:'Current Volume',data:%s,backgroundColor:'rgb(59,130,246)',borderRadius:1},
				{label:'Gap to Next Tier',data:%s,backgroundColor:'rgba(148,163,184,0.3)',borderRadius:1}
			]
		},
		options:{
			indexAxis:'y',
			layout:{padding:{left:5,right:30}},
			plugins:{
				title:{display:true,text:'Low-Hanging Fruit (80%%-99%% to next tier)',color:'%s',font:{size:20,family:'Inter,system-ui,sans-serif',weight:'600'},padding:{bottom:15}},
				legend:{position:'bottom',labels:{color:'%s',font:{size:12,family:'Inter,system-ui,sans-serif'},usePointStyle:true}},
				datalabels:{color:'%s',font:{size:11,weight:'600'},align:'center',formatter:function(v,ctx){if(val===0)return'';var val=v;if(val>=1e6)return '$'+(Math.round(val/1e5)/10)+'M';if(val>=1e3)return '$'+Math.round(val/1e3)+'K';return '$'+val}}
			},
			scales:{
				x:{stacked:true,grid:{color:'%s',drawBorder:false},ticks:{color:'%s',callback:function(v){if(v>=1e6)return '$'+v/1e6+'M';if(v>=1e3)return '$'+v/1e3+'K';return '$'+v}}},
				y:{stacked:true,grid:{display:false},ticks:{color:'%s',font:{size:12,weight:'500'}}}
			}
		}
	}`,
		toJSArray(names),
		toFloatArray(volumes),
		toFloatArray(gaps),
		textColor, textColor, textColor, gridColor, textColor, textColor,
	)
	return buildURL(config)
}

// RetentionRiskChart builds a doughnut chart showing current Plat/Gold retention risk.
func RetentionRiskChart(safe, volRisk, certRisk, deepRisk int) string {
	config := fmt.Sprintf(`{
		type:'doughnut',
		data:{
			labels:['Safe','Volume Risk','Cert Risk','Deep Risk'],
			datasets:[{
				data:[%d,%d,%d,%d],
				backgroundColor:['%s','%s','%s','%s'],
				borderColor:'%s',
				borderWidth:2,
				spacing:3,
				hoverOffset:8
			}]
		},
		options:{
			cutout:'55%%',
			plugins:{
				title:{display:true,text:'FY27 Retention Risk (Current Plat & Gold)',color:'%s',font:{size:20,family:'Inter,system-ui,sans-serif',weight:'600'},padding:{bottom:20}},
				legend:{position:'bottom',labels:{color:'%s',font:{size:13},padding:20,usePointStyle:true}},
				datalabels:{color:'white',font:{size:15,weight:'bold'},textShadowBlur:4,textShadowColor:'rgba(0,0,0,0.5)',formatter:function(v,ctx){if(v===0)return '';return v}}
			}
		}
	}`,
		safe, volRisk, certRisk, deepRisk,
		riskColors["safe"], riskColors["volBlock"], riskColors["certBlock"], riskColors["deepGap"],
		bgColor, textColor, textColor,
	)
	return buildURL(config)
}

// ConcentrationChart builds a doughnut chart for Volume concentration (Top 3 vs Next 7 vs Rest).
func ConcentrationChart(top3, next7, rest float64, center string) string {
	config := fmt.Sprintf(`{
		type:'doughnut',
		data:{
			labels:['Top 3 Partners','Next 7 Partners','Rest of Channel'],
			datasets:[{
				data:[%f,%f,%f],
				backgroundColor:['%s','%s','%s'],
				borderColor:'%s',
				borderWidth:2,
				spacing:3
			}]
		},
		options:{
			cutout:'60%%',
			plugins:{
				title:{display:true,text:'%s — Revenue Concentration',color:'%s',font:{size:20,family:'Inter,system-ui,sans-serif',weight:'600'},padding:{bottom:20}},
				legend:{position:'bottom',labels:{color:'%s',font:{size:13},padding:20,usePointStyle:true}},
				datalabels:{color:'white',font:{size:14,weight:'bold'},textShadowBlur:4,textShadowColor:'rgba(0,0,0,0.5)',formatter:function(v,ctx){if(v===0)return '';var t=ctx.dataset.data.reduce(function(a,b){return a+b},0);return Math.round(v/t*100)+'%%'}}
			}
		}
	}`,
		top3, next7, rest,
		concentrationColors[0], concentrationColors[1], concentrationColors[2],
		bgColor, center, textColor, textColor,
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
