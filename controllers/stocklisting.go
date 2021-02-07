package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

var (
	investingBaseURL string = "https://www.investing.com/stock-screener/?sp=country::42|sector::a|industry::a|equityType::a|exchange::62%3Ceq_market_cap;"
	investingPage    int    = 1
	wg               sync.WaitGroup
	mutex            = sync.Mutex{}
	workingDir       string
	listResult       []*StockInfo
	mutexRW          = sync.RWMutex{}
	timeStart        time.Time
	timeEnd          time.Duration
	options          = []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", false),
		chromedp.Flag("hide-scrollbars", false),
		chromedp.Flag("mute-audio", true),
		chromedp.Flag("start-maximized", true),
		chromedp.UserAgent(`Mozilla/5.0 (Windows NT 6.3; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/73.0.3683.103 Safari/537.36`),
	}
)

// StockListing = stock company listing data structure
type StockListing struct {
	Date      string
	StockList []*StockInfo
}

// StockInfo = stock company data structure
type StockInfo struct {
	Company   string
	Market    string
	Symbol    string
	StockCode string
	Sector    string
	Industry  string
	MarketCap interface{}
	StockURL  string
}

const (
	stocklistingFile = "stocklisting.json"
)

func init() {
	workingDir, _ = os.Getwd()
}

// RetrieveStockListing = retrive stock listing from json file
func RetrieveStockListing() *StockListing {
	file, err := ioutil.ReadFile(workingDir + "/static/file/stocklisting/" + stocklistingFile)
	if err != nil {
		log.Fatal(err)
	}
	var data = &StockListing{}
	err = json.Unmarshal(file, &data)
	if err != nil {
		log.Fatal(err)
	}
	return data
}

