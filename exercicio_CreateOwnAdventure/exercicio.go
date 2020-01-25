package main

import (
	"flag"
	"net/http"
	"readOwnAdventure"
	"fmt"
)

var defaultFile = flag.String("i", "default.json", "Must be a string")
func main() {
	myMap, _ := readOwnAdventure.ReadConfigFile(*defaultFile)
	for k,v:=range myMap{
		readOwnAdventure.BuildRoute(k,v)
	}
	err := http.ListenAndServe("localhost:8000",nil)
	fmt.Println(err)

}
