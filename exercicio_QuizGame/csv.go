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

//Initialize func
//Input: file descriptor from the csv file and number of bytes that will be read from the csv file each readI/O call
//Output: none
//Objective: save the file descriptor from the file where the function will read and 
//reset the buffer where the fields will be saved.
func Initialize(fidIN *os.File,readPerCall int) {

	if(readPerCall != 0){
		size1read = readPerCall
	}
	fid = fidIN //save file descriptor
	outString = nil //reset the buffer where the fields value will be saved
}
//GetCurrentLine func
//Input: None
//Output: Slice of string and a bool value 
//Objective: returns the field values for the last read line of the csv file; if no line was ever read from the csv file, 
//this function returns nil slice and a false bool value. Otherwise, it returs the slice that contains the field 
//value and a true bool value
func GetCurrentLine()([]string,bool){
	if(outString==nil){
		return nil,false
	}
	return outString,true
}

//Read1Field func
//Input: none
//Output: the field value and a bool
//Objective: It will return to the user the next field value; If there is no more field to read from the current 
//line or a line was never read, then this function will call Read1Line() in order to fill the buffer with new values;
//If the call to Read1Line() fails (returns a false bool value), than this function returns a empty string and a false bool value;
//Otherwise, it returns a string with the field value and a true bool value.
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

//Read1Line func
//Input: none
//Output: pointer to a slice of string and a bool
//Objective: This function will read from the csv file  1 line and will save each field in a slice string;
//Each call of this function will read a new line from the csv file. If the progammer calls the Read1Field function and 
//then calls the Read1Line, the function will return a new line if she exist and will not return the rest of the line that corresponds
//to the output given by Read1Field;
//In case of success this function returns a pointer slice of string that contains each field and a true bool value;
//In case of failure, this function returns nil slice of string and a false bool value.
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

//read1Line func
//Input: none
//Output: byte slice and a bool value
//Objective: read a new line from the csv file; In sucess the function returns the slice of bytes read that
//corresponds to a new line and a true bool value; In case of failure, the function returns a nil slice and a false bool value.
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
	for idx = bytes.IndexAny(conteudo[contagem:n+contagem], "\r\n"); idx < 0; idx = bytes.IndexAny(conteudo[contagem:n+contagem], "\r\n") { //for cicle that will search for the end of line flag = \r\n
		if err == io.EOF { //Reach end of file
			break
		}
		n, err = fid.Read(aux)//get more size1read bytes from the csv file
		if err != nil && err != io.EOF { //something went wrong in the read of the file
			return nil, false
		}
		conteudo = append(conteudo, aux[:n]...) //append the last size1read bytes read from the csv file to the contuedo
		contagem += nPrev//only need to search this last bytes read 
		nPrev = n

	}
	conteudo[contagem+idx] = 59//insert the char ; in the end of the line so that later it could be more easy to extract the last field value
	conteudo = conteudo[:contagem+idx+1] //only want until the end of the line. i could have read bytes from the next line which we are not our interest now
	if err == io.EOF { //reach end of file, just return
		return conteudo, true
	}
	
	fid.Seek(int64(len(conteudo)-1)+2+origPos, 0)//in case i read bytes from the next line, i need to put the file pointer back to end of the line so that in next call, this function will be capable to extracl the entire next line
	return conteudo, true

}

//countIDXSpecialChar func
//Input: slyce of bytes, string and uint32
//Output: slice of uint32 and int value
//Objective: find in the slice byte input the char charIN and save the index of that char in the slice byte input
//the indexs are saved in a slice with an offset given by the offset input variable; In case o sucess, 
//the function returns the slice with index and the quantity of those charIN caracters present in the slice input.
func countIDXSpecialChar(in []byte,charIN string,offset uint32) ([]uint32,int){
	outIDX := make([]uint32,len(in)+1) //at max i can only have len(in) of charIN present in the slice byte input
	idx := bytes.IndexAny(in,charIN) //find the first charIN
	if(idx < 0){//No charIN was found
		return nil,0
	}
	var outCount,countIDX int = 0,idx
	outIDX[outCount] = uint32(countIDX)+offset//save the index+offset of the first charIN found 
	for idx =bytes.IndexAny(in[countIDX+1:],charIN) ;idx >=0;idx = bytes.IndexAny(in[countIDX+1:],charIN){ //find the rest of charIN
		outCount++
		countIDX +=idx+1//marks the index where the next search will be done
		outIDX[outCount] = uint32(countIDX)+offset
	}
	return outIDX[:outCount+1],outCount+1 //cut the vector with index to have the length equal to the number of charIN present in the slice byte input
}

