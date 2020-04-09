package main

import (
	"bytes"
	"os"
	"regexp"
	"strings"
)

var tagWithNoEnd []string = []string{"area", "base", "br", "col", "command", "!doctype", "embed", "hr", "img", "input", "keygen", "link", "meta", "param", "source", "track", "wbr"}

type JavaScriptFunc struct { //Estrutura que vai guardar a informação sobre a função em javascript: Nome da fnção, Argumentos de Entrada,O inicio e o fim da função
	FunctionName               string
	InputParameters            []string
	BeginFunction, EndFunction int
}

type CssObject struct { //Estrutura que vai o objecto declarado no script <style>. Vou guardar o nome do objecto, o nome da pseudoClass caso exista, e a propriedade e o seu respetivo valor
	Selector    string
	PseudoClass string
	ProVal      map[string]string
}
type HtmlElement struct {
	Element                              string
	AttrValue                            map[string]string
	BeginAttr, EndAttr, idxBegin, idxEnd int
	MyJSFunc                             []JavaScriptFunc
	MyCSSObject                          []CssObject
	myHTMLElementDown                    []*HtmlElement
	myHTMLElementUp                      *HtmlElement
}

var myFirstHTMLElement *HtmlElement

func ParseHTMLFile(file string) bool {

	fd, err := os.Open(file)
	if err != nil {
		return false
	}
	defer fd.Close()
	end, err := fd.Seek(0, 2)
	readScript := make([]byte, end)
	fd.ReadAt(readScript, 0)
	padraoHTMLtag := regexp.MustCompile("<[^<>]+>")          //Expressao para encontrar todos as html tag do ficheiro. Inclui a tag de abertura e a tag que fecha
	htmlTAGidx := padraoHTMLtag.FindAllIndex(readScript, -1) //Encontrar as html tags. Recebo o index do inicio e do fim da html tag
	if htmlTAGidx == nil {                                   //Nao encontrei nenhuma tag. este não deve ser um ficheiro html
		return false
	}
	mapToTag := make(map[string]int, len(htmlTAGidx))
	mapToTag2 := make(map[string][]*HtmlElement)
	var htmlElementUP *HtmlElement = nil
	padraoHTMLAttrBegin := regexp.MustCompile("^<[\\w!]+(\\s|>)") //Expressao para encontrar o nome da tag numa abertura de tag <.... >
	padraoHTMLAttrEnd := regexp.MustCompile("<\\/\\w+>")          //Expressao para encontrar o nome da tag no fecho de uma tag </...>
	var aux string
	var aux1 []byte
	myChannArray := make([]chan int8, 0, 10)
	myChannArrayIDX := make([]int, cap(myChannArray))
	myChannActiveIDX := make([]bool, cap(myChannArray))
	var maximoAvailable int = cap(myChannArray)
	var quantidade int
	for i := 0; i < cap(myChannArray); i++ {
		myChannArray = append(myChannArray, make(chan int8, 2))
		myChannArrayIDX[i] = i
		myChannActiveIDX[i] = false
	}
	for i := 0; i < len(htmlTAGidx); i++ {
		aux1 = padraoHTMLAttrBegin.Find(readScript[htmlTAGidx[i][0]:htmlTAGidx[i][1]]) //descobrir qual é o nome da tag de abertura <.... >
		if aux1 == nil {                                                               //nao deve ser uma tag de abertura <.... >
			aux1 = padraoHTMLAttrEnd.Find(readScript[htmlTAGidx[i][0]:htmlTAGidx[i][1]]) //descobrir qual é o nome da tag de fecho </..>
			if aux1 == nil {                                                             //tambem não é uma tag de fecho. Por isso continuo
				continue
			}
			aux = strings.ToLower(string(aux1[2 : len(aux1)-1])) //Obter o nome da tag, nao esquecendo que o vetor aux1 é contituido por [<,/,....,>] e é por isto que quero desde o 2 index ate ao length-1, não incluido
			if encontrarTagwithNoend(aux) {                      //ver se a html tag encontrada </....> é uma das tag que nao tem end tag. Caso o seja vamos corrigir supondo que é uma tag de abertura dessa tag
				auxPointer := HtmlElement{Element: aux, BeginAttr: htmlTAGidx[i][0], EndAttr: htmlTAGidx[i][1] + 1, myHTMLElementUp: htmlElementUP, myHTMLElementDown: nil, idxBegin: i, idxEnd: i}
				htmlElementUP.myHTMLElementDown = append(htmlElementUP.myHTMLElementDown, &auxPointer)
				parseElement(&auxPointer, htmlTAGidx[i], htmlTAGidx[i], readScript, myChannArray, myChannArrayIDX, myChannActiveIDX, &quantidade, &maximoAvailable)
			} else {
				_, ok := mapToTag[aux] //ver se o nome ja foi registado, ou seja,se ja existe
				if ok == false {       // nao existe, entao para o proximo
					continue
				}
				mapToTag[aux]--        //a tag ja estava registada
				if mapToTag[aux] < 0 { // se o valor associado à tag for negativa,quer dizer que não temos nenhuma abertura feita da tag <... > para este fecho </...>
					mapToTag[aux] = 0
					continue
				}
				htmlElementSend := mapToTag2[aux][mapToTag[aux]]
				parseElement(htmlElementSend, htmlTAGidx[htmlElementSend.idxBegin], htmlTAGidx[i], readScript, myChannArray, myChannArrayIDX, myChannActiveIDX, &quantidade, &maximoAvailable)
				htmlElementSend.EndAttr = htmlTAGidx[i][0]
				htmlElementSend.idxEnd = i
				htmlElementUP = htmlElementSend.myHTMLElementUp
				mapToTag2[aux] = mapToTag2[aux][0 : len(mapToTag2[aux])-1]

			}

			continue
		}

		aux = strings.ToLower(string(aux1[1 : len(aux1)-1]))
		htmlElementaux := &HtmlElement{Element: aux, BeginAttr: htmlTAGidx[i][1] + 1, myHTMLElementUp: htmlElementUP, myHTMLElementDown: make([]*HtmlElement, 0, 10), idxBegin: i}
		if encontrarTagwithNoend(aux) {
			htmlElementaux.EndAttr = htmlTAGidx[i][1] + 1
			htmlElementaux.BeginAttr = htmlTAGidx[i][0]
			htmlElementaux.idxEnd = i

			if htmlElementUP == nil {
				htmlElementUP = htmlElementaux
				myFirstHTMLElement = htmlElementaux
				parseElement(htmlElementaux, htmlTAGidx[htmlElementaux.idxBegin], htmlTAGidx[i], readScript, myChannArray, myChannArrayIDX, myChannActiveIDX, &quantidade, &maximoAvailable)
				continue
			}
			htmlElementUP.myHTMLElementDown = append(htmlElementUP.myHTMLElementDown, htmlElementaux)
			parseElement(htmlElementaux, htmlTAGidx[htmlElementaux.idxBegin], htmlTAGidx[i], readScript, myChannArray, myChannArrayIDX, myChannActiveIDX, &quantidade, &maximoAvailable)

		} else {
			if _, ok := mapToTag2[aux]; ok == false {
				mapToTag2[aux] = make([]*HtmlElement, 0, 10)
			}
			mapToTag2[aux] = append(mapToTag2[aux], htmlElementaux)
			mapToTag[aux]++
			if htmlElementUP == nil {
				htmlElementUP = htmlElementaux
				myFirstHTMLElement = htmlElementaux
				continue
			}
			htmlElementUP.myHTMLElementDown = append(htmlElementUP.myHTMLElementDown, htmlElementaux)
			htmlElementUP = htmlElementaux
		}
	}
	for i := 0; i < len(myChannArray); i++ {
		if myChannActiveIDX[i] == true {
			<-myChannArray[i]
		}
	}
	return true
}

