package controllers

import (
	"os/exec"
)

const (
	pyEXE        string = "C:/Python/Python39/python.exe"
	pyScript     string = "D:/khong-programming/python/python-stockinfo-investing/stockinfo.py"
	pyFunc       string = "--func"
	pyCountry    string = "--country"
	pySavepath   string = "--savepath"
	pyStockName  string = "--stock"
	pyYearSearch string = "--year"
)

type companyDetail struct {
	Country  string `json:"country"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Isin     string `json:"isin"`
	Currency string `json:"currency"`
	Symbol   string `json:"symbol"`
	Category string `json:"category"`
}

type PythonScriptArgs struct {
	FunctionAllStock     string
	FunctionStockInfo    string
	Country              string
	StockListingLocation string
	StockInfoLocation    string
	StockSymbol          string
	Year                 string
}

func RetrieveStockInfo(stock, country, year, savepath string) {
	cmd := exec.Command(pyEXE, pyScript, pyFunc, "get_stock", pyStockName, stock, pySavepath, savepath, pyCountry, country, pyYearSearch, year)
	cmd.Run()
}

// 'get_list', 'get_stock'
