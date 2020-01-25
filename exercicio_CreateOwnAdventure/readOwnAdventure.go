package readOwnAdventure

import(
	"os"
	"encoding/json"
	"fmt"
	"strings"
	"html/template"
	"net/http"
)

type htmlTemplateData struct{
	Comment string
	Title string
	Text []string
	Image []string
	Link []string
} //this struct will save everything that will be need to create the webpage for certain chappter

var allContent map[string]htmlTemplateData //association of each chaptter with the webpage
var myHtmlTemplate *template.Template //html webpage template 

//init func
//Input:none
//Output:none
//Objective: Allocate memory for the map which associates each chapter with an webpage and also load 
//the html webpage template
func init(){
	allContent = make(map[string]htmlTemplateData,50) 
	myHtmlTemplate = template.Must(template.ParseFiles("templatePage.html"))
}

//ReadConfigFile func
//Input: String which is the path of the config file
//Output: Map that will give the association of each chapter/web path with the json file that contains 
//the webpage content(title of chapter,text,images and buttons to redirect to another chappers) and a 
//bool that indicates the success of function
//Objective: Read config file that has for each path/chapter the path of json file that contains the web page info/content of web diretory
func ReadConfigFile(file string) (map[string]string,bool){
	if(!strings.HasSuffix(file,".json")){ //find if config file is json type

		return nil,false
	}
	fp,err := os.Open(file)//open config file
	if(err !=nil){
		return nil,false
	}
	defer fp.Close()
	defaultDeco := json.NewDecoder(fp) //create a new json decoder for that file
	myMap := map[string]string{} //create the map that this function will output
	defaultDeco.Decode(&myMap) //read json file and put its content on map
	delete(myMap,"comment") //delete the comment key because it is not need for program
	if _,ok :=myMap["startPoint"];!ok{ //check if it is a config file. A config file must have a key named "startPoint" that indicates the start web diretory
		return nil,false
	}
	buildMainRoute(myMap["startPoint"]) //build the handle func for the path "/". This function will redirect for the path that is the value of the key "startPoint"
	delete(myMap,"startPoint") //delete because it will not be need anymore
	return myMap,true
}
//buildMainRoute func
//Input: String that indicates the path that which user will be redirect from the "/"path
//Output: None
//Objective: build a handle func that redirects user from the "/" path to the startPoint path 
func buildMainRoute(path string){
	http.HandleFunc("/",func (w http.ResponseWriter, r* http.Request){
		http.Redirect(w,r,"/"+path,301)
		return
	})
}
//BuildRoute func
//Input: String that indicates the web path and another string that indicates the json file associated with that path
//Output: None
//Objective: build the http handle func for each chapter/path.
func BuildRoute(path string,file string){

	myPathData,ok := allContent[path] //get the htmlTemplateData that is associate with the path. This struct contains all that is need to build the html page for this chapter
	if(!ok){ //don't have the struct build for this path
		if(!readChapper(path,file)){ //build the struct
			return
		}
		myPathData=allContent[path] //get the struct htmlTemplateData for this path
	}
	http.HandleFunc("/"+path,func(w http.ResponseWriter, r* http.Request){
	if err:= myHtmlTemplate.Execute(w,myPathData);err!=nil{ //build the webpage by using the struct htmlTemplateData and the html template and send as a response to the get
		fmt.Printf("Erro no path /%s:\n%v\n",path,err)
	}
	return
	})

}
//readChapper func
//Input: A string that is the web path of the chapter and a string that has the path of the json file that contains information needed to build the chapter webpage
//Output: Bool value that indicates the sucess of the function
//Objective: read the json file and load its content to a htmlTemplateData struct
func readChapper(link string,file string) (bool){
	fp,err := os.Open(file) //open file
	if(err != nil){
		return false
	}
	defer fp.Close()
	FileDeco := json.NewDecoder(fp) //create a new decoder
	var myChapperMap htmlTemplateData
	err = FileDeco.Decode(&myChapperMap) //fil the htmlTemplateData struct with the json content
	if(err != nil){
		return false
	}

	allContent[link] =myChapperMap //save that struct on the map associated to the web path
	return true
}

