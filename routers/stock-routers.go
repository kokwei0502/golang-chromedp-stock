package routers

import (
	"fmt"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kokwei0502/golang-malaysia-stock-analysis/controllers"
)

var (
	workingDir   string
	listingData  *controllers.StockListing
	categoryList category
)

func init() {
	workingDir, _ = os.Getwd()
	listingData = controllers.RetrieveStockListing()
	sortAlphabet(listingData.StockList, "A-Z")

}

// StockListingIndexPage = All stock listing
func StockListingIndexPage(w http.ResponseWriter, r *http.Request) {
	categoryList = createSectorIndList(listingData)                         // Get the category list
	listingData.StockList = convertMarketCaptoString(listingData.StockList) // Convert market cap from int to string
	if r.Method == "POST" {
		r.ParseForm()
		sortAZRest := r.PostFormValue("sort-submit")     // Sort alphabet
		sortSector := r.PostFormValue("sort-sector")     // Sort sector to generate new category list
		sortIndustry := r.PostFormValue("sort-industry") // Sort industry to generate new industry list
		companySymbol := r.PostFormValue("company-info") // Individual company submit with graph
		sectorInfo := r.PostFormValue("sector-info")     // Get all company listing in sector with graph
		IndustryInfo := r.PostFormValue("industry-info") // Get all company listing in industry with graph
		if sortAZRest != "" {
			switch sortAZRest {
			case "sort-reset": // Reset all datas to original
				listingData = controllers.RetrieveStockListing()
			default:
				listingData.StockList = sortingAtoZ(listingData.StockList, sortAZRest)
			}
		} else if strings.ToLower(sortSector) != "choose sector" {
			listingData.StockList = sortSectorIndustry(listingData.StockList, "sector", sortSector)

		} else if strings.ToLower(sortIndustry) != "choose industry" {
			listingData.StockList = sortSectorIndustry(listingData.StockList, "industry", sortIndustry)
		} else if companySymbol != "" {
			var indexNum string
			for k, v := range listingData.StockList {
				if v.Symbol == strings.Split(companySymbol, "@")[0] {
					indexNum = strconv.Itoa(k)
				}
			}
			http.Redirect(w, r, "/stock?index="+indexNum, http.StatusSeeOther)
			return
		} else if sectorInfo != "" {
			http.Redirect(w, r, "/stocklist?sector="+sectorInfo, http.StatusSeeOther)
			return
		} else if IndustryInfo != "" {
			http.Redirect(w, r, "/stocklist?industry="+IndustryInfo, http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)

	}
	pageData := struct {
		Title            string
		StockListingData *controllers.StockListing
		CategoryList     category
	}{
		Title:            "Stock Listing",
		StockListingData: listingData,
		CategoryList:     categoryList,
	}
	controllers.AllHTMLTemplates.ExecuteTemplate(w, "stocklisting.html", pageData)
}

// SectorIndustryListingIndexPage = Index page for company listing with graph either in sector or industry
func SectorIndustryListingIndexPage(w http.ResponseWriter, r *http.Request) {
	timeStart := time.Now()
	sectorParams := r.URL.Query().Get("sector")
	industryParams := r.URL.Query().Get("industry")
	stockListing := controllers.RetrieveStockListing()
	var filterList []*controllers.StockInfo
	if sectorParams != "" {
		sectorParams = strings.ReplaceAll(strings.Split(r.URL.RawQuery, "sector=")[1], "%20", " ") // r.URL.RawQuery = get the raw URL string format start from keyword "sector="
		for k := range stockListing.StockList {
			if stockListing.StockList[k].Sector == sectorParams {
				filterList = append(filterList, stockListing.StockList[k])
			}
		}
	} else if industryParams != "" {
		industryParams = strings.ReplaceAll(strings.Split(r.URL.RawQuery, "industry=")[1], "%20", " ") // r.URL.RawQuery = get the raw URL string format start from keyword "industry="
		for k := range stockListing.StockList {
			if strings.ToLower(stockListing.StockList[k].Industry) == strings.ToLower(industryParams) {
				filterList = append(filterList, stockListing.StockList[k])
			}
		}
	}

	listCompanyHistPrice := controllers.GetStockInfoInvesting(filterList, -10) // Get all listing for info, historical prices and ratios
	for i := 0; i < len(listCompanyHistPrice); i++ {
		if listCompanyHistPrice[i].CompanyInfo.Symbol != "" {
			stockPrice := listCompanyHistPrice[i].HistPrice
			// Sort the date
			sort.Slice(stockPrice, func(i, j int) bool {
				dateX, _ := strconv.ParseInt(stockPrice[i].Date, 10, 64)
				dateXI := time.Unix(dateX, 0)
				dateY, _ := strconv.ParseInt(stockPrice[j].Date, 10, 64)
				dateYI := time.Unix(dateY, 0)
				return dateXI.Before(dateYI)
			})
			for _, v := range stockPrice {
				i, _ := strconv.ParseInt(v.Date, 10, 64)
				v.Date = time.Unix(i, 0).Format("2006-01-02") // Change the date format to "yyyy-mm-dd" to match the graphing data
			}
		}
	}

	// Page data render
	pageData := struct {
		StockListing []*controllers.CompanyHistPrice
	}{StockListing: listCompanyHistPrice}
	timeEnd := time.Since(timeStart)
	fmt.Println(timeEnd)
	controllers.AllHTMLTemplates.ExecuteTemplate(w, "multiplestock.html", pageData)

}

// IndividualStockInfoIndexPage = Index page for individual company data
func IndividualStockInfoIndexPage(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.URL.Query().Get("index"))
	index, _ := strconv.Atoi(r.URL.Query().Get("index"))
	var stockList []*controllers.StockInfo
	stockList = append(stockList, listingData.StockList[index])
	stockHistPrice := controllers.GetStockInfoInvesting(stockList, -10)
	// fmt.Println(listingData.StockList[index].Company)
	for k := range stockHistPrice {
		sort.Slice(stockHistPrice[k].HistPrice, func(i, j int) bool {
			dateX, _ := strconv.ParseInt(stockHistPrice[k].HistPrice[i].Date, 10, 64)
			dateXI := time.Unix(dateX, 0)
			dateY, _ := strconv.ParseInt(stockHistPrice[k].HistPrice[j].Date, 10, 64)
			dateYI := time.Unix(dateY, 0)
			return dateXI.Before(dateYI)
		})
		for _, v := range stockHistPrice[k].HistPrice {
			i, _ := strconv.ParseInt(v.Date, 10, 64)
			v.Date = time.Unix(i, 0).Format("2006-01-02")
		}
	}

	// generalInfo := controllers.GetStockGeneralInfo(listingData.StockList[index].StockCode)
	pageData := struct {
		StockInvestingInfo *controllers.CompanyHistPrice
	}{StockInvestingInfo: stockHistPrice[0]}
	controllers.AllHTMLTemplates.ExecuteTemplate(w, "stockinfo.html", pageData)
}

