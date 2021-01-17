package main

import (
	"encoding/gob"
	"os"
	"strconv"
	"strings"
	"fmt"
	"math/rand"
	"time"
	"net"
	"sync"
)


type NodeID struct {
	Endpoint string
	nodeSerialNum int
}

type Info struct {
	UniqueNum int
	Name   string
	Method string
	Type string
	Index  string
	Endpoint string
	nodeSerialNum int
}

var portNum string
var lst [3]int
var alive map[string]bool 
var leader bool
var leaderPort string
var log map[int]Info
var check map[int]int
var uniNum int
var quorum map[int]int
var id NodeID
var queue []chan Info
var shoppingList []string = []string{"Tomatoes", "Eggs","Milk"}


func main() {
	alive = make(map[string]bool)
	parseArg(os.Args[1], os.Args[2])
	parseArg(os.Args[3], os.Args[4])

	checkLeaderInit()
	
	log = make(map[int]Info)
	check = make(map[int]int)
	quorum = make(map[int]int)
	quorum[0] = 2

	uniNum = rand.Int()

	ln, err := net.Listen("tcp", string(":"+fmt.Sprint(portNum)))
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("port : " + fmt.Sprint(portNum))

	num,_ := strconv.Atoi(portNum)
	nodeNum := num - 8090
	id = NodeID(NodeID{Endpoint : portNum, nodeSerialNum : nodeNum})
	fmt.Println("my serial number is :", id.nodeSerialNum)

	queue = []chan Info{}

	for i:= 0; i < 4; i++ {
		subQueue := make(chan Info, 1000)
		queue = append(queue, subQueue)
		go Acceptor(subQueue)
	}

	heartbeat()

	//accepting messages
	for{
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		defer conn.Close()
		var message Info
		decoder := gob.NewDecoder(conn)
		e := decoder.Decode(&message)
		if e != nil {
			fmt.Println(e)
			return
		}
		
		//if message is from the frontend 
		if (message.UniqueNum == -1 ){
			go func() {
				encoder := gob.NewEncoder(conn)
				time.Sleep(3 * time.Second)
				error := encoder.Encode(shoppingList)
				fmt.Println(shoppingList)
				if error != nil {
					fmt.Println(error)
					return
				}
				fmt.Println(shoppingList)
			}()

			//acknowledge that itself is the proposer
			if (message.Method == "LEADER"){
				if leader == false {
					leader = true
					fmt.Println("I am now the leader!")
					leaderPort = id.Endpoint
				}
			}

			//if asking for INDEX, no need to process, just return the data
			if !(message.Method == "INDEX" || message.Method == "CHECK" || message.Method == "ALIVE" || message.Method == "LEADER") {
				message.nodeSerialNum = 3
				queue[3] <- message
			}
			
		}else {
			if message.Type == "PING" {
				leaderPort = message.Endpoint
				leader = false
			}
			num,_ := strconv.Atoi(message.Endpoint)
			nodeNum := num - 8090
			message.nodeSerialNum = nodeNum
			queue[message.nodeSerialNum] <-message
		}
		
	}
}

func heartbeat() {
	//ping its followers every 2 seconds
	go func(){
		for {
			if leader == true{
				for i:= 0; i < 3; i++ {
					prt := strconv.Itoa(lst[i])
					if (prt != id.Endpoint) {
						time.Sleep(1 * time.Second)
						betweenServer ("", "", "", "PING", 0,prt, id.Endpoint, id.nodeSerialNum)
					}
				}
			}
		}
	}()
}

func Acceptor(q chan Info) {
		for {
			message := <-q
			fmt.Println("Received : "+message.Type + " messege Method " + message.Method +" To " + portNum + " from ", message.Endpoint, "with serial number ",message.UniqueNum, "from :", message.nodeSerialNum)
			//leader
			if leader == true {
				toFollowers(message)
			}else {
				reply(message)
			}
		}

}

