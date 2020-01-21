package main

import (
	"database/sql"
	"flag"
	"encoding/json"
	"fmt"
	"os"
	"log"
	"net/http"
	"gopkg.in/yaml.v2"
	_ "github.com/mattn/go-sqlite3"
)


var fileYAML  = flag.String("y","default.yml","must be a string!")
var fileJSON = flag.String("j","default.json","must be a string!")
func buildHandleByMap(myMap map[string]string) {

	for k, v := range myMap {
		var k1, v1 string = k, v
		http.HandleFunc(k1, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, v1, 301)
		}) // each request calls handler
	}
}
func main() {

	path2route := map[string]string{
		"/dog":         "https://dentastix.pedigree.pt/",
		"/cat":         "https://www.whiskas.pt/os-nossos-produtos/gatinhos",
		"/raspberryPI": "https://www.raspberrypi.org/",
	}
	path2routeYAML := map[string]string{}
	path2routeJSON := map[string]string{}
	path2routeDB := map[string]string{}
	readerYaml,err := os.Open(*fileYAML)
	if(err != nil){
		fmt.Println(err)
	}
	readerJson,err := os.Open(*fileJSON)
	if(err != nil){
		fmt.Println(err)
	}
	decoderYAML := yaml.NewDecoder(readerYaml)
	decoderJSON := json.NewDecoder(readerJson)
	err = decoderJSON.Decode(&path2routeJSON)
	err = decoderYAML.Decode(&path2routeYAML)
	readerYaml.Close()
	readerJson.Close()
	mydatabase,err:=sql.Open("sqlite3","mydb.db")
	if(err != nil){
		fmt.Println(err)
	}
	rows,_ := mydatabase.Query("Select * from route2new")
	var url,route string
	for rows.Next(){
		rows.Scan(&url,&route)
		path2routeDB[url] = route
	}
	mydatabase.Close()
	buildHandleByMap(path2route)
	buildHandleByMap(path2routeYAML)
	buildHandleByMap(path2routeJSON)
	buildHandleByMap(path2routeDB)
	log.Fatal(http.ListenAndServe("localhost:8000", nil))



}
