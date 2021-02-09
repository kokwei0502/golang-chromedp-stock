package routers

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kokwei0502/golang-chromedp-stock/controllers"
)

// YahooFinanceListing = Company historical prices, info and ratios data structure
type YahooFinanceListing struct {
	HistoricalPrice *controllers.YahooFinanceHistoricalData
	GeneralInfo     *controllers.MalaysiaStockBizCompanyInfo
	GeneralRatio    *[]*controllers.GeneralInfoKLSE
}

var (
	stockListing  []*controllers.MalaysiaStockBizCompanyInfo
	finalListing  []*YahooFinanceListing
	boolInit      = true
	lengthListing int
	startLength   int = 0
	pageList          = []string{}
)

func init() {
	stockListing = controllers.RetrieveMalaysiaStockBizListing() // Retrieve stock listing from json file
}

// IndexPageYahooFinance = Index page for stock historical price from finance.yahoo, general ratio from klsescreener.com
func IndexPageYahooFinance(w http.ResponseWriter, r *http.Request) {
	stockCode := r.URL.Query().Get("code")                    // Get the stock code from url address
	generalRatio := controllers.GetGeneralInfoKLSE(stockCode) // Get general ratios from klsescreener.com
	var generalInfo *controllers.MalaysiaStockBizCompanyInfo
	for k := range stockListing {
		if stockListing[k].StockCode == stockCode { // Check the stock code match the stock listing data
			generalInfo = stockListing[k]
		}
	}
	histprice := controllers.GetHistYahooFinance(stockCode) // Get historical prices from finance.yahoo
	companyInfo := &YahooFinanceListing{                    // Company data structure (historical price, info and ratios)
		HistoricalPrice: histprice,
		GeneralInfo:     generalInfo,
		GeneralRatio:    &generalRatio,
	}
	pageData := struct {
		CompanyInfo *YahooFinanceListing // Company info
	}{
		CompanyInfo: companyInfo,
	}
	controllers.AllHTMLTemplates.ExecuteTemplate(w, "stockinfo-yahoo-finance.html", pageData)
}

const (
	seperatePage = 5
)

// IndexPageYahooFinanceListing = Index page for filter listing by sector or sub sector
func IndexPageYahooFinanceListing(w http.ResponseWriter, r *http.Request) {
	timestart := time.Now()
	if boolInit { // Check the page initial data, if true means the page is 1st time visit
		finalListing = getSectorListing(r.URL.RawQuery, stockListing) // Get the data listing by sector or sub sector
		boolInit = false                                              // Change the status to false, means will not always retrieve data from function
		if len(finalListing) >= seperatePage {                        // Check if the length of data listing is greater than "seperate page value", then need to seperte data by pages
			lengthListing = seperatePage                                    // Initially data[num], num should be seperate num
			x := int(math.Round(float64(len(finalListing)) / seperatePage)) // Calculate total pages needed
			for y := 1; y <= x; y++ {
				pageList = append(pageList, strconv.Itoa(y)) // Append page number to list
			}
		} else {
			lengthListing = len(finalListing) // If length of listing is lesser than seperate num, then data[num] num will get the length of listing
		}
	}

	if r.Method == "POST" {
		r.ParseForm()
		pageNum := r.PostFormValue("page-data") // Get the page number
		if pageNum != "" {
			num, _ := strconv.Atoi(pageNum) // Convert page number(string) to integer
			if num == 1 {                   // If number is 1, means still remains at initial listing for 1st page
				startLength = 0
			} else {
				startLength = (num - 1) * seperatePage // Get the data[num1:lengthListing] num1
			}
			if len(finalListing) < lengthListing { // If lengthListing greater than data listing, means already last page reached, just take the length of listing for last page
				lengthListing = len(finalListing)
			} else {
				lengthListing = num * seperatePage
			}
		}
	}
	pageData := struct { // Page data structure
		StockListing []*YahooFinanceListing // Listing for all companies by sector or sub sector
		Page         []string               // Page list
	}{StockListing: finalListing[startLength:lengthListing], Page: pageList}
	timeend := time.Since(timestart)
	fmt.Println(timeend.Seconds())
	fmt.Println("end")
	controllers.AllHTMLTemplates.ExecuteTemplate(w, "stocklisting-yahoo-finance.html", pageData)
}

func getSectorListing(urlraw string, stocklist []*controllers.MalaysiaStockBizCompanyInfo) []*YahooFinanceListing {
	var category string
	var sectorName string
	if strings.HasPrefix(urlraw, "sector") {
		category = "sector"
		sectorName = strings.Replace(strings.ReplaceAll(urlraw, "%20", " "), "sector=", "", 1)
	} else if strings.HasPrefix(urlraw, "subsector") {
		category = "subsector"
		sectorName = strings.Replace(strings.ReplaceAll(urlraw, "%20", " "), "subsector=", "", 1)
	}
	var wg = sync.WaitGroup{}
	ch := make(chan int, len(stocklist))
	wg.Add(len(stocklist))
	for i := 0; i < len(stocklist); i++ {
		go func(i int) {
			ch <- i
			var sectorFromList string
			switch category {
			case "sector":
				sectorFromList = stocklist[i].Sector
			case "subsector":
				sectorFromList = stocklist[i].SubSector
			}
			if sectorFromList == sectorName {
				histprice := controllers.GetHistYahooFinance(stocklist[i].StockCode)
				ratio := controllers.GetGeneralInfoKLSE(stocklist[i].StockCode)
				if len(histprice.Price) > 1 {
					finalListing = append(finalListing, &YahooFinanceListing{
						HistoricalPrice: histprice,
						GeneralInfo:     stocklist[i],
						GeneralRatio:    &ratio,
					})
				}
			}
			wg.Done()
			<-ch
		}(i)
	}
	wg.Wait()

	return finalListing
}