//countSpecialChars func
//Input: slice byte
//Output: map with keys of uint64 and values as slice of uint32
//Objective: find the begin and the end index of each field value in the slice byte input 
//and also in each field value find the index of the " caracter.
func countSpecialChars(in []byte) (map[uint64][]uint32){	
	mapCommaQuotes := make(map[uint64][]uint32,len(in)+1) //the map will have at max len(in) keys because that is the case when all field values are empty 
	idxComma := bytes.IndexAny(in,";") //find the first ; character
	var countIDX int =idxComma
	idxQuote,countQuoteInter := countIDXSpecialChar(in[0:countIDX],"\"",0) //find the index and the number of " caracteres present since the begin of in until the first ; character
	var inicio uint64
	var auxidxQuote []uint32 = make([]uint32,0,100) //allocate the buffer that will save temporally the index of " characters in the filed value 
	var countString uint32 //it has the order of the field value
	auxidxQuote = append(auxidxQuote,idxQuote...) //save the indexs of " characters found in the supposedly field value
	if(countQuoteInter & 1 ==0){ //if exist the " character in a field value, the number present bust be even
		auxidxQuote = append(auxidxQuote,countString) //save the order of the field value in the input slice byte
		mapCommaQuotes[uint64(countIDX) <<32 | inicio] = make([]uint32,len(auxidxQuote))	//Save the begin index in the 32 least significant bits and the end index in the 32 most significant bits(this is the key of the map)
		//allocate also the memory to save the slice which has the index of the " caracters in the field value and also its order
		copy(mapCommaQuotes[uint64(countIDX) <<32 | inicio],auxidxQuote)//copy of the index of the " characters and the field value order
		
		inicio = uint64(countIDX)+1 // index where the next field value will start
		auxidxQuote = auxidxQuote[:0] //clear the slice 
		countQuoteInter = 0 //clear the counter of the " character present in the field value
		countString++
	}

	inicioPrev := countIDX+1 // index that indicates where i will start the search for the " character
	
	for idxComma = bytes.IndexAny(in[countIDX+1:],";");idxComma >=0;idxComma=bytes.IndexAny(in[countIDX+1:],";"){
		countIDX +=idxComma+1 //upload the index value where the next search for the ; character will start
		idxQuote,CountQuote := countIDXSpecialChar(in[inicioPrev:countIDX],"\"",uint32(uint64(inicioPrev)-inicio))//count the number of " char present in the slice. the offset argument will be 
		//zero in case we are start in a new field value. Otherwise it will have a number that with the sum of the index of the 
		//" char found in the input slice byte of countIDXSpecialChar function it will give the truth position of the " 
		//character in the field value
		inicioPrev = countIDX+1//update the begin index of the next search for the " character
		countQuoteInter +=CountQuote
		auxidxQuote = append(auxidxQuote,idxQuote...)
		if(countQuoteInter & 1 !=0){ //didn't found an even number of " character in the slice so that indicates that i have not yet found the end index of the field value
			continue
		}
		
		auxidxQuote = append(auxidxQuote,countString)
		mapCommaQuotes[uint64(countIDX) <<32 | inicio] = make([]uint32,len(auxidxQuote))
		copy(mapCommaQuotes[uint64(countIDX) <<32 | inicio], auxidxQuote)
		countString++
		auxidxQuote = auxidxQuote[:0]
		inicio = uint64(countIDX)+1 
		countQuoteInter = 0
	}

	return mapCommaQuotes
}