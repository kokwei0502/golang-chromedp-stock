package controllers

import (
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"unicode"
)

// GeneralInfoKLSE = Data structure for ratios data from klsescreener.com
type GeneralInfoKLSE struct {
	Title, Content string
}

const (
	klseBaseURL = "https://www.klsescreener.com/v2/stocks/view/"
)

// GetGeneralInfoKLSE = Get basic ratio data from klsecreener.com
func GetGeneralInfoKLSE(stockcode string) []*GeneralInfoKLSE {
	res, err := http.Get(klseBaseURL + stockcode)
	if err != nil {
		log.Fatal(err)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal(err)
	}
	var listContent []*GeneralInfoKLSE
	stringRemoveSpace := removeSpace(string(body))                                                                 // Remove all white spaces and line breaks (easy to search strings)
	regMain := regexp.MustCompile(`<tableclass="stock_detailstabletable-hovertable-stripedtable-theme(.*?)table>`) // Extract strings in betwwen keywords set
	tableresult := regMain.FindStringSubmatch(stringRemoveSpace)
	if len(tableresult) > 0 { // Check whether found strings
		regTR := regexp.MustCompile(`<tr(.*?)tr>`) // Extract results from keywords
		totalTR := regTR.FindAllStringSubmatch(tableresult[1], -1)

		for k := range totalTR {
			regDet := regexp.MustCompile(`>(.*?)</td`) // Extract FINAL results from keywords
			listRes := regDet.FindAllStringSubmatch(totalTR[k][1], -1)

			// Append to result listing
			listContent = append(listContent, &GeneralInfoKLSE{
				Title: strings.Split(listRes[0][1], ">")[1], Content: strings.Split(listRes[1][1], ">")[1],
			})
		}
	}
	return listContent
}

func removeSpace(s string) string {
	rr := make([]rune, 0, len(s))
	for _, r := range s {
		if !unicode.IsSpace(r) {
			rr = append(rr, r)
		}
	}
	return string(rr)
}
