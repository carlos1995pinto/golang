package csv

import (
	"bytes"
	"io"
	"os"
	//"fmt"
)

var fid *os.File
var size1read int = 10
var fimRead uint64 = 0xFFFFFFFF00000000
func Initialize(fidIN *os.File) {
	fid = fidIN
}

func Read1Line() ([]string,bool) {
	out,err := read1Line()
	if(err == false){
		return nil,false
	}
	mapCommaQuote := countSpecialChars(out)
	outString := make([]string,len(mapCommaQuote))
	var inicioS,fimS uint32
	var lenV int
	var lenS uint32 
	var compensation uint32
	//var lenV int
	for k,v := range mapCommaQuote{
		lenV = len(v)
		inicioS = uint32((fimRead >>32) & k)
		fimS = uint32((fimRead &k)>>32)
		s:= out[inicioS:fimS]
		lenS =fimS-inicioS
		if(lenV!=1){
			s=s[v[0]+1:v[lenV-2]]
			compensation = 1
		}

		for i:=1;i<lenV-2;i+=2{ //agora só existem pares 
			copy(s[v[i]-compensation: lenS-compensation-2],s[v[i+1]-compensation:])
			s = s[:lenS-compensation-2]
			compensation++
		}
		outString[v[lenV-1]]=string(s)
		//fmt.Printf("%s\n",outString[v[lenV-1]])
	}
	//var StringCount,inicio,fim int
	return outString,true
}

func read1Line() ([]byte, bool) {
	if fid == nil {
		return nil, false
	}
	origPos, _ := fid.Seek(0, 1)
	conteudo := make([]byte, size1read, 2*size1read)
	n, err := fid.Read(conteudo)
	if err != nil && n == 0{        
		return nil, false
	}
	var idx, nPrev, contagem int = 0, n, 0
	aux := make([]byte, size1read)
	for idx = bytes.IndexAny(conteudo[contagem:n+contagem], "\r\n"); idx < 0; idx = bytes.IndexAny(conteudo[contagem:n+contagem], "\r\n") {
		if err == io.EOF {
			break
		}
		n, err = fid.Read(aux)
		if err != nil && err != io.EOF {
			return nil, false
		}
		conteudo = append(conteudo, aux[:n]...)
		contagem += nPrev
		nPrev = n

	}
	conteudo[contagem+idx] = 59//inserir ; no conteudo para que possa ir ate ao fim
	conteudo = conteudo[:contagem+idx+1]
	if err == io.EOF {
		return conteudo, true
	}
	
	fid.Seek(int64(len(conteudo)-1)+2+origPos, 0)
	return conteudo, true

}

func countIDXSpecialChar(in []byte,charIN string,offset uint32) ([]uint32,int){
	outIDX := make([]uint32,len(in)+1)
	idx := bytes.IndexAny(in,charIN)
	if(idx < 0){
		return nil,0
	}
	var outCount,countIDX int = 0,idx
	outIDX[outCount] = uint32(countIDX)+offset
	for idx =bytes.IndexAny(in[countIDX+1:],charIN) ;idx >=0;idx = bytes.IndexAny(in[countIDX+1:],charIN){
		outCount++
		countIDX +=idx+1
		outIDX[outCount] = uint32(countIDX)+offset
	}
	return outIDX[:outCount+1],outCount+1
}

func countSpecialChars(in []byte) (map[uint64][]uint32){	
	mapCommaQuotes := make(map[uint64][]uint32,len(in)+1)
	idxComma := bytes.IndexAny(in,";")
	var countIDX int =idxComma
	idxQuote,countQuoteInter := countIDXSpecialChar(in[0:countIDX],"\"",0)
	var inicio uint64
	var auxidxQuote []uint32 = make([]uint32,0,100)
	var countString uint32
	auxidxQuote = append(auxidxQuote,idxQuote...)
	if(countQuoteInter & 1 ==0){
		auxidxQuote = append(auxidxQuote,countString)
		mapCommaQuotes[uint64(countIDX) <<32 | inicio] = make([]uint32,len(auxidxQuote))	//Guardar o fim nos 32 bits mais significativos, guardar o inicio nos 32 bits menos significativos
		copy(mapCommaQuotes[uint64(countIDX) <<32 | inicio],auxidxQuote)
		
		inicio = uint64(countIDX)+1 // Caso o idxComma simbolize o fim de um campo CSV
		auxidxQuote = auxidxQuote[:0]
		countQuoteInter = 0
		countString++
	}

	inicioPrev := countIDX+1
	//idxComma marca o fim de um campo do CSV e o inicio de outro campo CSV
	
	for idxComma = bytes.IndexAny(in[countIDX+1:],";");idxComma >=0;idxComma=bytes.IndexAny(in[countIDX+1:],";"){
		countIDX +=idxComma+1
		idxQuote,CountQuote := countIDXSpecialChar(in[inicioPrev:countIDX],"\"",uint32(uint64(inicioPrev)-inicio))
		inicioPrev = countIDX+1
		countQuoteInter +=CountQuote
		auxidxQuote = append(auxidxQuote,idxQuote...)
		if(countQuoteInter & 1 !=0){ //Por alguma razao ainda nao tenho aqui as " todas.Lembrar que se houver " num campo, estes tem que estar em numero par
			continue
		}
		
		auxidxQuote = append(auxidxQuote,countString)
		mapCommaQuotes[uint64(countIDX) <<32 | inicio] = make([]uint32,len(auxidxQuote))
		copy(mapCommaQuotes[uint64(countIDX) <<32 | inicio], auxidxQuote)
		countString++
		auxidxQuote = auxidxQuote[:0]
		inicio = uint64(countIDX)+1 // Caso o idxComma simbolize o fim de um campo CSV
		countQuoteInter = 0
	}

	return mapCommaQuotes
}