func encontrarTagwithNoend(tag string) bool {
	for i := 0; i < len(tagWithNoEnd); i++ {
		if strings.Compare(tag, tagWithNoEnd[i]) == 0 {
			return true
		}
	}
	return false
}

func parseElement(elementToParse *HtmlElement, openTag []int, endTag []int, readScript []byte, canaisComm []chan int8, canaisAvailableIDX []int, canaisAvailable []bool, quantidadeUsada, maximoDisponivel *int) {

	padraoHTMLAttrEnd := regexp.MustCompile(`(?m)(?P<key>\b\w+\b)\s*=\s*"(?P<value>[^"]*)"+`)
	findMyattrValue := padraoHTMLAttrEnd.FindAllSubmatchIndex(readScript[openTag[0]:openTag[1]], -1)
	if findMyattrValue != nil {
		elementToParse.AttrValue = make(map[string]string)
		for _, r := range findMyattrValue {
			elementToParse.AttrValue[string(readScript[openTag[0]+r[2]:openTag[0]+r[3]])] = string(readScript[openTag[0]+r[4] : openTag[0]+r[5]])
		}
	}

	if strings.Compare("script", elementToParse.Element) == 0 {
		if *quantidadeUsada == *maximoDisponivel {
			*quantidadeUsada = 0
			*maximoDisponivel = 0
			for *maximoDisponivel == 0 {
				*maximoDisponivel = reformularCanaisDisponiveis(canaisComm, canaisAvailableIDX, canaisAvailable)
			}
		}
		go parseScriptContentTag(readScript, openTag[1], endTag[0], elementToParse, canaisComm[canaisAvailableIDX[*quantidadeUsada]])
		canaisAvailable[canaisAvailableIDX[*quantidadeUsada]] = true
		*quantidadeUsada++
	} else if strings.Compare("style", elementToParse.Element) == 0 {
		if *quantidadeUsada == *maximoDisponivel {
			*quantidadeUsada = 0
			*maximoDisponivel = 0
			for *maximoDisponivel == 0 {
				*maximoDisponivel = reformularCanaisDisponiveis(canaisComm, canaisAvailableIDX, canaisAvailable)
			}
		}
		go parseStyleContentTag(readScript, openTag[1], endTag[0], elementToParse, canaisComm[canaisAvailableIDX[*quantidadeUsada]])
		canaisAvailable[canaisAvailableIDX[*quantidadeUsada]] = true
		*quantidadeUsada++
	}
	return
}