func toFollowers(message Info) {
	switch message.Type {
	case "OK":
		//if heard back for quorum, send accept message
		check[message.UniqueNum] += 1
		if (check[message.UniqueNum] > quorum[0] || quorum[0] == 0){
			go func() {
				for i:=0; i < 3; i++{
					prt := strconv.Itoa(lst[i])
					if (prt != id.Endpoint && alive[prt] == true) {
						betweenServer(message.Method, message.Index, message.Name, "ACCEPT", message.UniqueNum,prt,id.Endpoint,id.nodeSerialNum)
					}else {
						log[message.UniqueNum] = message
					}
				}
				log[message.UniqueNum] = message
				execute(message)
			}()
		}

	case "PREPARE":
		check[message.UniqueNum] += 1

	//front-end request, send prepare message to all nodes including itself
	case "":
		//send as a go routine to avoid deadlock on itself
		check[uniNum] = 0
		go func() {
			for i:=0; i < 3; i++{
				prt := strconv.Itoa(lst[i])
				if alive[prt] == true {
					betweenServer(message.Method, message.Index, message.Name,"PREPARE", uniNum,prt,id.Endpoint,id.nodeSerialNum)
				}
			}
			var mtx sync.Mutex
			mtx.Lock()
			uniNum += 1
			mtx.Unlock()
		}()
	}
}

func reply(message Info) {
	switch message.Type {
	//check if the same value had already been claimed, if not, send back OK
	case "PREPARE":
		fmt.Println("Unique number of this request : ", message.UniqueNum, "Received by Serial Number :", id.nodeSerialNum)
		_, ok := log[message.UniqueNum]
		if  !ok {
			betweenServer(message.Method, message.Index, message.Name, "OK", message.UniqueNum, leaderPort, id.Endpoint,id.nodeSerialNum)
		}
	//get accept message, call execute to update log
	case "ACCEPT":
		log[message.UniqueNum] = message
		execute(message)
	case "PING":
	}
}

func checkLeaderInit() {
	minimum := lst[0]
	for i:= 1; i < 3; i++ {
		prtNum := lst[i]
		prt := strconv.Itoa(prtNum)
		if alive[prt] == true {
			if (prtNum < minimum) {
				minimum = prtNum
			}
		}
	}
	prt,_ := strconv.Atoi(portNum)
	if minimum == prt {
		leader = true
		fmt.Println("I am leader")
	}
	leaderPort = strconv.Itoa(minimum)
}

func parseArg(name string, str string) {
	switch name {
	case "--listen":
		portNum = str
		lst[0],_ = strconv.Atoi(portNum)
		alive[str] = true
		
	case "--backend":
		backend := strings.Split(strings.Replace(str, ":", "", -1), ",")
		for i:= 0; i < len(backend);i++ {
			ptr, _ := strconv.Atoi(backend[i])
			lst[i + 1] = ptr
			alive[backend[i]] = true
		}
	}
}

func execute(message Info) {
	switch message.Method {
	case "CREATE":
		go func (){
			var mtx sync.Mutex
			mtx.Lock()
			shoppingList = append(shoppingList, message.Name)
			mtx.Unlock()
		}()
					
	case "UPDATE":
		go func(){
			idx,_ := strconv.Atoi(message.Index)
			var mtx sync.Mutex
			mtx.Lock()
			shoppingList[idx] = message.Name
			mtx.Unlock()
		}()
					
	case "DELETE":
		go func(){
			fmt.Println("DELETING!!!!")
			idx,_ := strconv.Atoi(message.Index)
			var mtx sync.Mutex
			mtx.Lock()
			shoppingList = append(shoppingList[:idx], shoppingList[idx+1:]...)
			mtx.Unlock()
		}()
	}
}

func betweenServer (req string, idx string, name string, typ string, uni int,target string, edprt string, nodeSerialNum int) {
	conn, err := net.Dial("tcp", "localhost:"+ target)
	if err != nil {
		fmt.Println("Detected failure on " + target + " at " + time.Now().Format(time.RFC850))
		if alive[target] != false{
			alive[target] = false
			quorum[0] -= 1
		}
		fmt.Println("Quorum is ", quorum[0])
		return
	}
	defer conn.Close()

	if alive[target] == false {
		alive[target] = true
		quorum[0] += 1
		for _,msg := range log {
			betweenServer(msg.Method,msg.Index,msg.Name,"ACCEPT",msg.UniqueNum,target,id.Endpoint,id.nodeSerialNum)
		}
		fmt.Println(target, " is BACK!! quorum now is ",quorum[0])
	}
	fmt.Println("Sending Type: "+typ + " To " + target + " Received from ", edprt, "with serial number ",uni, "from node :", nodeSerialNum)
	message := Info(Info{ UniqueNum: uni, Name: name, Type: typ ,Method: req, Index: idx, Endpoint : edprt, nodeSerialNum : nodeSerialNum})

	
	//encode message
	encoder := gob.NewEncoder(conn)
	err = encoder.Encode(message)
	if err != nil {
		panic(err)
	}

}
