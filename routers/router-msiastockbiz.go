package routers

import (
	"log"
	"net/http"

	"github.com/kokwei0502/golang-chromedp-stock/controllers"
)

// IndexPageMsiaStockBiz = Index page for stock listing from malaysia.stockbiz
func IndexPageMsiaStockBiz(w http.ResponseWriter, r *http.Request) {
	stockListing := controllers.RetrieveMalaysiaStockBizListing()
	if r.Method == "POST" {
		if err := r.ParseForm(); err != nil {
			log.Fatal(err)
		}
		stockData := r.PostFormValue("stock-data")         // Get the individual stock code
		sectorData := r.PostFormValue("sector-data")       // Get the sector name
		subSectorData := r.PostFormValue("subsector-data") // Get the sub sector name
		if stockData != "" {
			http.Redirect(w, r, "/yahoofinance?code="+stockData, http.StatusSeeOther)
			return
		} else if sectorData != "" {
			http.Redirect(w, r, "/yflisting?sector="+sectorData, http.StatusSeeOther)
			return
		} else if subSectorData != "" {
			http.Redirect(w, r, "/yflisting?subsector="+subSectorData, http.StatusSeeOther)
			return
		}
	}
	pageData := struct {
		StockListing []*controllers.MalaysiaStockBizCompanyInfo
	}{StockListing: stockListing}
	controllers.AllHTMLTemplates.ExecuteTemplate(w, "malaysiastockbiz.html", pageData)
}