func parseStyleContentTag(readscript []byte, beginOfContentScript, endOfContentScript int, htmlElementMy *HtmlElement, ch chan int8) {

	offset := beginOfContentScript
	findOpenBrac := bytes.Index(readscript[offset:endOfContentScript], []byte("{"))
	var findCloseBrac int
	myListofCSS := make([]CssObject, 0, 10)
	myChannArray := make([]chan int8, 0, 10)
	var myChannIDXarray = make([]int, cap(myChannArray))
	var myChannActive = make([]bool, cap(myChannArray))
	for i := 0; i < cap(myChannArray); i++ {
		myChannArray = append(myChannArray, make(chan int8, 2))
		myChannIDXarray[i] = i
		myChannActive[i] = false
	}
	var maximoAvailable int = len(myChannArray)
	var quantidade int
	var idxEnd, idxBegin, idxDot = -1, -1, -1
	for findOpenBrac > 0 {
		findOpenBrac += offset
		findCloseBrac = bytes.Index(readscript[findOpenBrac:endOfContentScript], []byte("}"))
		if findCloseBrac < 0 {
			break
		}
		findCloseBrac += findOpenBrac
		myListofCSS = append(myListofCSS, CssObject{Selector: "unknow", PseudoClass: "unknow"})
		if quantidade == maximoAvailable {
			quantidade = 0
			maximoAvailable = 0
			for maximoAvailable == 0 {
				maximoAvailable = reformularCanaisDisponiveis(myChannArray, myChannIDXarray, myChannActive)
			}
		}
		go readAttrValueCSS(readscript[findOpenBrac:findCloseBrac], &myListofCSS[len(myListofCSS)-1], myChannArray[myChannIDXarray[quantidade]])
		myChannActive[myChannIDXarray[quantidade]] = true
		quantidade++
		for i := findOpenBrac - 1; true; i-- {
			if readscript[i] != ' ' && idxEnd == -1 {
				idxEnd = i + 1
				idxDot = i + 1
			} else if readscript[i] == '.' {
				idxDot = i
			} else if readscript[i] == ' ' && idxEnd != -1 {
				idxBegin = i + 1
			}
		}
		myListofCSS[len(myListofCSS)-1].Selector = string(readscript[idxBegin:idxDot])
		if idxDot < idxEnd {
			myListofCSS[len(myListofCSS)-1].PseudoClass = string(readscript[idxDot+1 : idxEnd])
		}
		offset = findCloseBrac + 1
		findOpenBrac = bytes.Index(readscript[offset:endOfContentScript], []byte("{"))
	}
	for i := 1; i < len(myChannArray); i++ {
		if myChannActive[i] == true {
			<-myChannArray[i]
		}
	}
	ch <- 1
	return
}

