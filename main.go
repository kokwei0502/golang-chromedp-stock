package main

import (
	"log"
	"net/http"

	"github.com/kokwei0502/golang-chromedp-stock/controllers"
	"github.com/kokwei0502/golang-chromedp-stock/routers"
)

func main() {
	controllers.RetrieveAllTemplate()
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	http.HandleFunc("/", routers.StockListingIndexPage)
	http.HandleFunc("/stock", routers.IndividualStockInfoIndexPage)
	http.HandleFunc("/stocklist", routers.SectorIndustryListingIndexPage)
	log.Println("Listening on :8080...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}

	// controllers.GetStockGeneralInfo("6947")
	// controllers.GetStockInfoInvesting("https://www.investing.com/equities/tenaga-nasional-bhd", -10)
	// controllers.UpdateStockListing()
}
