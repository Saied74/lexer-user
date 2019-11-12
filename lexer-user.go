//Moving the code to test the parser as a user to this package and turning
//parser itself to a parser package.

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	parser "service/lexer"
	"strings"
)

const seqFlow = "SequenceFlow"

type element struct {
	VarPart  map[string]string
	OutNodes []string //sequenceFlow ID
	InNodes  []string //sequenceFlow ID
	ToLinks  []int
}

//AllNodes used to be exported but not anymore
var AllNodes []element // TODO: need to get rid of the export
var currNode = element{
	VarPart:  map[string]string{}, //{"Foo": "Foo"},
	OutNodes: []string{},
	InNodes:  []string{},
	ToLinks:  []int{},
}

func getPattern(fileName string) ([][]string, error) {
	var pattern [][]string
	content, err := ioutil.ReadFile(fileName) //get the whole file
	if err != nil {
		return [][]string{}, fmt.Errorf("open error %v on file %s", err, fileName)
	}
	pat1 := strings.Split(string(content), "\n") //split into lines
	for _, pat2 := range pat1 {                  //scan the lines
		pat3 := strings.Split(pat2, "|") //split into comma seperated fields
		pattern = append(pattern, pat3)  //append to the output
	}
	return pattern, nil
}

func getPrintOrder(pattern [][]string) []string {
	for _, pat := range pattern {
		if pat[0] == "print" {
			return pat[1:]
		}
	}
	return []string{}
}

func printOutput(printOrder []string) {
	fmt.Println("-------------------------------------------------------------")
	for n, item := range AllNodes {
		fmt.Println("N = ", n)
		fmt.Println("-------------------------------------------------------------")
		for _, key := range printOrder {
			fmt.Println(key, ": \t", item.VarPart[key])
		}
		fmt.Println("Out Nodes: ", item.OutNodes)
		fmt.Println("In Nodes: ", item.InNodes)
		fmt.Println("To Links: ", item.ToLinks)
		fmt.Println("-------------------------------------------------------------")
	}
}

func findStart() (int, error) {
	// var starts []elemen
	var places []int
	for n, start := range AllNodes {
		if start.VarPart["nodeType"] == "startEvent" {
			// starts = append(starts, start)
			places = append(places, n)
		}
	}
	if len(places) == 0 {
		return -1, fmt.Errorf("start node was not found")
	}
	if len(places) == 1 {
		return places[0], nil
	}
	return -1, fmt.Errorf("more than one start was found at %v", places)
}

//current node is an edge, so it will link to only one other node
//edge nodes target field points to the ID of the next action node
func nextNode(currNode int) (int, bool) {
	for n, node := range AllNodes {
		if node.VarPart["id"] == AllNodes[currNode].VarPart["targetRef"] {
			return n, true
		}
	}
	return -1, false
}

//current node is an action node, next nodes are edge nodes
//an action node can have many nodes eminating from it.
func nextEdge(currNode int) ([]int, bool) {
	var nextNext []int
	for _, link1 := range AllNodes[currNode].OutNodes {
		for n, node := range AllNodes {
			if link1 == node.VarPart["id"] {
				nextNext = append(nextNext, n)
			}
		}
	}
	return nextNext, true
}

func linkChain() {
	for n, node := range AllNodes {
		if !strings.HasPrefix(node.VarPart["id"], seqFlow) {
			nextNext, ok := nextEdge(n)
			if !ok {
				fmt.Println("nextNext", nextNext, AllNodes[nextNext[0]].VarPart["nodeID"])
			}
			for _, nextN := range nextNext {
				m, ok := nextNode(nextN)
				if !ok {
					fmt.Println("M: ", m)
					os.Exit(1)
				}
				node.ToLinks = append(node.ToLinks, m)
				AllNodes[n].ToLinks = node.ToLinks
			}
		}
	}
	return
}

type tracker struct {
	nodes   []int //nodes to revisit
	pos     int   //position of nodes to visit next
	node    int   //node to visit
	second  bool  //marker for the second run
	visited []int //nodes visited
}

type stateFn func(t *tracker) stateFn

func (t *tracker) run() {
	for state := firstChain; state != nil; {
		state = state(t)
	}
}