// UpdateStockListing = chromedp to retrieve stock listing from investing.com
func UpdateStockListing() {
	timeStart := time.Now()
	// Setup chromedp
	options = append(chromedp.DefaultExecAllocatorOptions[:], options...) // append the chromedp options
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), options...)
	defer cancel()
	mainCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	mainCtx, cancel = context.WithTimeout(mainCtx, 2*time.Minute) // timeout setting
	defer cancel()

	// Navigate to the main page and check the total number of stocks found
	baseURL := investingBaseURL + strconv.Itoa(investingPage)
	var totalFoundstr string
	var nodeMain []*cdp.Node
	err := chromedp.Run(mainCtx,
		chromedp.Navigate(baseURL),
		chromedp.Nodes(`table[id="resultsTable"] > tbody > tr`, &nodeMain, chromedp.ByQueryAll, chromedp.AtLeast(1)),
		chromedp.Text(`span.js-total-results`, &totalFoundstr, chromedp.ByQuery, chromedp.AtLeast(0)),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Calculate total page return based on 50 records each page
	totalFound, err := strconv.ParseFloat(totalFoundstr, 64)
	if err != nil {
		log.Fatal(err)
	}
	totalPageCal := totalFound / 50
	var totalPage int
	if totalPageCal != float64(int(totalPageCal)) {
		totalPage = int(totalPageCal) + 1
	}
	fmt.Println(totalPage)
	// create results structure and use later for new chrome tab for each page
	type resultStructure struct {
		ctx             context.Context
		canceltab       context.CancelFunc
		url             string
		nodeMainResults []*cdp.Node
		nodeSignupPopup []*cdp.Node
		result          *StockInfo
	}
	var resultCtx []*resultStructure
	for i := 1; i <= totalPage; i++ {
		newCtx, newCancel := chromedp.NewContext(mainCtx)
		resultCtx = append(resultCtx, &resultStructure{
			ctx:             newCtx,
			canceltab:       newCancel,
			url:             investingBaseURL + strconv.Itoa(i),
			nodeMainResults: nil,
			nodeSignupPopup: nil,
			result:          &StockInfo{},
		})
	}

	// gorountine to open tabs for each page info and retrieve data
	chanInvesting := make(chan string, 20)
	wg.Add(len(resultCtx))

	// var nodeMainResults, nodeSignupPopup []*cdp.Node
	for g := 0; g < len(resultCtx); g++ {
		go func(g int) {
			chanInvesting <- "start"
			err := chromedp.Run(resultCtx[g].ctx,
				chromedp.Navigate(resultCtx[g].url),
				chromedp.WaitVisible(`section[class="bottom"]`, chromedp.ByQuery),
				chromedp.Nodes(`div[class="signupWrap js-gen-popup dark_graph"]`, &resultCtx[g].nodeSignupPopup, chromedp.ByQuery, chromedp.AtLeast(0)),
			)
			if err != nil {
				log.Fatal(err)
			}
			if len(resultCtx[g].nodeSignupPopup) == 1 {
				err := chromedp.Run(resultCtx[g].ctx,
					chromedp.Click(`i[class="popupCloseIcon largeBannerCloser"]`, chromedp.ByQuery),
					chromedp.Sleep(500*time.Millisecond),
				)
				if err != nil {
					log.Fatal(err)
				}
			}
			err = chromedp.Run(resultCtx[g].ctx,
				chromedp.WaitVisible(`table[id="resultsTable"] > tbody > tr`),
				chromedp.Click("div.colSelectIconWrapper", chromedp.ByQuery),
				chromedp.Click(`input[id="SS_4"]`, chromedp.ByQuery),
				chromedp.Click(`input[id="SS_5"]`, chromedp.ByQuery),
				chromedp.Click(`a[id="selectColumnsButton_stock_screener"`, chromedp.ByQuery),
				chromedp.Nodes(`table[id="resultsTable"] > tbody > tr`, &resultCtx[g].nodeMainResults, chromedp.ByQueryAll, chromedp.AtLeast(1)),
			)
			if err != nil {
				log.Fatal(err)
			}
			for i := 0; i < len(resultCtx[g].nodeMainResults); i++ {
				var marketcapStr string
				var link map[string]string
				if err = chromedp.Run(resultCtx[g].ctx,
					chromedp.Text(`td[data-column-name="name_trans"]`, &resultCtx[g].result.Company, chromedp.ByQuery, chromedp.AtLeast(0), chromedp.FromNode(resultCtx[g].nodeMainResults[i])),
					chromedp.Text(`td[data-column-name="viewData.symbol"]`, &resultCtx[g].result.Symbol, chromedp.ByQuery, chromedp.AtLeast(0), chromedp.FromNode(resultCtx[g].nodeMainResults[i])),
					chromedp.Text(`td[data-column-name="sector_trans"]`, &resultCtx[g].result.Sector, chromedp.ByQuery, chromedp.AtLeast(0), chromedp.FromNode(resultCtx[g].nodeMainResults[i])),
					chromedp.Text(`td[data-column-name="industry_trans"]`, &resultCtx[g].result.Industry, chromedp.ByQuery, chromedp.AtLeast(0), chromedp.FromNode(resultCtx[g].nodeMainResults[i])),
					chromedp.Text(`td[data-column-name="eq_market_cap"]`, &marketcapStr, chromedp.ByQuery, chromedp.AtLeast(0), chromedp.FromNode(resultCtx[g].nodeMainResults[i])),
					chromedp.Attributes(`td[data-column-name="name_trans"] > a`, &link, chromedp.ByQuery, chromedp.AtLeast(0), chromedp.FromNode(resultCtx[g].nodeMainResults[i])),
				); err != nil {
					log.Fatal(err)
				}

				resultCtx[g].result.StockURL = "https://www.investing.com" + link["href"]
				switch string(marketcapStr[len(marketcapStr)-1]) {
				case "M":
					marketcapStr = strings.ReplaceAll(marketcapStr, "M", "")
					resultCtx[g].result.MarketCap, _ = strconv.ParseFloat(marketcapStr, 64)
					resultCtx[g].result.MarketCap = resultCtx[g].result.MarketCap.(float64) * 1000000
				case "B":
					marketcapStr = strings.ReplaceAll(marketcapStr, "B", "")
					resultCtx[g].result.MarketCap, _ = strconv.ParseFloat(marketcapStr, 64)
					resultCtx[g].result.MarketCap = resultCtx[g].result.MarketCap.(float64) * 1000000000
				}
				fmt.Println(resultCtx[g].result.Company)
				mutexRW.Lock()
				listResult = append(listResult, &StockInfo{
					Company:   resultCtx[g].result.Company,
					Symbol:    resultCtx[g].result.Symbol,
					StockCode: "",
					Sector:    resultCtx[g].result.Sector,
					Industry:  resultCtx[g].result.Industry,
					MarketCap: resultCtx[g].result.MarketCap,
					StockURL:  resultCtx[g].result.StockURL,
				})
				mutexRW.Unlock()
			}
			<-chanInvesting
			wg.Done()
			resultCtx[g].canceltab()
		}(g)
	}
	wg.Wait()
	data := &StockListing{
		Date:      time.Now().Format("02-January-2006"),
		StockList: listResult,
	}
	jsonData, _ := json.MarshalIndent(data, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(workingDir+"/static/file/stocklisting/"+stocklistingFile, jsonData, 0644)
	if err != nil {
		log.Fatal(err)
	}
	UpdateStockDetailStockBiz()
	timeEnd = time.Since(timeStart)
	fmt.Println(timeEnd)
}

var (
	stockbizURL  = "https://www.malaysiastock.biz/Listed-Companies.aspx?type=A&value="
	listStockBiz []struct {
		StockCode, Company, Market string
	}
)

// UpdateStockDetailStockBiz = update stock code to investing.com data from malaysiastock.biz
func UpdateStockDetailStockBiz() {
	timeStart = time.Now()
	// Setup chromedp
	options = append(chromedp.DefaultExecAllocatorOptions[:], options...) // append chromedp options
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), options...)
	defer cancel()
	mainCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	mainCtx, cancel = context.WithTimeout(mainCtx, 2*time.Minute) // timeout setting
	defer cancel()
	err := chromedp.Run(mainCtx) // Open chrome browser
	if err != nil {
		log.Fatal(err)
	}

	// Setup malaysiastock.biz
	var listAlpha = []string{"0", "A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L", "M", "N", "O", "P", "Q", "R", "S", "T", "U", "V", "W", "X", "Y", "Z"}
	var contextList []struct {
		ctx         context.Context
		cancel      context.CancelFunc
		url         string
		nodeResults []*cdp.Node
		stockCode   string
		company     string
		market      string
	}
	for i := 0; i < len(listAlpha); i++ {
		newCtx, newCancel := chromedp.NewContext(mainCtx) // create new tab for each page for website
		contextList = append(contextList, struct {
			ctx         context.Context
			cancel      context.CancelFunc
			url         string
			nodeResults []*cdp.Node
			stockCode   string
			company     string
			market      string
		}{newCtx, newCancel, stockbizURL + listAlpha[i], nil, "", "", ""})
	}
	// listStockBiz is the data structure from malaysiastock.biz

	// goroutine to capture data from malaysiastock.biz
	chanStockBiz := make(chan string, 10)
	wg.Add(len(contextList))
	for i := 0; i < len(contextList); i++ {
		go func(i int) {
			chanStockBiz <- "start"
			err := chromedp.Run(contextList[i].ctx,
				chromedp.Navigate(contextList[i].url),
				chromedp.WaitVisible(`#MainContent_tStock`, chromedp.ByID),
				chromedp.Nodes(`table[id="MainContent_tStock"] > tbody > tr`, &contextList[i].nodeResults, chromedp.AtLeast(0), chromedp.ByQueryAll),
			)
			if err != nil {
				log.Fatal(err)
			}
			if len(contextList[i].nodeResults) > 1 {
				mutexRW.Lock()

				for x := 1; x < len(contextList[i].nodeResults); x++ {
					var nodeMarket []*cdp.Node
					err = chromedp.Run(contextList[i].ctx,
						chromedp.Text(`td:nth-child(1) > h3 > a`, &contextList[i].stockCode, chromedp.ByQuery, chromedp.FromNode(contextList[i].nodeResults[x])),
						chromedp.Nodes(`td:nth-child(1) > h3 > span`, &nodeMarket, chromedp.ByQuery, chromedp.AtLeast(0), chromedp.FromNode(contextList[i].nodeResults[x])),
						chromedp.Text(`td:nth-child(1) > h3:last-child`, &contextList[i].company, chromedp.ByQuery, chromedp.FromNode(contextList[i].nodeResults[x])),
					)
					if err != nil {
						log.Fatal(err)
					}
					fmt.Println(len(nodeMarket))
					if len(nodeMarket) > 0 {
						err = chromedp.Run(contextList[i].ctx,
							chromedp.Text(`td:nth-child(1) > h3 > span`, &contextList[i].market, chromedp.ByQuery, chromedp.FromNode(contextList[i].nodeResults[x])),
						)
						if err != nil {
							log.Fatal(err)
						}
					} else {
						contextList[i].market = "N/A"
					}
					listStockBiz = append(listStockBiz, struct{ StockCode, Company, Market string }{contextList[i].stockCode, contextList[i].company, contextList[i].market})
				}
				mutexRW.Unlock()
			}
			<-chanStockBiz
			wg.Done()
			contextList[i].cancel()
		}(i)
	}
	wg.Wait()

	// pass data (stockcode) to investing.com data and re-save to json format
	investingResults := RetrieveStockListing() // get investing.com data from existing json file

	var companyCheck string
	for i := range investingResults.StockList {
		// check the company name from investing.com data contains "bhd", need to escape "bhd" due to proper keyword is "berhad"
		if strings.HasSuffix(strings.ToLower(investingResults.StockList[i].Company), "bhd") {
			lastIn := strings.LastIndex(strings.ToLower(investingResults.StockList[i].Company), "bhd")
			companyCheck = strings.ToLower(investingResults.StockList[i].Company[:lastIn])
		} else {
			companyCheck = strings.ToLower(investingResults.StockList[i].Company)
		}
		if len(companyCheck) > 8 { // if length of company name is bigger than 8, only take first 8 keywords to search
			companyCheck = companyCheck[:8]
		}

		// match company name data from investing.com and malaysiastock.biz and pass the stock code to investing.com data
		for k := range listStockBiz {
			investingResults.StockList[i].Market = listStockBiz[k].Market
			x := strings.Contains(strings.ToLower(listStockBiz[k].Company), companyCheck)
			if x {
				stockCode1 := strings.Index(listStockBiz[k].StockCode, "(")
				stockCode2 := strings.Index(listStockBiz[k].StockCode, ")")
				if stockCode1+stockCode2 > 0 {
					investingResults.StockList[i].StockCode = listStockBiz[k].StockCode[stockCode1+1 : stockCode2]
				}
			}
		}
	}

	// Write data to file in json format
	jsonData, _ := json.MarshalIndent(investingResults, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(workingDir+"/static/file/stocklisting/"+stocklistingFile, jsonData, 0644)
	if err != nil {
		log.Fatal(err)
	}

	timeEnd := time.Since(timeStart)
	fmt.Println(timeEnd)
}