func readAttrValueCSS(readScript []byte, myCSS *CssObject, myChann chan int8) {

	regexpFindAttrValue := regexp.MustCompile(`(?m)(?P<key>\w+):\s+(?P<value>\w+);`)
	allIndexes := regexpFindAttrValue.FindAllSubmatchIndex(readScript, -1)
	if allIndexes == nil {
		myChann <- 1
		return
	}
	myCSS.ProVal = make(map[string]string, len(allIndexes))
	for _, r := range allIndexes {
		myCSS.ProVal[string(readScript[r[2]:r[3]])] = string(readScript[r[4]:r[5]])
	}
	myChann <- 1
	return
}

func parseScriptContentTag(readscript []byte, beginOfContentScript, endOfContentScript int, htmlElementMy *HtmlElement, ch chan int8) {

	funcIDX := bytes.Index(readscript[beginOfContentScript:endOfContentScript+1], []byte("function"))
	if funcIDX < 0 {
		ch <- 1
		return
	}

	functionWordSize := 8
	var firstOpenBrecet, offset int = 0, beginOfContentScript
	myRegExpression := regexp.MustCompile(`\w+|([\w*[,]*])`)
	myRegExpressionFunction := regexp.MustCompile(`function\s+\w+\s*\([\w*,]*\)\s*{`)
	myJSfunction := make([]JavaScriptFunc, 0, 10)
	myChannArray := make([]chan int8, 0, 10)
	myChannIDXArray, myChannIDXActive := make([]int, cap(myChannArray)), make([]bool, cap(myChannArray))
	for i := 0; i < cap(myChannArray); i++ {
		myChannArray = append(myChannArray, make(chan int8, 2))
		myChannIDXArray[i] = i
		myChannIDXActive[i] = false
	}
	var maximoAvailable int = len(myChannArray)
	var quantidade int
	for funcIDX > 0 {
		funcIDX += offset //obter o verdadeiro iDX no script
		firstOpenBrecet = bytes.Index(readscript[funcIDX+functionWordSize+1:endOfContentScript+1], []byte("{"))
		if firstOpenBrecet < 0 {
			offset = funcIDX + functionWordSize + 1
			funcIDX = bytes.Index(readscript[offset:endOfContentScript+1], []byte("function"))
			continue
		}
		firstOpenBrecet += funcIDX + functionWordSize + 1
		if myRegExpressionFunction.Match(readscript[funcIDX:firstOpenBrecet+1]) == false {
			offset = funcIDX + functionWordSize + 1
			funcIDX = bytes.Index(readscript[offset:endOfContentScript+1], []byte("function"))
			continue
		}
		auxJS := JavaScriptFunc{BeginFunction: firstOpenBrecet + 1}
		funcHeader := myRegExpression.FindAll(readscript[funcIDX+functionWordSize+1:firstOpenBrecet], -1)
		if funcHeader == nil {
			offset = funcIDX + functionWordSize + 1
			funcIDX = bytes.Index(readscript[offset:endOfContentScript+1], []byte("function"))
			continue
		}
		auxJS.FunctionName = string(funcHeader[0])
		if len(funcHeader) != 1 {
			auxJS.InputParameters = make([]string, len(funcHeader)-1)
			for i := 1; i < len(funcHeader); i++ {
				auxJS.InputParameters[i-1] = string(funcHeader[i])
			}
		}
		myJSfunction = append(myJSfunction, auxJS)
		if quantidade == maximoAvailable {
			quantidade = 0
			maximoAvailable = 0
			for maximoAvailable == 0 {
				maximoAvailable = reformularCanaisDisponiveis(myChannArray, myChannIDXArray, myChannIDXActive)
			}
		}
		go findEndOfJSfunction(&myJSfunction[len(myJSfunction)-1], readscript, firstOpenBrecet+1, endOfContentScript+1, myChannArray[myChannIDXArray[quantidade]])
		myChannIDXActive[myChannIDXArray[quantidade]] = true
		quantidade++
		offset = firstOpenBrecet + 1
		funcIDX = bytes.Index(readscript[offset:endOfContentScript+1], []byte("function"))
	}

	if len(myJSfunction) == 0 {
		htmlElementMy.MyJSFunc = nil
		ch <- 1
		return
	}
	for i := 0; i < len(myChannArray); i++ {
		if myChannIDXActive[i] == true {
			<-myChannArray[i]
		}
	}
	for i := 0; i < len(myJSfunction); i++ {
		if myJSfunction[i].EndFunction == -1 {
			copy(myJSfunction[i:len(myJSfunction)-1], myJSfunction[i+1:len(myJSfunction)])
			myJSfunction = myJSfunction[:len(myJSfunction)-1]
		}
	}

	htmlElementMy.MyJSFunc = myJSfunction
	ch <- 1
	return
}

