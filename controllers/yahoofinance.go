package controllers

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	// Yahoo finance request base URL
	yfURL string = "https://sg.finance.yahoo.com/quote/{stockcode}.KL/history?period1={period1}&period2={period2}&interval=1d&filter=history&frequency=1d&includeAdjustedClose=true"
)

// YahooFinanceHistoricalData = Data structure for historical data from finance.yahoo
type YahooFinanceHistoricalData struct {
	Price []struct {
		Date     interface{} `json:"date"`
		Close    float64     `json:"close"`
		Open     float64     `json:"open"`
		High     float64     `json:"high"`
		Low      float64     `json:"low"`
		Volume   float64     `json:"volume"`
		AdjClose float64     `json:"adjclose"`
	} `json:"prices"`
}

// GetHistYahooFinance = Get historical price data from finance.yahoo
func GetHistYahooFinance(stockcode string) *YahooFinanceHistoricalData {
	period1 := strconv.Itoa(int(time.Now().AddDate(-10, 0, 0).Unix())) // Date today minus 10 years
	period2 := strconv.Itoa(int(time.Now().Unix()))                    // Date today

	// Http request to yahoo.finance
	url := strings.Replace(strings.Replace(strings.Replace(yfURL, "{period1}", period1, 1), "{period2}", period2, 1), "{stockcode}", stockcode, 1)
	res, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}

	x := regexp.MustCompile(`\"HistoricalPriceStore":(.*?)\,"isPending`) // Find strings from response body
	findStr := x.FindStringSubmatch(string(body))
	var histData = &YahooFinanceHistoricalData{}
	if len(findStr) > 0 { // If able to find keywords from regexp
		json.Unmarshal([]byte(findStr[1]+"}"), &histData)
		sort.Slice(histData.Price, func(i, j int) bool { // Sort the data by "date" from oldest to latest
			return histData.Price[i].Date.(float64) < histData.Price[j].Date.(float64)
		})

		for k := range histData.Price {
			// Convert date format to "yyyy-mm-dd" to match ploting criteria
			histData.Price[k].Date = time.Unix(int64(histData.Price[k].Date.(float64)), 0).Format("2006-01-02")
			histData.Price[k].Date = histData.Price[k].Date.(string)
		}
	}
	return histData
}