// StockPrice = Data structure for company historical stock prices
type StockPrice struct {
	Date                   string
	Close, Open, High, Low float64
	Volume                 float64
	Change                 string
}

// CompanyHistPrice = Data structure for company basic infos, historical prices, general ratios
type CompanyHistPrice struct {
	CompanyInfo  *StockInfo
	HistPrice    []*StockPrice
	GeneralRatio []*StockGeneralInfo
}

// GetStockInfoInvesting = get the stock historical price from investing.com
func GetStockInfoInvesting(stocklist []*StockInfo, year int) []*CompanyHistPrice {
	var stockPriceList []*CompanyHistPrice
	timeStart := time.Now()
	// Setup chromedp
	options = append(chromedp.DefaultExecAllocatorOptions[:], options...) // append the chromedp options
	allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), options...)
	defer cancel()
	mainCtx, cancel := chromedp.NewContext(allocCtx)
	defer cancel()
	// mainCtx, cancel = context.WithTimeout(mainCtx, 3*time.Minute) // timeout setting
	// defer cancel()

	// Run the browser first to further open new tabs
	err := chromedp.Run(mainCtx)
	if err != nil {
		log.Fatal(err)
	}

	// Create a context for further automated
	var contextList []struct {
		ctx       context.Context
		cancel    context.CancelFunc
		stockinfo *StockInfo
	}
	for k := range stocklist { // Loop through all the stocklist and create new context list with URL
		newCtx, newCancel := chromedp.NewContext(mainCtx)
		contextList = append(contextList, struct {
			ctx       context.Context
			cancel    context.CancelFunc
			stockinfo *StockInfo
		}{ctx: newCtx, cancel: newCancel, stockinfo: stocklist[k]},
		)
	}

	// Start to scrape the data from investing.com
	wg.Add(len(contextList))
	myCH := make(chan int, 5)
	for i := 0; i < len(contextList); i++ {
		go func(i int) {
			myCH <- i
			var stockCode string
			var priceList = []*StockPrice{}
			var historicalLink map[string]string
			err = chromedp.Run(contextList[i].ctx,
				chromedp.Navigate(contextList[i].stockinfo.StockURL),
				chromedp.WaitVisible("#quotes_summary_current_data", chromedp.ByID),
				chromedp.Text(`div[id="quotes_summary_current_data"] > div[class="right general-info"] > div:nth-child(4) > span:nth-child(2)`, &stockCode, chromedp.ByQuery),
				chromedp.Attributes(`ul[id="pairSublinksLevel2"] > li:nth-child(3) > a`, &historicalLink, chromedp.ByQuery, chromedp.AtLeast(0)),
			)
			if err != nil {
				log.Fatal(err)
			}

			// Get the stockcode from investing.com, previous code from stockbiz malaysia that data not completed
			// if the stock code lenght less than 4 then add "0" before stock code from investing.com
			if contextList[i].stockinfo.StockCode == "" { // check the stockbiz.malaysia stock code whether is empty
				if len(stockCode) < 4 {
					stockCode = strings.Repeat("0", 4-len(stockCode)) + stockCode
				}
				contextList[i].stockinfo.StockCode = stockCode
			}

			// Check the availability for historical link from investing.com
			boolHistLink := strings.Contains(historicalLink["href"], "historical")
			if boolHistLink { // if get the historical link
				// var nodeHistResults []*cdp.Node
				var res string

				dateToday := strings.Split(time.Now().Format("01/02/2006"), "/")                   // Date format for investing.com for retrieve data
				dateEnd := strings.Split(time.Now().AddDate(year, 0, 0).Format("01/02/2006"), "/") // Date format for investing.com for retrieve data

				err = chromedp.Run(contextList[i].ctx,
					chromedp.Navigate("https://www.investing.com"+historicalLink["href"]), // Navigate to investing.com historical url
					chromedp.WaitVisible("#widgetFieldDateRange", chromedp.ByID),
					chromedp.Click(`#widgetFieldDateRange`, chromedp.ByID), // Click the date range input form
					chromedp.Focus(`#startDate`, chromedp.ByID),
					chromedp.SendKeys(`#startDate`, strings.Repeat("\b", 10), chromedp.ByID),                     // Backspace the date start input field
					chromedp.SendKeys(`#startDate`, dateEnd[0]+"/\b"+dateEnd[1]+"/\b"+dateEnd[2], chromedp.ByID), // Send the start date to the input field
					chromedp.Focus(`#endDate`, chromedp.ByID),
					chromedp.SendKeys(`#endDate`, strings.Repeat("\b", 10), chromedp.ByID),                           // Backspace the date end input field
					chromedp.SendKeys(`#endDate`, dateToday[0]+"/\b"+dateToday[1]+"/\b"+dateToday[2], chromedp.ByID), // Send the end date to the input field
					chromedp.Click(`#applyBtn`, chromedp.ByID),                                                       // Click the apply button to retrive historical data
					chromedp.WaitVisible("#curr_table", chromedp.ByID),
					// chromedp.Nodes(`table[id="curr_table"] > tbody tr`, &nodeHistResults, chromedp.AtLeast(0), chromedp.ByQueryAll), // Append all the results to list
					chromedp.OuterHTML(`table[id="curr_table"] > tbody`, &res, chromedp.ByQuery), // Retrieve response string format start from result's table
				)
				if err != nil {
					log.Fatal(err)
				}

				// Rearrange the response string body
				rowData := strings.Split(res, "</tr>") // Individual result start with <tr> and end with </tr>
				var wg = sync.WaitGroup{}
				var ch = make(chan int, 1)
				wg.Add(len(rowData) - 1)
				for i := 0; i < len(rowData)-1; i++ {
					dataRealVal := regexp.MustCompile(`<td data-real-value="(.*?)"`) // Retrieve value from keywords <td data-real-value="(data need)"
					dataChange := regexp.MustCompile(`>(.*?)<`)                      // Retrieve "percentage changes" from last <tr> column without keyword "data-real-value"
					go func(i int) {
						splitTD := strings.Split(rowData[i], "/td>")
						ch <- i
						closePrice, _ := strconv.ParseFloat(dataRealVal.FindStringSubmatch(splitTD[1])[1], 64) // Close price
						openPrice, _ := strconv.ParseFloat(dataRealVal.FindStringSubmatch(splitTD[2])[1], 64)  // Open price
						highPrice, _ := strconv.ParseFloat(dataRealVal.FindStringSubmatch(splitTD[3])[1], 64)  // High price
						lowPrice, _ := strconv.ParseFloat(dataRealVal.FindStringSubmatch(splitTD[4])[1], 64)   // Low price
						volume, _ := strconv.ParseFloat(dataRealVal.FindStringSubmatch(splitTD[5])[1], 64)     // Volume
						priceList = append(priceList, &StockPrice{
							Date:   dataRealVal.FindStringSubmatch(splitTD[0])[1], // Date in unix format
							Close:  closePrice,
							Open:   openPrice,
							High:   highPrice,
							Low:    lowPrice,
							Volume: volume,
							Change: dataChange.FindStringSubmatch(splitTD[6])[1], // Percentage changes
						})
						<-ch
						wg.Done()
					}(i)
				}
				wg.Wait()
			}
			generalInfo := GetStockGeneralInfo(contextList[i].ctx, stockCode) // Get the general ratios from klsescreener
			stockPriceList = append(stockPriceList, &CompanyHistPrice{
				CompanyInfo:  contextList[i].stockinfo,
				HistPrice:    priceList,
				GeneralRatio: generalInfo,
			})
			contextList[i].cancel()
			wg.Done()
			<-myCH
		}(i)

	}
	wg.Wait()
	timeEnd := time.Since(timeStart)
	fmt.Println(timeEnd)
	return stockPriceList
}