func findEndOfJSfunction(myJS *JavaScriptFunc, readScript []byte, beginOfSearch, endOfSearch int, mychann chan int8) {

	contagem := 1
	offset := beginOfSearch
	for ; contagem != 0 && beginOfSearch < endOfSearch; offset++ {
		aux := bytes.IndexAny(readScript[offset:endOfSearch], "{}")
		offset += aux
		if readScript[offset] == '{' {
			contagem++
		} else {
			contagem--
		}
	}

	if contagem != 0 {
		mychann <- -1
		myJS.EndFunction = -1
		return
	}
	myJS.EndFunction = offset
	mychann <- 1
	return
}

func reformularCanaisDisponiveis(arrayOfChannels []chan int8, idxChannel []int, activeChannel []bool) (disponivel int) {

	disponivel = 0
	for i := 0; i < len(arrayOfChannels); i++ {
		if len(arrayOfChannels[i]) > 0 {
			<-arrayOfChannels[i]
			idxChannel[disponivel] = i
			activeChannel[i] = false
			disponivel++
		}

	}
	return
}

//RecoverAttrInTag
//Output: Copy of the html elements which tag name is equal to tag and a array of strings
//with the value of the attribute attr of those html elements
//Objective: Return an array of html elements copy which tag is equal to the input argument "tag"
func RecoverAttrInTag(tag, attr string) (array []HtmlElement, value []string) {
	myHTMLelementArray := make([]*HtmlElement, 0, 10)                                           //create an array of pointers which will point to the html elements we want
	myHTMLelementArray = recoverAttrInTagRec(tag, attr, myHTMLelementArray, myFirstHTMLElement) //find html elements with tag name equal to tag and with value assign to attribute attr
	value = make([]string, len(myHTMLelementArray))                                             //array of strings which will be saved the value of the attr attributes
	array = make([]HtmlElement, len(myHTMLelementArray))                                        //array of html elements which will be the destination of a copy of those html elements with tag name equal to tag and assigned value of attr attribute
	for i := 0; i < len(myHTMLelementArray); i++ {                                              //copy each html element found to the array and extract the value of attr attribute
		array[i] = *myHTMLelementArray[i]
		value[i] = myHTMLelementArray[i].AttrValue[attr]
	}
	return
}

//recoverAttrInTagRec
//Output: array of pointers to html elements with tag equal to tag and value assign to attr attribute
//Objective: find html elements with tag equal to tag and value assign to attr attribute recursively
func recoverAttrInTagRec(tag, attr string, htmlArray []*HtmlElement, Start *HtmlElement) []*HtmlElement {

	if Start == nil { //if we think the struct as a tree, if start htmlElement pointer is nil, means that i don't have a tree, or that i reach the end of a branch
		return htmlArray
	}
	for i := 0; i < len(Start.myHTMLElementDown); i++ { //go to all the potential leafs or new branchs that belong to the Start branch
		htmlArray = recoverAttrInTagRec(tag, attr, htmlArray, Start.myHTMLElementDown[i])
	}

	if strings.Compare(Start.Element, tag) == 0 { //see if Start has a tag name equal to tag
		if _, ok := Start.AttrValue[attr]; ok == true { //see if the Start html element has a attribute call attr
			htmlArray = append(htmlArray, Start)
		}
	}
	return htmlArray
}