func sortSectorIndustry(data []*controllers.StockInfo, category string, keyword string) []*controllers.StockInfo {
	var x []*controllers.StockInfo
	switch category {
	case "sector":
		for k := range data {
			if data[k].Sector == keyword {
				x = append(x, data[k])
			}
		}
	case "industry":
		for k := range data {
			if data[k].Industry == keyword {
				x = append(x, data[k])
			}
		}
	}
	if len(x) == 0 {
		x = data
	}
	return x
}

func sortingAtoZ(data []*controllers.StockInfo, subval string) []*controllers.StockInfo {
	var stringCont bool
	listKeywords := []string{"market", "company"}
	for k := range listKeywords {
		keyword := listKeywords[k]
		stringCont = strings.Contains(subval, keyword)
		if stringCont {
			seq := subval[len(subval)-3:]
			switch keyword {
			case "market":
				convertMarketCaptoFloat(data)
				sortFloat(data, seq)
				convertMarketCaptoString(data)
			case "company":
				sortAlphabet(data, seq)
			}
		}
	}
	return data
}

func convertMarketCaptoString(data []*controllers.StockInfo) []*controllers.StockInfo {
	for _, v := range data {
		switch v.MarketCap.(type) {
		case float64:
			if (v.MarketCap.(float64) / 1000000) < 1000 {
				v.MarketCap = fmt.Sprintf(`%.2fM`, (math.Round((v.MarketCap.(float64)/1000000)*100) / 100))
			} else if (v.MarketCap.(float64) / 1000000000) < 1000 {
				v.MarketCap = fmt.Sprintf(`%.2fB`, (math.Round((v.MarketCap.(float64)/1000000000)*100) / 100))
			}
		}
	}
	return data
}

func convertMarketCaptoFloat(data []*controllers.StockInfo) {
	for _, v := range data {
		switch v.MarketCap.(type) {
		case string:
			var marketcapStr string
			lastI := v.MarketCap.(string)[len(v.MarketCap.(string))-1]
			switch string(lastI) {
			case "M":
				marketcapStr = strings.ReplaceAll(v.MarketCap.(string), "M", "")
				v.MarketCap, _ = strconv.ParseFloat(marketcapStr, 64)
				v.MarketCap = v.MarketCap.(float64) * 1000000
			case "B":
				marketcapStr = strings.ReplaceAll(v.MarketCap.(string), "B", "")
				v.MarketCap, _ = strconv.ParseFloat(marketcapStr, 64)
				v.MarketCap = v.MarketCap.(float64) * 1000000000
			}
		}
	}
}

func sortAlphabet(data []*controllers.StockInfo, seq string) {
	switch seq {
	case "A-Z":
		sort.Slice(data, func(i, j int) bool {
			return strings.Title(data[i].Company) < strings.Title(data[j].Company)
		})
	case "Z-A":
		sort.Slice(data, func(i, j int) bool {
			return strings.Title(data[j].Company) < strings.Title(data[i].Company)
		})
	}

}

func sortFloat(data []*controllers.StockInfo, seq string) {
	switch seq {
	case "A-Z":
		sort.Slice(data, func(i, j int) bool {
			return data[i].MarketCap.(float64) < data[j].MarketCap.(float64)
		})
	case "Z-A":
		sort.Slice(data, func(i, j int) bool {
			return data[i].MarketCap.(float64) > data[j].MarketCap.(float64)
		})
	}

}

type category struct {
	ListSector   []string
	ListIndustry []string
}

func createSectorIndList(data *controllers.StockListing) category {
	times := time.Now()
	dataList := data.StockList
	mapSector := make(map[string]bool)
	mapIndustry := make(map[string]bool)
	var listSector = []string{"Choose Sector"}
	var listIndustry = []string{"Choose Industry"}

	for k := range dataList {

		if _, ok := mapSector[dataList[k].Sector]; !ok {
			if !strings.HasPrefix(dataList[k].Sector, "-") {
				mapSector[dataList[k].Sector] = true
				listSector = append(listSector, dataList[k].Sector)
			}
		}

	}

	for k := range dataList {
		if _, ok := mapIndustry[dataList[k].Industry]; !ok {
			if !strings.HasPrefix(dataList[k].Industry, "-") {
				mapIndustry[dataList[k].Industry] = true
				listIndustry = append(listIndustry, dataList[k].Industry)
			}
		}
	}

	categoryData := category{
		ListSector:   listSector,
		ListIndustry: listIndustry,
	}
	timee := time.Since(times)
	fmt.Println(timee)
	return categoryData

}
