package main

import (
	"bufio"
	"time"
	"math/rand"
	"csv"
	"flag"
	"fmt"
	"os"
	"strings"
)

var filename = flag.String("f", "problems.csv", "must be a csv file")
var timer = flag.Int("t",30,"must be an int")
var random = flag.Bool("r",false,"must be true or false")
var quizTimer *time.Timer
type questao struct{
	questao string
	respostaCorreta string
	respostaUtilizador string
	questionada bool 
	resultado bool
}

func makeQuiz(initiate chan struct{},quiz []questao){

	stdinRead := bufio.NewReader(os.Stdin)
	
	fmt.Print("Press \"Enter\" to Start the quiz\n")
	_,_=stdinRead.ReadBytes('\n')
	initiate <-struct{}{}
	var erro error
	if(*random == false){
		for i:=0;i<len(quiz);i++{
			v := &quiz[i]
			fmt.Printf("Q%dº:%s\nR:",i+1,v.questao)
			v.respostaUtilizador,erro = stdinRead.ReadString('\n')
			if(erro != nil){
				break
			}
			v.respostaUtilizador = strings.TrimSpace(strings.ToLower(v.respostaUtilizador))
			if(v.respostaUtilizador == v.respostaCorreta){
				v.resultado = true
			}
		}
	}else{
		for i:=0;i<len(quiz);i++{
			idx:= rand.Intn(len(quiz)-1)
			if(quiz[idx].questionada ==true){
				i--;
				continue
			}
			v := &quiz[idx]
			fmt.Printf("Q%dº:%s\n",i+1,(*v).questao)
			(*v).respostaUtilizador,erro = stdinRead.ReadString('\n')
			if(erro != nil){
				break
			}
			(*v).questionada = true
			(*v).respostaUtilizador = strings.TrimSpace(strings.ToLower((*v).respostaUtilizador))
			if((*v).respostaUtilizador == (*v).respostaCorreta){
				(*v).resultado = true
			}
		}

	}
	quizTimer.Stop()
	
}

func main() {

	if !strings.HasSuffix(*filename, ".csv") {
		fmt.Fprint(os.Stderr, "File must be a csv extension\n")
		return
	}
	fp, err := os.Open(*filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not open %s\n", *filename)
		return
	}
	defer fp.Close()
	csv.Initialize(fp, 100)
	var questoesQuiz []questao = make([]questao,100,100)
	var contadorQuestoes int
	for contadorQuestoes = 0;contadorQuestoes <100;contadorQuestoes++  {

		out,errorInfo :=csv.Read1Line()
		if(errorInfo == false){
			break
		}
		questoesQuiz[contadorQuestoes] = questao{out[0],out[1],"",false,false}
	}
	questoesQuiz = questoesQuiz[:contadorQuestoes]
	canalComm := make(chan struct{})
	go makeQuiz(canalComm,questoesQuiz)
	<-canalComm
	quizTimer = time.NewTimer(time.Second*time.Duration(*timer))
	<-quizTimer.C
	var countCertas =0
	fmt.Print("Resultado do Quiz:\n")
	for i,v:= range questoesQuiz{
		fmt.Printf("Q%dº:%v\n",i,v.resultado)
		if(v.resultado==true){
			countCertas++
		}
	}
	fmt.Printf("\nRespostas Corretas:%d\tRespostas Incorretas:%d\n",countCertas,contadorQuestoes-countCertas)

}