// StockGeneralInfo = Stock ratios from klsesreener
type StockGeneralInfo struct {
	Title, Content string
}

// GetStockGeneralInfo = Get the stock general ratios from klsescreener
func GetStockGeneralInfo(ctx context.Context, stockcode string) []*StockGeneralInfo {
	url := "https://www.klsescreener.com/v2/stocks/view/" + string(stockcode)
	timeStart := time.Now()
	// Setup chromedp
	// options = append(chromedp.DefaultExecAllocatorOptions[:], options...) // append the chromedp options
	// allocCtx, cancel := chromedp.NewExecAllocator(context.Background(), options...)
	// defer cancel()
	// mainCtx, cancel := chromedp.NewContext(allocCtx)
	// defer cancel()
	// mainCtx, cancel = context.WithTimeout(mainCtx, 2*time.Minute) // timeout setting
	// defer cancel()

	var nodeInfo []*cdp.Node
	err := chromedp.Run(ctx, // Directly from the function input to use the chrome tab
		chromedp.Navigate(url),
		chromedp.WaitVisible("#page", chromedp.ByID),
		chromedp.Nodes(`div[class="table-responsive"] > table[class="stock_details table table-hover table-striped table-theme"] > tbody > tr`, &nodeInfo, chromedp.AtLeast(0), chromedp.ByQueryAll),
	)
	if err != nil {
		log.Fatal(err)
	}
	var listInfo []*StockGeneralInfo
	for i := 0; i < len(nodeInfo); i++ {
		data := &StockGeneralInfo{}
		err := chromedp.Run(ctx,
			chromedp.Text(`td:nth-child(1)`, &data.Title, chromedp.AtLeast(0), chromedp.ByQuery, chromedp.FromNode(nodeInfo[i])),
			chromedp.Text(`td:nth-child(2)`, &data.Content, chromedp.AtLeast(0), chromedp.ByQuery, chromedp.FromNode(nodeInfo[i])),
		)
		if err != nil {
			log.Fatal(err)
		}
		listInfo = append(listInfo, data)
	}

	timeEnd := time.Since(timeStart)
	fmt.Println(timeEnd)
	return listInfo
}
