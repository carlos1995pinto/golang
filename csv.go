package csv

import (
	"bytes"
	"io"
	"os"
	//"fmt"
)

var fid *os.File //it will save the file descriptor of the csv file
var size1read int = 10 //it will read 10 bytes of the csv file each time
var fimRead uint64 = 0xFFFFFFFF00000000 //decode of the begin and end of each field in the all line
var outString []string //save the fields
var	currentIDX int = -1 //know which field has been return to the user

//Func :Initialize()
//Input: file descriptor from the csv file and number of bytes that will be read from the csv file each readI/O call
//Output: none
//Objective: save the file descriptor from the file where the function will read and 
//reset the buffer where the fields will be saved
func Initialize(fidIN *os.File,readPerCall int) {

	if(readPerCall != 0){
		size1read = readPerCall
	}
	fid = fidIN //save file descriptor
	outString = nil //reset the buffer where the fields value will be saved
}

//Func Read1Field()
//Input: none
//Output: the field value and a bool
//Objective: It will return to the user the next  field value. If there is no more field to read from the current 
//line or a line was never read, then this function will call Read1Line() in order to fill the buffer with new values.
//If the call to Read1Line() fails (returns a false bool value), than this function returns a empty string and a false bool value.
//Otherwise, it returns a string with the field value and a true bool value
func Read1Field() (string,bool){

	currentIDX++ //increase index to get next field value
	if(outString == nil || currentIDX==len(outString)){ //already output all field values from the current line or i have never read a line from the csv file
		_,ind := Read1Line() //read a new line
		currentIDX = 0 //reset the index 
		if(ind==false){ //something went wrong in the Read1Line() func
			return "",false
		}
		
	}

	return outString[currentIDX],true //return the field value
}

//Func: Read1Line()
//Input: none
//Output: pointer to a slice of string and a bool
//Objective: This function will read from the csv file  1 line and will save each field in a slice string. 
//Each call of this function will read a new line from the csv file. If the progammer calls the Read1Field function and 
//then calls the Read1Line, the function will return a new line if she exist and will not return the rest of the line that corresponds
//to the output given by Read1Field.
//	In case of success this function returns a pointer slice of string that contains each field and a true bool value. 
//In case of failure, this function returns nil slice of string and a false bool value
func Read1Line() ([]string,bool) {
	out,err := read1Line() //read a new line from the csv file
	if(err == false){ //fail to read a new line. Could be end of file reach or an error read
		return nil,false
	}
	mapCommaQuote := countSpecialChars(out) //get a map where each key indicates the begin and end of an field value and the values indicate where the char " is present in that field value
	if(outString == nil){ //th space to save the filed values was never alocate or is a new file that we are dealing
		outString = make([]string,len(mapCommaQuote)) //allocates the space need to save all the values in the line. This only needs to be done each new file. 
	}
	var inicioS,fimS uint32 //begin and end of field value  in the line
	var lenV int //len of the vetor with the indexs for the " char in the field value
	var lenS uint32 //len of the field value
	var compensation uint32 //compensation factor
	//var lenV int
	for k,v := range mapCommaQuote{ 
		lenV = len(v)
		inicioS = uint32((fimRead >>32) & k) //decode the begin of the field value in the line
		fimS = uint32((fimRead &k)>>32) // decode the end of the field value in the line
		s:= out[inicioS:fimS] //get the field value
		lenS =fimS-inicioS //get the len of the field value
		if(lenV!=1){ //the last value in the vetor V is the order of the field value in the line so all vector values from the map have at least one value
			s=s[v[0]+1:v[lenV-2]] //if it has more, than it means the field value has the " character. So we remove the first and the last which happens everytime that a field value has " and ; characters			compensation = 1
		}

		for i:=1;i<lenV-2;i+=2{ //eliminate the others " characters. Remind that each " insert by the user in the csv file, it will be saved in the csv file with the "" characters
			copy(s[v[i]-compensation: lenS-compensation-2],s[v[i+1]-compensation:]) //remove one of the " character from the pair
			s = s[:lenS-compensation-2] //erases the extra " character
			compensation++ //increase the compensation factor because the len of the field value has changed
		}
		outString[v[lenV-1]]=string(s) //convert the field value that is in bytes to a string and save him in the right order
		
	}
	currentIDX = len(outString)
	return outString,true
}

//Func: read1Line()
//Input: none
//Output: byte slice and a bool value
//Objective: read a new line from the csv file. In sucess the function returns the slice of bytes read that
//corresponds to a new line and a true bool value. In case of failure, the function returns a nil slice and a false bool value
func read1Line() ([]byte, bool) {
	if fid == nil {
		return nil, false
	}
	origPos, _ := fid.Seek(0, 1) //gets the current position of the csv read pointer
	conteudo := make([]byte, size1read, 2*size1read) //allocates size1read bytes but with twice the capacity. 
	n, err := fid.Read(conteudo) //read size1read bytes from the file
	if err != nil && n == 0{         //didn't have nothing to read or a read error has occur
		return nil, false
	}
	var idx, nPrev, contagem int = 0, n, 0
	aux := make([]byte, size1read) //auxiliar buffer
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