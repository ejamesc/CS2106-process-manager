package main

import (
	"fmt"
	"bufio"
	"flag"
	"os"
	"strings"
	"strconv"
	"container/list"
)

/* Matriculation number: U096996N
CS2106 Project Assignment 1
This program is written in Go r60.3.
Compiled using the 6g compiler version release.r60.3 9516+ on Darwin
Usage:
	./6.out      (file mode)
	./6.out -t   (REPL mode)
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
	PID             string
	Other_Resources *list.List
	Creation_Tree   CT
	Status          Stat
	Priority        int
}

type RCB struct {
	RID          string
	Status       string
	Waiting_List *list.List
}

type IO_RCB struct {
	Waiting_List *list.List
}

// Global Variables
// current running process and init process
var Curr, Init *PCB
var IO *IO_RCB
var terminal = flag.Bool("t", false, "use terminal mode for input")
var (
	PIDs          = make(map[string]*PCB) // keeps track of all processes
	Ready_List    = list.New()
	Resource_List = list.New()
)

// Operations on processes
// init for PCB, rarely used
func (p *PCB) Init() *PCB {
	p.PID = ""
	p.Other_Resources = list.New()
	p.Creation_Tree = CT{p, list.New()}
	p.Status = Stat{"ready_s", Ready_List}
	p.Priority = 0
	return p
}

// create new process
// returns os.Error if PID is not unique, or priority is wrong
// returns nil on success
func (p *PCB) Create(name string, priority int) os.Error {
	if _, ok := PIDs[name]; ok {
		return os.NewError("PID already taken")
	}
	if priority > 2 {
		return os.NewError("No such priority")
	}
	newP := PCB{name,
		list.New(),
		CT{p, list.New()},
		Stat{"ready_s", Ready_List},
		priority}

	PIDs[name] = &newP // add to PID name records
	p.Creation_Tree.Child.PushFront(&newP)
	listRLInsert(&newP)
	Scheduler()

	return nil
}

// destroy process
func (p *PCB) Destroy(pid string) {
	pcb := getPCB(pid)
	if pcb != Init {
		killTree(pcb)
	} else {
		fmt.Println("init cannot be destroyed")
	}
	Scheduler()
}

// kill creation_tree for given PCB
func killTree(p *PCB) {
	c := p.Creation_Tree.Child
	children := make([]*PCB, c.Len())

	for e := c.Front(); e != nil; e = e.Next() {
		children = append(children, e.Value.(*PCB))
	}

	for _, chld := range children {
		if chld != nil {
			killTree(chld)
		}
	}

	if p.Status.List == Ready_List {
		listRLRemove(p)
	} else {
		listRemove(p, p.Status.List)
	}

	// takes care of case where running PCB is deleted
	if p.Status.Type == "running" {
		Curr = nil
	}
	parent := p.Creation_Tree.Parent
	listRemove(p, parent.Creation_Tree.Child)

	// release all resources associated with p
	for e := p.Other_Resources.Front(); e != nil; e = e.Next() {
		p.release(e.Value.(*RCB).RID)
	}
	// remove PID and PCB pointer from map 'PIDs'
	PIDs[p.PID] = nil, false
}

// request a resource
func (p *PCB) Request(rid string) {
	if p == Init {
		fmt.Println("init not allowed to request resource")
		return
	}
	r := getRCB(rid)
	if r.Status == "free" {
		r.Status = "allocated"
		p.Other_Resources.PushBack(r)
	} else {
		r.Waiting_List.PushBack(p)
		listRLRemove(p)
		fmt.Printf("Process %s blocked; ", Curr.PID)
		p.Status.Type = "blocked_a"
		p.Status.List = r.Waiting_List
	}
	Scheduler()
}

// release a resource
func (p *PCB) Release(rid string) {
	p.release(rid)
	Scheduler()
}

// release a resource, without call to Scheduler
// for internal use only, e.g.: Destroy
func (p *PCB) release(rid string) {
	r := getRCB(rid)
	if r.Waiting_List.Len() == 0 {
		r.Status = "free"
	} else {
		pcb := r.Waiting_List.Front().Value.(*PCB)
		rcbListRemove(r, pcb.Other_Resources)
		r.Waiting_List.Remove(r.Waiting_List.Front()) // remove front
		pcb.Status.Type = "ready_a"
		pcb.Status.List = Ready_List
		listRLInsert(pcb)
	}
}

// timeout function
func (p *PCB) Time_out() {
	listRLInsert(Curr) // place pointer to Curr running p back into RL
	Curr.Status.Type = "ready_a"
	Curr = nil
	Scheduler()
}

// request IO resource
func (p *PCB) Request_IO() {
	p.Status.Type = "blocked_a"
	p.Status.List = IO.Waiting_List
	listRLRemove(p)
	fmt.Printf("Process %s blocked;", p.PID)

	iowl := IO.Waiting_List
	iowl.PushBack(p)
	Scheduler()
}

// IO release
func (p *PCB) IO_completion() {
	if IO.Waiting_List.Len() != 0 {
		pcb := IO.Waiting_List.Front().Value.(*PCB)
		listRemove(pcb, IO.Waiting_List)
		pcb.Status.Type = "ready"
		pcb.Status.List = Ready_List
		listRLInsert(pcb)
		Scheduler()
	} else {
		fmt.Println("No processes on IO")
	}
}

// calculates which process to run next
// also prints state
func Scheduler() {
	p := maxPriorityPCB()
	if Curr == nil || Curr.Status.Type != "running" || Curr.Priority < p.Priority {
		preempt(p, Curr)
	}
	fmt.Printf("Process %s is running\n", Curr.PID)
}

// preempt function used in scheduler
// replaces Curr running process with p
func preempt(p, prev *PCB) {
	if prev != nil {
		if prev.Status.Type != "blocked_a" {
			prev.Status.Type = "ready_a"
			if prev != Init { // edge case, init doesn't need to be re-added to RL
				listRLInsert(prev)
			}
		}
	}

	Curr = p
	p.Status.Type = "running"
	listRLRemove(p)
}

// find and return the highest priority PCB
// note that Curr, the current running PCB is not in the RL
func maxPriorityPCB() *PCB {
	system := Ready_List.Front()
	user := system.Next()
	init := user.Next()

	switch {
	// get top process from priority level 2
	case system.Value.(*list.List).Len() != 0:
		return system.Value.(*list.List).Front().Value.(*PCB)
	// get top process from priority level 1
	case user.Value.(*list.List).Len() != 0:
		return user.Value.(*list.List).Front().Value.(*PCB)
	// get top process from priority level 0
	case init.Value.(*list.List).Len() > 1:
		return init.Value.(*list.List).Front().Value.(*PCB)
	}
	return Init // return init
}

// main program
func main() {
	flag.Parse()

	in := bufio.NewReader(os.Stdin)

	// REPL mode
	if *terminal {
		initialize()

		var (
			i   string
			err os.Error
		)

		for {
			i, err = in.ReadString('\n')
			if err != nil {
				fmt.Println("Read error:", err)
			}
			i = strings.TrimSpace(i)

			if i == "quit" && len(strings.Split(i, " ")) == 1 {
				fmt.Println("process terminated")
				break
			}
			Manager(i)
		}
	// file mode
	} else {
		var (
			tmp        string
			error, err os.Error
			file       *os.File
			lines      []string
			path       string
		)

		// get the file path
		path, err = in.ReadString('\n')
		if err != nil {
			fmt.Println("Read error:", err)
		}
		// open the file
		path = strings.TrimSpace(path)
		if file, err = os.Open(path); err != nil {
			fmt.Println("File open error:", err)
		}
		// declare a new file reader
		reader := bufio.NewReader(file)
		// load each command into a slice named line
		for {
			if tmp, error = reader.ReadString('\n'); error == nil {
				tmp = strings.TrimSpace(tmp)
				lines = append(lines, tmp)
			}
			if error == os.EOF {
				break
			}
			if error != nil {
				fmt.Println("Readline error:", error)
				break
			}
		}
		// initialize the kernel environment
		initialize()
		// for each command, process it
		for _, v := range lines {
			Manager(v)
		}
	}

}

// Helper functions
// set up all the structs needed for the program to run
func initialize() {
	fmt.Print("init")

	// clear the global lists
	Ready_List.Init()
	Resource_List.Init()

	IO = &IO_RCB{list.New()}

	PIDs = make(map[string]*PCB)

	Init = &PCB{
		"init",
		list.New(),
		CT{nil, list.New()},
		Stat{"ready_s", Ready_List},
		0}
	Curr = Init
	PIDs["init"] = Init

	Ready_List.PushFront(list.New())
	Ready_List.PushFront(list.New())
	Ready_List.PushFront(list.New())

	listRLInsert(Init)

	Resource_List.PushFront(&RCB{"R1", "free", list.New()})
	Resource_List.PushFront(&RCB{"R2", "free", list.New()})
	Resource_List.PushFront(&RCB{"R3", "free", list.New()})
	Resource_List.PushFront(&RCB{"R4", "free", list.New()})

	fmt.Println(" ... done\nProcess init is running")
}

// handles commands and dispatches the appropirate operations
func Manager(cmd string) {
	cmds := strings.Split(cmd, " ")

	switch ins := cmds[0]; {
	case ins == "cr" && len(cmds) == 3:
		x, _ := strconv.Atoi(cmds[2])
		err := Curr.Create(cmds[1], x)
		if err != nil {
			fmt.Println(err)
		}
	case ins == "de" && len(cmds) == 2:
		Curr.Destroy(cmds[1])
	case ins == "req" && len(cmds) == 2:
		Curr.Request(cmds[1])
	case ins == "rel" && len(cmds) == 2:
		Curr.Release(cmds[1])
	case ins == "to" && len(cmds) == 1:
		Curr.Time_out()
	case ins == "init" && len(cmds) == 1:
		initialize()
	case ins == "rio" && len(cmds) == 1:
		Curr.Request_IO()
	case ins == "ioc" && len(cmds) == 1:
		Curr.IO_completion()
	case ins == "" && len(cmds) == 1:
		fmt.Println("")
	case ins == "quit" && len(cmds) == 1:
		fmt.Println("process terminated")
		break
	default:
		fmt.Println("Unknown command")
	}

}

// calculates where to place processes on the RL
func listRLInsert(p *PCB) {
	pr := p.Priority
	var e *list.Element

	switch {
	case pr == 2:
		e = Ready_List.Front()
	case pr == 1:
		e = Ready_List.Front().Next()
	case pr == 0:
		e = Ready_List.Front().Next().Next()
	}
	ls := e.Value.(*list.List)
	ls.PushBack(p)
}

// removes process from the RL
func listRLRemove(p *PCB) {
	pr := p.Priority
	var e *list.Element

	switch {
	case pr == 2:
		e = Ready_List.Front()
	case pr == 1:
		e = Ready_List.Front().Next()
	case pr == 0:
		e = Ready_List.Front().Next().Next()
	}
	ls := e.Value.(*list.List)
	listRemove(p, ls)
}

// removes PCB element from a linked list
func listRemove(p *PCB, ls *list.List) {
	for e := ls.Front(); e != nil; e = e.Next() {
		if e.Value.(*PCB).PID == p.PID {
			ls.Remove(e)
		}
	}
}

// removes RCB element from a linked list
func rcbListRemove(r *RCB, ls *list.List) {
	for e := ls.Front(); e != nil; e = e.Next() {
		if e.Value.(*RCB).RID == r.RID {
			ls.Remove(e)
		}
	}
}

// returns PCB item given PID
func getPCB(name string) *PCB {
	if res, ok := PIDs[name]; ok {
		return res
	}
	return nil
}

// returns RCB item given RID
func getRCB(rid string) *RCB {
	for e := Resource_List.Front(); e != nil; e = e.Next() {
		if e.Value.(*RCB).RID == rid {
			return e.Value.(*RCB)
		}
	}
	return nil
}
