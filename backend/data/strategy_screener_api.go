package data

import (
	"strings"
	"sync"

	"github.com/duke-git/lancet/v2/convertor"
)

const (
	waterStrategyQuery = "涨幅大于3%;涨幅小于5%;量比大于1.4;流通市值小于200亿;换手率大于5%;换手率小于10%;20天内有过涨停;非ST"
	minuteCheckWorkers = 8
	minValidMinuteBars = 10
)

// StrategyScreenerApi 固定策略选股。
type StrategyScreenerApi struct{}

func NewStrategyScreenerApi() *StrategyScreenerApi {
	return &StrategyScreenerApi{}
}

// RunWaterStrategyScreen 执行「水上策略」选股：
// 涨幅 3%-5%、20 天内有过涨停、量比>1.4、市值<200 亿、换手率 5%-10%、分时全天在均价线上方。
func (s *StrategyScreenerApi) RunWaterStrategyScreen() map[string]any {
	raw := NewSearchStockApi(waterStrategyQuery).SearchStock(500)
	if raw == nil {
		return map[string]any{
			"code":    -1,
			"message": "选股请求失败",
		}
	}
	if code, ok := raw["code"]; ok {
		codeVal, _ := convertor.ToInt(code)
		switch codeVal {
		case -1:
			return raw
		case 100:
			// continue
		default:
			return raw
		}
	}

	data, ok := raw["data"].(map[string]any)
	if !ok {
		return map[string]any{
			"code":    -1,
			"message": "选股响应格式异常",
		}
	}
	result, ok := data["result"].(map[string]any)
	if !ok {
		return map[string]any{
			"code":    -1,
			"message": "选股结果为空",
		}
	}
	columns, _ := result["columns"].([]any)
	dataListAny, _ := result["dataList"].([]any)
	preCount := len(dataListAny)
	if preCount == 0 {
		return raw
	}

	filtered := make([]any, 0, preCount)
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, minuteCheckWorkers)

	for _, item := range dataListAny {
		row, ok := item.(map[string]any)
		if !ok {
			continue
		}
		wg.Add(1)
		go func(stockRow map[string]any) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			tdxCode := eastMoneyRowToTdxCode(stockRow)
			if tdxCode == "" || !isAboveAvgAllDay(tdxCode) {
				return
			}
			mu.Lock()
			filtered = append(filtered, stockRow)
			mu.Unlock()
		}(row)
	}
	wg.Wait()

	result["dataList"] = filtered
	data["result"] = result
	data["strategyMeta"] = map[string]any{
		"name":           "水上策略",
		"query":          waterStrategyQuery,
		"preFilterCount": preCount,
		"postFilterCount": len(filtered),
		"criteria": []map[string]string{
			{"label": "涨幅", "value": "3% – 5%"},
			{"label": "涨停基因", "value": "20 天内有过涨停"},
			{"label": "量比", "value": "> 1.4"},
			{"label": "市值", "value": "< 200 亿"},
			{"label": "换手率", "value": "5% – 10%"},
			{"label": "分时形态", "value": "全天在均价线上方（水上）"},
		},
	}
	raw["data"] = data
	if columns != nil {
		_ = columns
	}
	return raw
}

func eastMoneyRowToTdxCode(row map[string]any) string {
	market := strings.ToUpper(strings.TrimSpace(convertor.ToString(row["MARKET_SHORT_NAME"])))
	code := strings.TrimSpace(convertor.ToString(row["SECURITY_CODE"]))
	if code == "" {
		return ""
	}
	if market == "" {
		if strings.HasPrefix(code, "6") {
			market = "SH"
		} else if strings.HasPrefix(code, "0") || strings.HasPrefix(code, "3") {
			market = "SZ"
		} else if strings.HasPrefix(code, "8") || strings.HasPrefix(code, "4") {
			market = "BJ"
		}
	}
	if market == "" {
		return ""
	}
	return code + "." + market
}

// isAboveAvgAllDay 判断分时是否全天运行在均价线上方（水上）。
func isAboveAvgAllDay(stockCode string) bool {
	bundle := NewTdxKLineApi().GetMinuteTimeDataAuto(stockCode)
	if bundle == nil || len(bundle.Items) == 0 {
		return false
	}

	validBars := 0
	for _, item := range bundle.Items {
		if item.Vol <= 0 || item.Price <= 0 || item.Avg <= 0 {
			continue
		}
		validBars++
		if item.Price+0.001 < item.Avg {
			return false
		}
	}
	return validBars >= minValidMinuteBars
}
