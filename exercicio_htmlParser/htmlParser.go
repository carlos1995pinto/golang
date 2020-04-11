package htmlParser

import (
	"bytes"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

var tagWithNoEnd []string = []string{"area", "base", "br", "col", "command", "!doctype", "embed", "hr", "img", "input", "keygen", "link", "meta", "param", "source", "track", "wbr"}
//JavaScriptFunc struct:
//Estrutura que vai guardar a informação sobre a função em javascript: 
//Nome da função, Argumentos de Entrada,O inicio e o fim da função
type JavaScriptFunc struct {
	FunctionName               string
	InputParameters            []string
	BeginFunction, EndFunction int
}
//CSSObject struct:
//Estrutura que vai o objecto declarado no script <style>.
//Vou guardar o nome do objecto, o nome da pseudoClass caso exista, e a propriedade e o seu respetivo valor
type CSSObject struct { 
	Selector    string
	PseudoClass string
	ProVal      map[string]string
	valido bool
}
//HTMLElement struct:
//Estrutura que vai guardar a informação e conteudo relevante sobre uma tag de html presente no ficheiro
//Guarda os atributos e valores respectivos dessa tag, guarda o inicio e o fim dessa tag no ficheiro,
//caso a tag seja "script", guarda as funções. Caso a tag seja "style" guarda os objectos CSS mencionados e as suas propriedades 
//Tambem tem um ponteiro para os elementos tag que estão dentro desta tag e um ponteiro para o elemento tag anterior
type HTMLElement struct {
	Element                              string
	AttrValue                            map[string]string
	BeginAttr, EndAttr, idxBegin, idxEnd int
	MyJSFunc                             []JavaScriptFunc
	MyCSSObject                          []CSSObject
	myHTMLElementDown                    []*HTMLElement
	myHTMLElementUp                      *HTMLElement
}

type htmlTOthreads struct {
	tipo                 int8 //1->script//2->style
	begin,end			 int
	htmlObject           *HTMLElement
}

type htmlTothreadsInside struct {
	tipo int8 //1->script//2->style
	beginStyle, endStyle int
	cssobjectPTR         *CSSObject
	jsObjectPTR           *JavaScriptFunc
}




var numberOfCores int
var chan2htmlWorkVerifiy int64 

var chan2html chan htmlTOthreads
var chan2htmlBool bool
var chan2htmlWork int64 =0
var chan2htmlREP chan int8 
var mutexchan2html sync.Mutex

var mutexStyleScriptParse sync.Mutex
var mutexStyleScriptParseGO sync.Mutex

var chan2ScriptStyleWork int64 =0
var chan2CssJava chan htmlTothreadsInside
var chan2CssJavaBool bool


var  threadUPcount  int 
var  threadDOWNcount int

var myFirstHTMLElement *HTMLElement

var savefile string 

func init(){

	numberOfCores =  runtime.NumCPU()
	for i:=0;i<numberOfCores;i++{
		chan2htmlWorkVerifiy =chan2htmlWorkVerifiy | 0xFF
		chan2htmlWorkVerifiy = chan2htmlWorkVerifiy<<8
	}
	chan2htmlWork = chan2htmlWorkVerifiy
}
//ParseHTMLFile function:
//Input: html filename
//Objective: Extract each html tag,his attribute values if such exist,
//JavaScript function (including function name and input arguments),
//CSS objects (tag, pseudo-class and attribute-value)
func ParseHTMLFile(file string) bool {
	savefile = file
	chan2CssJavaBool = false
	chan2htmlBool = false
	chan2htmlWork = chan2htmlWorkVerifiy
	chan2ScriptStyleWork = chan2htmlWorkVerifiy
	threadUPcount = 0
	threadDOWNcount = 0
	myFirstHTMLElement= nil
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
	mapToTag2 := make(map[string][]*HTMLElement)
	var htmlElementUP *HTMLElement = nil
	padraoHTMLAttrBegin := regexp.MustCompile("^<[\\w!]+(\\s|>)") //Expressao para encontrar o nome da tag numa abertura de tag <.... >
	padraoHTMLAttrEnd := regexp.MustCompile("<\\/\\w+>")          //Expressao para encontrar o nome da tag no fecho de uma tag </...>
	var aux string
	var aux1 []byte

	for i := 0; i < len(htmlTAGidx); i++ {
		aux1 = padraoHTMLAttrBegin.Find(readScript[htmlTAGidx[i][0]:htmlTAGidx[i][1]]) //descobrir qual é o nome da tag de abertura <.... >
		if aux1 == nil {                                                               //nao deve ser uma tag de abertura <.... >
			aux1 = padraoHTMLAttrEnd.Find(readScript[htmlTAGidx[i][0]:htmlTAGidx[i][1]]) //descobrir qual é o nome da tag de fecho </..>
			if aux1 == nil {                                                             //tambem não é uma tag de fecho. Por isso continuo
				continue
			}
			aux = strings.ToLower(string(aux1[2 : len(aux1)-1])) //Obter o nome da tag, nao esquecendo que o vetor aux1 é contituido por [<,/,....,>] e é por isto que quero desde o 2 index ate ao length-1, não incluido
			if encontrarTagwithNoend(aux) {                      //ver se a html tag encontrada </....> é uma das tag que nao tem end tag. Caso o seja vamos corrigir supondo que é uma tag de abertura dessa tag
				auxPointer := HTMLElement{Element: aux, BeginAttr: htmlTAGidx[i][0], EndAttr: htmlTAGidx[i][1] + 1, myHTMLElementUp: htmlElementUP, myHTMLElementDown: nil, idxBegin: i, idxEnd: i}
				htmlElementUP.myHTMLElementDown = append(htmlElementUP.myHTMLElementDown, &auxPointer)
				parseElement(&auxPointer, htmlTAGidx[i], htmlTAGidx[i], readScript)
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
				parseElement(htmlElementSend, htmlTAGidx[htmlElementSend.idxBegin], htmlTAGidx[i], readScript)
				htmlElementSend.EndAttr = htmlTAGidx[i][0]
				htmlElementSend.idxEnd = i
				htmlElementUP = htmlElementSend.myHTMLElementUp
				mapToTag2[aux] = mapToTag2[aux][0 : len(mapToTag2[aux])-1]

			}

			continue
		}

		aux = strings.ToLower(string(aux1[1 : len(aux1)-1]))
		htmlElementaux := &HTMLElement{Element: aux, BeginAttr: htmlTAGidx[i][1] + 1, myHTMLElementUp: htmlElementUP, myHTMLElementDown: make([]*HTMLElement, 0, 10), idxBegin: i}
		if encontrarTagwithNoend(aux) {
			htmlElementaux.EndAttr = htmlTAGidx[i][1] + 1
			htmlElementaux.BeginAttr = htmlTAGidx[i][0]
			htmlElementaux.idxEnd = i

			if htmlElementUP == nil {
				htmlElementUP = htmlElementaux
				myFirstHTMLElement = htmlElementaux
				parseElement(htmlElementaux, htmlTAGidx[htmlElementaux.idxBegin], htmlTAGidx[i], readScript)
				continue
			}
			htmlElementUP.myHTMLElementDown = append(htmlElementUP.myHTMLElementDown, htmlElementaux)
			parseElement(htmlElementaux, htmlTAGidx[htmlElementaux.idxBegin], htmlTAGidx[i], readScript)

		} else {
			if _, ok := mapToTag2[aux]; ok == false {
				mapToTag2[aux] = make([]*HTMLElement, 0, 10)
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
	htmlElementToStopUpthreads := htmlTOthreads{tipo:-1}
	for i:=0;i<threadUPcount;i++{
		chan2html <-htmlElementToStopUpthreads
	}
	for i:=0;i<threadUPcount;i++{
		<-chan2htmlREP
		
	}
	close(chan2html)
	close(chan2htmlREP)
	cleanCSSJS(myFirstHTMLElement)
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

func parseElement(elementToParse *HTMLElement, openTag []int, endTag []int, readScript []byte) {

	padraoHTMLAttrEnd := regexp.MustCompile(`(?m)(?P<key>\b\w+\b)\s*=\s*"(?P<value>[^"]*)"+`)
	findMyattrValue := padraoHTMLAttrEnd.FindAllSubmatchIndex(readScript[openTag[0]:openTag[1]], -1)
	if findMyattrValue != nil {
		elementToParse.AttrValue = make(map[string]string)
		for _, r := range findMyattrValue {
			elementToParse.AttrValue[string(readScript[openTag[0]+r[2]:openTag[0]+r[3]])] = string(readScript[openTag[0]+r[4] : openTag[0]+r[5]])
		}
	}

	if strings.Compare("script", elementToParse.Element) == 0 {
		if threadUPcount < numberOfCores {
			if (chan2htmlBool == false){
				chan2html = make(chan htmlTOthreads, 10*(numberOfCores-1))
				chan2htmlREP = make(chan int8,numberOfCores)
				chan2htmlBool =true
			}
			if chan2htmlWork ^ chan2htmlWorkVerifiy == 0 {
				go parseContentTag(readScript, int8(threadUPcount))
				threadUPcount++
			}
		}
		MyHTMLElementToSend := htmlTOthreads{tipo:1,begin:openTag[1],end: endTag[0],htmlObject:elementToParse }
		chan2html<-MyHTMLElementToSend
	} else if strings.Compare("style", elementToParse.Element) == 0 {
		if threadUPcount < numberOfCores {
			if (chan2htmlBool == false){
				chan2html = make(chan htmlTOthreads, 10*(numberOfCores-1))
				chan2htmlREP = make(chan int8,numberOfCores)
				chan2htmlBool =true
			}
			if chan2htmlWork ^ chan2htmlWorkVerifiy == 0 {
				go parseContentTag(readScript, int8(threadUPcount))
				threadUPcount++
			}
		}
		MyHTMLElementToSend := htmlTOthreads{tipo:2,begin:openTag[1],end: endTag[0],htmlObject:elementToParse }
		chan2html<-MyHTMLElementToSend
	}
	return
}

func parseContentTag(readScript []byte, threadIDX int8){
	var MyHTMLElementReceived htmlTOthreads
	var auxWorker int64 = 1<<threadIDX
	chanToChildThreads := make(chan int8,numberOfCores)
	mythreadsLaunch :=0
	for true{
		
		mutexchan2html.Lock()
		chan2htmlWork = chan2htmlWork^auxWorker	
		MyHTMLElementReceived = <- chan2html
		chan2htmlWork = chan2htmlWork^auxWorker
		mutexchan2html.Unlock()
		if MyHTMLElementReceived.tipo == 1{
			parseScriptContentTag(readScript,MyHTMLElementReceived.begin,MyHTMLElementReceived.end,MyHTMLElementReceived.htmlObject,&mythreadsLaunch,chanToChildThreads)

		}else if MyHTMLElementReceived.tipo == 2{
			parseStyleContentTag(readScript,MyHTMLElementReceived.begin,MyHTMLElementReceived.end,MyHTMLElementReceived.htmlObject,&mythreadsLaunch,chanToChildThreads)
		}else{
			break
		}

	}
	justTofinish := htmlTothreadsInside{tipo:-1}
	for i:=0;i<mythreadsLaunch;i++{
		chan2CssJava <-justTofinish
	}
	for i:=0;i<mythreadsLaunch;i++{
		<-chanToChildThreads
	}
	close(chanToChildThreads)
	chan2htmlREP <- threadIDX
	return
}

func parseStyleContentTag(readscript []byte, beginOfContentScript, endOfContentScript int, htmlElementMy *HTMLElement,mythreadsLaunch *int,chanToChildThreads chan int8) {

	offset := beginOfContentScript
	findOpenBrac := bytes.Index(readscript[offset:endOfContentScript], []byte("{"))
	var findCloseBrac int
	htmlElementMy.MyCSSObject = make([]CSSObject, 0, 10)

	var idxEnd, idxBegin, idxDot = -1, -1, -1
	for findOpenBrac > 0 {
		findOpenBrac += offset
		findCloseBrac = bytes.Index(readscript[findOpenBrac:endOfContentScript], []byte("}"))
		if findCloseBrac < 0 {
			if(len(htmlElementMy.MyCSSObject)==0){
				htmlElementMy.MyCSSObject = nil
			}
			break
		}
		findCloseBrac += findOpenBrac
		htmlElementMy.MyCSSObject = append(htmlElementMy.MyCSSObject, CSSObject{Selector: "unknow", PseudoClass: "unknow"})
		mutexStyleScriptParseGO.Lock()
		if(threadDOWNcount <numberOfCores){
			if(chan2CssJavaBool == false){
				chan2CssJava = make(chan htmlTothreadsInside,10*(numberOfCores-1))
				chan2CssJavaBool = true
			}
			if(chan2ScriptStyleWork^chan2htmlWorkVerifiy ==0){
				go readStyleScriptObjectFunc(readscript,int8(threadDOWNcount),chanToChildThreads)
				*mythreadsLaunch++
				threadDOWNcount++
			}
			
		}
		mutexStyleScriptParseGO.Unlock()
		auxToSend :=htmlTothreadsInside{tipo:2,cssobjectPTR:&htmlElementMy.MyCSSObject[len(htmlElementMy.MyCSSObject)-1],beginStyle:findOpenBrac,endStyle:findCloseBrac}
		chan2CssJava<-auxToSend
	
		for i := findOpenBrac - 1; true; i-- {
			if readscript[i] != ' ' &&readscript[i] != '\n' && idxEnd == -1 {
				idxEnd = i + 1
				idxDot = i + 1
			} else if readscript[i] == '.' {
				idxDot = i
			} else if (readscript[i] == ' ' || readscript[i] =='\n') && idxEnd != -1 {
				idxBegin = i + 1
				break;
			}
		}
		htmlElementMy.MyCSSObject[len(htmlElementMy.MyCSSObject)-1].Selector = string(readscript[idxBegin:idxDot])
		if idxDot < idxEnd {
			htmlElementMy.MyCSSObject[len(htmlElementMy.MyCSSObject)-1].PseudoClass = string(readscript[idxDot+1 : idxEnd])
		}
		offset = findCloseBrac + 1
		findOpenBrac = bytes.Index(readscript[offset:endOfContentScript], []byte("{"))
		idxEnd, idxBegin, idxDot = -1, -1, -1
	}
	
	return
}

func readStyleScriptObjectFunc(readScript []byte,threadIDX int8, repChan chan int8){
	var myCSSJSObject htmlTothreadsInside
	var auxWorker int64 = 1<<threadIDX
	
	for true{
		mutexStyleScriptParse.Lock()
		chan2ScriptStyleWork = chan2ScriptStyleWork^auxWorker
		myCSSJSObject=<-chan2CssJava
		chan2ScriptStyleWork = chan2ScriptStyleWork^auxWorker
		mutexStyleScriptParse.Unlock()
		if myCSSJSObject.tipo == 1{
			findEndOfJSfunction(myCSSJSObject.jsObjectPTR,readScript[myCSSJSObject.beginStyle:myCSSJSObject.endStyle],myCSSJSObject.beginStyle)
		}else if myCSSJSObject.tipo == 2{
			readAttrValueCSS(readScript[myCSSJSObject.beginStyle:myCSSJSObject.endStyle],myCSSJSObject.cssobjectPTR)

		}else{
			break
		}
	}
	repChan <-1
	return
}

func readAttrValueCSS(readScript []byte, myCSS *CSSObject) {

	regexpFindAttrValue := regexp.MustCompile(`(?m)\s*(?P<key>\w+)\s*:\s*(?P<value>\w+);`)
	allIndexes := regexpFindAttrValue.FindAllSubmatchIndex(readScript, -1)
	if allIndexes == nil {
		myCSS.valido =false
		return
	}
	myCSS.ProVal = make(map[string]string, len(allIndexes))
	for _, r := range allIndexes {
		myCSS.ProVal[string(readScript[r[2]:r[3]])] = string(readScript[r[4]:r[5]])
	}
	myCSS.valido = true
	return
}

func parseScriptContentTag(readscript []byte, beginOfContentScript, endOfContentScript int, htmlElementMy *HTMLElement,mythreadsLaunch *int,chanToChildThreads chan int8) {

	funcIDX := bytes.Index(readscript[beginOfContentScript:endOfContentScript+1], []byte("function"))
	if funcIDX < 0 {
		return
	}
	functionWordSize := 8
	var firstOpenBrecet, offset int = 0, beginOfContentScript
	myRegExpression := regexp.MustCompile(`\w+|([\w*[,]*])`)
	myRegExpressionFunction := regexp.MustCompile(`function\s+\w+\s*\([\w*,]*\)\s*{`)
	htmlElementMy.MyJSFunc = make([]JavaScriptFunc, 0, 10)

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
		htmlElementMy.MyJSFunc = append(htmlElementMy.MyJSFunc, auxJS)
		mutexStyleScriptParseGO.Lock()
		if(threadDOWNcount <numberOfCores){
			if(chan2CssJavaBool == false){
				chan2CssJava = make(chan htmlTothreadsInside,10*(numberOfCores-1))
				chan2CssJavaBool = true
			}
			if(chan2ScriptStyleWork^chan2htmlWorkVerifiy ==0){
				go readStyleScriptObjectFunc(readscript,int8(threadDOWNcount),chanToChildThreads)
				*mythreadsLaunch++
				threadDOWNcount++
			}
			
		}
		mutexStyleScriptParseGO.Unlock()
		auxSendToThread := htmlTothreadsInside{tipo:1,beginStyle:firstOpenBrecet+1,endStyle: endOfContentScript+1,jsObjectPTR: &htmlElementMy.MyJSFunc[len(	htmlElementMy.MyJSFunc)-1]  }
		chan2CssJava <- auxSendToThread
		//go findEndOfJSfunction(&myJSfunction[len(myJSfunction)-1], readscript, ,, myChannArray[myChannIDXArray[quantidade]])

		offset = firstOpenBrecet + 1
		funcIDX = bytes.Index(readscript[offset:endOfContentScript+1], []byte("function"))
	}

	if len(htmlElementMy.MyJSFunc) == 0 {
		htmlElementMy.MyJSFunc = nil
		return
	}

	return
}

func findEndOfJSfunction(myJS *JavaScriptFunc, readScript []byte, offset int) {

	contagem := 1
	var aux,offSetINT int

	for ; contagem != 0 && aux <len(readScript); aux++ {
		aux := bytes.IndexAny(readScript[aux:], "{}")
		if aux <0{
			break
		}
		offSetINT +=aux
		offset += aux
		if readScript[offSetINT] == '{' {
			contagem++
		} else {
			contagem--
		}
	}

	if contagem != 0 {
		myJS.EndFunction = -1
		return
	}
	myJS.EndFunction = offset
	return
}

//RecoverAttrInTag function:
//Output: Copy of the html elements which tag name is equal to tag and a array of strings
//with the value of the attribute attr of those html elements
//Objective: Return an array of html elements copy which tag is equal to the input argument "tag"
func RecoverAttrInTag(tag, attr string) (array []HTMLElement, value []string) {
	myHTMLelementArray := make([]*HTMLElement, 0, 10)                                           //create an array of pointers which will point to the html elements we want
	myHTMLelementArray = recoverAttrInTagRec(tag, attr, myHTMLelementArray, myFirstHTMLElement) //find html elements with tag name equal to tag and with value assign to attribute attr
	value = make([]string, len(myHTMLelementArray))                                             //array of strings which will be saved the value of the attr attributes
	array = make([]HTMLElement, len(myHTMLelementArray))                                        //array of html elements which will be the destination of a copy of those html elements with tag name equal to tag and assigned value of attr attribute
	for i := 0; i < len(myHTMLelementArray); i++ {                                              //copy each html element found to the array and extract the value of attr attribute
		array[i] = *myHTMLelementArray[i]
		value[i] = myHTMLelementArray[i].AttrValue[attr]
	}
	return
}

//recoverAttrInTagRec function:
//Output: array of pointers to html elements with tag equal to tag and value assign to attr attribute
//Objective: find html elements with tag equal to tag and value assign to attr attribute recursively
func recoverAttrInTagRec(tag, attr string, htmlArray []*HTMLElement, Start *HTMLElement) []*HTMLElement {

	if Start == nil { //if we think the struct as a tree, if start HTMLElement pointer is nil, means that i don't have a tree, or that i reach the end of a branch
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

//cleanCSSJS function:
//Input: Pointer to the start of HTMLElement Tree
//Objective: Delete the invalid CSS Objects and JavaScript function elements.
func cleanCSSJS(Start *HTMLElement){
	if Start == nil { //if we think the struct as a tree, if start HTMLElement pointer is nil, means that i don't have a tree, or that i reach the end of a branch
		return 
	}
	for i := 0; i < len(Start.myHTMLElementDown); i++ { //go to all the potential leafs or new branchs that belong to the Start branch
		cleanCSSJS(Start.myHTMLElementDown[i])
	}

	if strings.Compare(Start.Element, "style") == 0 { //see if Start has a tag name equal to tag
		if Start.MyCSSObject != nil{
			cleanCSS(Start)
		}
	}else if(strings.Compare(Start.Element, "script") == 0){
		if Start.MyJSFunc != nil{
			cleanJS(Start)
		}
	}
	return 
}

//cleanCSS function:
//Input: Pointer to the HTMLElement which tag is equal to style
//Objective: Delete the invalid CSS Objects elements.
func cleanCSS(cleanCSSObj *HTMLElement){

	lixo:=len(cleanCSSObj.MyCSSObject)
	for i:=0;i<lixo;i++{
		if(cleanCSSObj.MyCSSObject[i].valido == false){
			copy(cleanCSSObj.MyCSSObject[i:lixo-1],cleanCSSObj.MyCSSObject[i+1:lixo])
			lixo--
		}
	}
	cleanCSSObj.MyCSSObject = cleanCSSObj.MyCSSObject[:lixo]
}

//cleanJS function:
//Input: Pointer to the HTMLElement which tag is equal to script
//Objective: Delete the invalid JS Objects elements.
func cleanJS(cleanJSObj *HTMLElement){

	lixo:=len(cleanJSObj.MyJSFunc)
	for i:=0;i<lixo;i++{
		if(cleanJSObj.MyJSFunc[i].EndFunction== -1){
			copy(cleanJSObj.MyJSFunc[i:lixo-1],cleanJSObj.MyJSFunc[i+1:lixo])
			lixo--
		}
	}
	cleanJSObj.MyJSFunc = cleanJSObj.MyJSFunc[:lixo]
}

//RecoverTag function:
//Input: string with the name of tag to recover
//Output: Array of HTMLElements which tag value is equal to "tag" input
//Objective: Find HTMLElements which tag is equal to "tag" input and return those as array
func RecoverTag(tag string) (array []HTMLElement) {
	myHTMLelementArray := make([]*HTMLElement, 0, 10)//create an array of pointers which will point to the html elements we want
	myHTMLelementArray = recoverTagRec(tag, myHTMLelementArray, myFirstHTMLElement) //find html elements with tag name equal to tag 
	array = make([]HTMLElement, len(myHTMLelementArray))//array of html elements which will be the destination of a copy of those html elements with tag name equal to tag 
	for i := 0; i < len(myHTMLelementArray); i++ { //copy each html element found to the array 
		array[i] = *myHTMLelementArray[i]
	}
	return
}

//recoverTagRec function:
//Output: array of HTMLElements pointers with "tag" input equal to tag 
//Objective: find HTMLElements with tag equal to "tag" input recursively
func recoverTagRec(tag string, htmlArray []*HTMLElement, Start *HTMLElement) []*HTMLElement {

	if Start == nil { //if we think the struct as a tree, if start HTMLElement pointer is nil, means that i don't have a tree, or that i reach the end of a branch
		return htmlArray
	}
	for i := 0; i < len(Start.myHTMLElementDown); i++ { //go to all the potential leafs or new branchs that belong to the Start branch
		htmlArray = recoverTagRec(tag, htmlArray, Start.myHTMLElementDown[i])
	}

	if strings.Compare(Start.Element, tag) == 0 { //see if Start has a tag name equal to tag
			htmlArray = append(htmlArray, Start)
	}
	return htmlArray
}

//RecoverAttr function:
//Output: Copy of the html elements which attr atribute has an assigned value and a array of strings
//with the value of the attribute attr of those html elements
//Objective: Return an array of html elements copy which contains a value for the attr attribute
func RecoverAttr(attr string) (array []HTMLElement, value []string) {
	myHTMLelementArray := make([]*HTMLElement, 0, 10)                                           //create an array of pointers which will point to the html elements we want
	myHTMLelementArray = recoverAttrRec(attr, myHTMLelementArray, myFirstHTMLElement) //find html elements with value assign to attribute attr
	value = make([]string, len(myHTMLelementArray))                                             //array of strings which will be saved the value of the attr attributes
	array = make([]HTMLElement, len(myHTMLelementArray))                                        //array of html elements which will be the destination of a copy of those html elements whith assigned value of attr attribute
	for i := 0; i < len(myHTMLelementArray); i++ {                                              //copy each html element found to the array and extract the value of attr attribute
		array[i] = *myHTMLelementArray[i]
		value[i] = myHTMLelementArray[i].AttrValue[attr]
	}
	return
}

//recoverAttrRec function:
//Output: array of pointers to html elements which contains an value assigned to attribute "attr"
//Objective: find html elements which contains an value assigned to attribute "attr" recursively
func recoverAttrRec(attr string, htmlArray []*HTMLElement, Start *HTMLElement) []*HTMLElement {

	if Start == nil { //if we think the struct as a tree, if start HTMLElement pointer is nil, means that i don't have a tree, or that i reach the end of a branch
		return htmlArray
	}
	for i := 0; i < len(Start.myHTMLElementDown); i++ { //go to all the potential leafs or new branchs that belong to the Start branch
		htmlArray = recoverAttrRec(attr, htmlArray, Start.myHTMLElementDown[i])
	}

	if _, ok := Start.AttrValue[attr]; ok == true { //see if the Start html element has a attribute call attr
			htmlArray = append(htmlArray, Start)
	}
	return htmlArray
}