func (t *tracker) next() bool {
	if t.pos >= len(t.nodes) {
		return false
	}
	t.pos++
	return true
}

var tkr = tracker{[]int{0}, 0, 0, false, []int{}}
var t = &tkr

func (t *tracker) hasInt(n int) bool {
	for _, i := range t.nodes {
		if i == n {
			return true
		}
	}
	return false
}

func (t *tracker) haveVisited(n int) bool {
	for _, i := range t.visited {
		if i == n {
			return true
		}
	}
	return false
}

func multiIn(n int) bool {
	if len(AllNodes[n].InNodes) > 1 {
		return true
	}
	return false
}

func multiOut(n int) bool {
	counts := len(AllNodes[n].OutNodes)
	if counts > 1 {
		return true
	}
	return false
}

func noOut(n int) bool {
	if len(AllNodes[n].OutNodes) == 0 {
		return true
	}
	return false
}

func firstChain(t *tracker) stateFn {
	if strings.HasPrefix(AllNodes[t.node].VarPart["id"], seqFlow) {
		t.node++
		return firstChain
	}
	if !t.haveVisited(t.node) {
		fmt.Printf("Location: %d, ID: %s\n", t.node,
			AllNodes[t.node].VarPart["id"])
		t.visited = append(t.visited, t.node)
	}
	if !multiIn(t.node) && !multiOut(t.node) && !noOut(t.node) {
		t.node = AllNodes[t.node].ToLinks[0]
		return firstChain
	}
	if multiIn(t.node) {
		if noOut(t.node) {
			return nil
		}
		if multiOut(t.node) {
			if !t.hasInt(t.node) {
				t.nodes = append(t.nodes, t.node)
			}
		}
		t.node = AllNodes[t.node].ToLinks[0]
		return firstChain
	}
	if multiOut(t.node) {
		if !t.hasInt(t.node) {
			t.nodes = append(t.nodes, t.node)
		}
		t.node = AllNodes[t.node].ToLinks[0]
		return firstChain
	}
	return nil
}

func secondChain(t *tracker) {
	for _, i := range AllNodes[t.node].ToLinks[1:] {
		t.node = i
		t.run()
	}
}

func getItems(pattern [][]string, dat string) {
	item := parser.Lex(pattern, dat)
	for {
		newItem := <-item
		switch newItem.ItemKey {
		case "nodeType":
			if currNode.VarPart == nil {
				currNode.VarPart = map[string]string{"nodeType": newItem.ItemValue}
			} else {
				currNode.VarPart["nodeType"] = newItem.ItemValue
			}
		case "object":
			AllNodes = append(AllNodes, currNode)
			currNode = element{}
		case "EOF":
			return
		default:
			currNode.VarPart[newItem.ItemKey] = newItem.ItemValue
			switch newItem.ItemKey {
			case "incoming":
				currNode.InNodes = append(currNode.InNodes, newItem.ItemValue)
			case "outgoing":
				currNode.OutNodes = append(currNode.OutNodes, newItem.ItemValue)
			}
		}
	}
}

func main() {
	if len(os.Args) == 1 {
		log.Fatal("Please suppy filename, it was missing")
	}
	fileName := os.Args[1]
	splitName := strings.Split(fileName, ".")
	if len(splitName) > 1 {
		log.Fatal("Please supply a filename without qualifier, you entered: ",
			fileName)
	}
	dat, err := ioutil.ReadFile(
		os.Getenv("SERVDATA") + "/bpmn/" + fileName + ".bpmn")
	if err != nil {
		log.Fatal(err)
	}

	pattern, err := getPattern("pattern.csv")
	if err != nil {
		log.Fatal("reading pattern", err)
	}

	printOrder := getPrintOrder(pattern)

	getItems(pattern, string(dat))
	linkChain()
	printOutput(printOrder)

	n, err := findStart()
	if err != nil {
		log.Fatal("got error from find start", err)
	}
	t.nodes = []int{n}
	t.pos = 0
	t.node, _ = findStart()
	t.run()
	for _, i := range t.nodes {
		t.node = i
		secondChain(t)
	}
}
