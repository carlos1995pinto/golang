package main

import (
	"fmt"
	"htmlParser"
)

func main() {
	fileName := "webTeste.html"
	htmlParser.ParseHTMLFile(fileName)
	_,outvalues := htmlParser.RecoverAttr("href")
	for i:=0;i<len(outvalues);i++{
		fmt.Printf("Valor %d=%s\n",i,outvalues[i])
	}
}
