package main

/* Matric number: U096996N

This program is written in Go, Google's system language.
*/

import (
	"fmt"
	//"flag"
	"io/ioutil"
	"container/list"
)

var (
	GPID          = 0
	Init          = PCB{0, list.New(), CT{nil, list.New()}, Stat{"ready_s", Ready_List}, 0}
	Ready_List    = list.New()
	Resource_List = list.New()
	IO            = list.New()
)

/*// Command flag
var terminal = flag.Bool("t", false, "use terminal mode for input")
*/

// Structs
type Stat struct {
	Type string
	List *list.List
}

type CT struct {
	Parent *PCB
	Child  *list.List
}

type PCB struct {
	PID             int
	Other_Resources *list.List
	Creation_Tree   CT
	Status          Stat
	Priority        int
}

type RCB struct {
	RID          int
	Status       Stat
	Waiting_List *list.List
}

type IO_RCB struct {
	Waiting_List *list.List
}

// Operations on processes

// create new process
func (p *PCB) Create(priority int) {
	newP := PCB{newPID(),
		list.New(),
		CT{p, list.New()},
		Stat{"ready_s", Ready_List},
		priority}

	listInsert(&newP, p.Creation_Tree.Child)
	listInsert(&newP, Ready_List)
	Scheduler()
}

// suspend process
func (p *PCB) Suspend(pid int) {
	pcb := getPCB(pid)
	s := pcb.Status.Type
	if s == "blocked_a" || s == "blocked_s" {
		pcb.Status.Type = "blocked_s"
	} else {
		pcb.Status.Type = "ready_s"
	}
	Scheduler()
}

// activate process
func (p *PCB) Activate(pid int) {
	pcb := getPCB(pid)
	if pcb.Status.Type == "ready_s" {
		pcb.Status.Type = "ready_a"
		Scheduler()
	} else {
		pcb.Status.Type = "blocked_a"
	}
}

// destroy processes
func (p *PCB) Destroy(pid int) {
	pcb = getPCB(pid)
	killTree(pcb)
	Scheduler()
}

// kill creation_tree for given PCB
func killTree(p *PCB) {

}

// scheduler
func Scheduler() {

}

func (p *PCB) Request_IO() {
	p.Status.Type = "blocked_a"
	p.Status.List = IO
	listRemove(p, Ready_List)

	iowl := IO.Front().Value.(IO_RCB).Waiting_List
	listInsert(p, iowl)
	Scheduler()
}

// returns a new PID from the global var GPID
func newPID() int {
	GPID += 1
	return GPID
}

// get PCB based on pid by recursing through
// all children of Init
func getPCB(pid int) *PCB {
	ct := Init.Creation_Tree.Child
	for e := ct.Front(); e != nil; e = e.Next() {
		if e.Value.(PCB).PID == pid {
			return e.Value.(*PCB)
		}
	}
	return getChildPCB(ct, pid)
}

// helper function for getPCB
func getChildPCB(ls *list.List, pid int) *PCB {
	if ls == nil {
		return nil
	}

	for e := ls.Front(); e != nil; e = e.Next() {
		if e.Value.(PCB).PID == pid {
			return e.Value.(*PCB)
		} else {
			res := getChildPCB(e.Value.(*PCB).Creation_Tree.Child, pid)
			if res != nil {
				return res
			}
		}
	}
	return nil
}

// removes elements from list
func listRemove(p *PCB, ls *list.List) {
	for e := ls.Front(); e != nil; e = e.Next() {
		if e.Value.(PCB).PID == p.PID {
			ls.Remove(e)
		}
	}
}

// inserts process into list
func listInsert(p *PCB, ls *list.List) {
	e := new(list.Element)
	e.Value = p
	ls.PushFront(e)
}

func read(title string) string {
	filename := title
	body, err := ioutil.ReadFile(filename + ".txt")
	if err != nil {
		panic(err)
	}
	return string(body)
}

func main() {
	//flag.Parse()
	in := ""

	/*if *terminal {
		// REPL mode

	} else {
		// read file mode
	}*/

	fmt.Println("hello", in)
}
