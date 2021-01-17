package main

import (
	"encoding/gob"
	"fmt"
	"net"
	"time"
	"strings"
	"os"
	"strconv"

	"github.com/kataras/iris/v12"
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
var backend string = "localhost :8090,:8091,8092"
var portNum string = "8080"
var endpoint int
var endpointList [3]int
var alive map[string]bool
var id NodeID

func main() {
	alive = make(map[string]bool)
	//var err error
	var idx string
	// initialize app
	app := iris.New()

	//Parse flags

	parseArg(os.Args[1], os.Args[2])
	parseArg(os.Args[3], os.Args[4])

	num,_ := strconv.Atoi(portNum)
	id = NodeID{Endpoint : portNum, nodeSerialNum : num - 8080 + 3}

	app.RegisterView(iris.HTML("./src/views", ".html"))

	app.Get("/", func(ctx iris.Context) {
		lst := client("INDEX", "0", "", fmt.Sprint(endpoint))
		ctx.ViewData("shoppingList", lst)
		ctx.View("index.html") 
	})
	
	//create
	app.Post("/create", func(ctx iris.Context) {
		name := ctx.PostValue("itemName")
		lst := client("CREATE", "0", name, fmt.Sprint(endpoint))
		ctx.ViewData("shoppingList",lst)
		ctx.View("index.html")
	})

	app.Get("/delete/{idx}", func(ctx iris.Context) {
		idx = ctx.PostValue("Index")
		lst := client("DELETE", idx, "", fmt.Sprint(endpoint))
		ctx.ViewData("shoppingList",lst)
		ctx.View("index.html")
		ctx.Redirect("/")
	})

	app.Get("/edit", func(ctx iris.Context) {

		ctx.View("edit.html")

	})

	app.Post("/update",func(ctx iris.Context) {
		name := ctx.PostValue("newName")
		idx := ctx.PostValue("idx")
		lst := client("UPDATE",idx,name,fmt.Sprint(endpoint))
		ctx.ViewData("shoppingList",lst)
		ctx.View("edit.html")
	})

	//check leader and update alive information
	go func(){
		for {
			time.Sleep(1 * time.Second)
			prt := strconv.Itoa(endpoint)
			lst := client("CHECK","0","",fmt.Sprint(prt))
			if lst[0] == "ERROR" {
				alive[prt] = false
				checkLeader(alive)
			}
		}
	}()

	// Run App or Listen on localhost
	app.Run(iris.Addr(":" + fmt.Sprint(portNum)))

}
func parseArg(name string, str string) {
	//fmt.Println(str)
	switch name {
	case "--listen":
		portNum = str
		
	case "--backend":
		backend := strings.Split(strings.Replace(str, ":", "", -1), ",")
		for i:= 0; i < len(backend);i++ {
			//fmt.Println(backend[i])
			ptr, _ := strconv.Atoi(backend[i])
			endpointList[i] = ptr
			alive[backend[i]] = true
			//fmt.Println(ptr)
		}
		endpoint = endpointList[0]
	}
}

func client(req string, idx string, name string, endpoint string) []string {
	conn, err := net.Dial("tcp", "localhost:"+ endpoint)
	if err != nil {
		fmt.Println("Detected failure on " + endpoint + " at " + time.Now().Format(time.RFC850))
		alive[endpoint] = false
		checkLeader(alive)
		return []string{"ERROR"}
	}
	defer conn.Close()

	message := Info(Info{ UniqueNum: -1, Name: name, Type: "",Method: req, Index: idx, Endpoint : id.Endpoint, nodeSerialNum : id.nodeSerialNum})
	//fmt.Println("senderNode :", id.nodeSerialNum)
	//encode message
	encoder := gob.NewEncoder(conn)
	err = encoder.Encode(message)
	if err != nil {
		panic(err)
	}
	var shoppingList []string

	//decode
	decoder := gob.NewDecoder(conn)
	for {
		e := decoder.Decode(&shoppingList)
		if e != nil {
			fmt.Println(e)
			return []string{"Panic!"}
		}
		return shoppingList
	} 
}


func checkLeader(alive map[string]bool){
	for i:= 0; i < 3; i++ {
		prt := strconv.Itoa(endpointList[i])
		if (alive[prt] == true) {
			lst := client("ALIVE","0","",fmt.Sprint(prt))
			if lst[0] == "ERROR" {
				alive[prt] = false
			}else{
				alive[prt] = true
			}
		}
	}

	const MaxUint = ^uint(0) 
	minimum := int(MaxUint >> 1) 
	for i:= 0; i < 3; i++ {
		prt :=strconv.Itoa(endpointList[i])
		if alive[prt] == true {
			if (endpointList[i] < minimum) {
				minimum = endpointList[i]
			}
		}
	}
	endpoint = minimum
	client("LEADER","0","",fmt.Sprint(endpoint))
	//give it a little time to update leader
	time.Sleep(1 * time.Second)
	fmt.Println("leader : ", endpoint)
}
