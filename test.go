package main

import (
	"fmt"
	"bufio"
	"os"
	"strings"
	"strconv"
	"container/list"
)

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

// Operations on processes
// init for PCB
func (p *PCB) Init() *PCB {
	p.PID = ""
	p.Other_Resources = list.New()
	p.Creation_Tree = CT{p, list.New()}
	p.Status = Stat{"ready_s", Ready_List}
	p.Priority = 0
	return p
}

// create new process
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
	fmt.Printf("process %s | %v\n", p.PID, p.Creation_Tree.Child.Len())
	listRLInsert(&newP)
	Scheduler()

	return nil
}

// destroy process
func (p *PCB) Destroy(pid string) {
	pcb := getPCB(pid)

	killTree(pcb)
	Scheduler()
}

// kill creation_tree for given PCB
func killTree(p *PCB) {
	for e := p.Creation_Tree.Child.Front(); e != nil; e = e.Next() {
		fmt.Println(e.Value.(*PCB).PID)
		killTree(e.Value.(*PCB))
	}

	if p.Status.List == Ready_List {
		fmt.Println("Delete from RL", p.PID)
		listRLRemove(p)
		showRL()
	} else {
		listRemove(p, p.Status.List)
	}

	// takes care of case where running PCB is deleted
	if p.Status.Type == "running" {
		Curr = nil
	}
	parent := p.Creation_Tree.Parent
	listRemove(p, parent.Creation_Tree.Child)
	// need to call release on all resources
	PIDs[p.PID] = nil, false
}

func Scheduler() {
	p := maxPriorityPCB()
	fmt.Println("Top process:", p.PID)
	if Curr == nil || Curr.Priority < p.Priority || Curr.Status.Type != "running"{
		preempt(p, Curr)
	}
	// print here, in case preempt does not occur
	fmt.Printf("Process %s is running\n", Curr.PID)
}

// replaces curr with p
func preempt(p, prev *PCB) {
	if prev != nil {
		prev.Status.Type = "ready_a"
		if prev.Status.List != Ready_List {
			listRLInsert(prev)
		}
	}
	Curr = p
	p.Status.Type = "running"

	listRLRemove(p)
}

func showRL() {
	system := Ready_List.Front()
	user := system.Next()
	init := user.Next()

	fmt.Println("Level 2:", system.Value.(*list.List).Len())
	fmt.Println("Level 1:", user.Value.(*list.List).Len())
	fmt.Println("Level 0:", init.Value.(*list.List).Len())
}

// find and return the highest priority PCB
func maxPriorityPCB() *PCB {
	system := Ready_List.Front()
	user := system.Next()
	init := user.Next()

	showRL()

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

// Variables
// current running process
var Curr, Init *PCB
var (
	PIDs          = make(map[string]*PCB) // keeps tracks of PIDs
	Ready_List    = list.New()
	Resource_List = list.New()
	IO            = list.New()
)

func main() {
	i := ""
	var err os.Error
	in := bufio.NewReader(os.Stdin)
	initialize()

	for {
		i, err = in.ReadString('\n')
		if err != nil {
			fmt.Println("Read error: ", err)
		}
		i = strings.TrimSpace(i)

		if i == "quit" && len(strings.Split(i, " ")) == 1 {
			fmt.Println("Exiting")
			break
		}
		Manager(i)
	}
}

// Helper functions
// set up all the structs needed for the program to run
func initialize() {
	fmt.Print("init")
	Init = &PCB{
		"init",
		list.New(),
		CT{nil, list.New()},
		Stat{"ready_s", Ready_List},
		0}
	Curr = Init

	Ready_List.PushFront(list.New())
	Ready_List.PushFront(list.New())
	Ready_List.PushFront(list.New())

	listRLInsert(Init)

	Resource_List.PushFront(&RCB{"R1", "free", list.New()})
	Resource_List.PushFront(&RCB{"R2", "free", list.New()})
	Resource_List.PushFront(&RCB{"R3", "free", list.New()})

	IO.PushFront(IO_RCB{list.New()})
	fmt.Println(" ... done")
}

// handles commands and dispatches the appropirate ops
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
	default:
		fmt.Println("Unknown command")
	}

}

// Calculates where to place processes on the RL
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

// Removes RL
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

// removes PCB element from list
func listRemove(p *PCB, ls *list.List) {
	for e := ls.Front(); e != nil; e = e.Next() {
		if e.Value.(*PCB).PID == p.PID {
			ls.Remove(e)
		}
	}
}

func getPCB(name string) *PCB {
	if res, ok := PIDs[name]; ok {
		return res
	}
	return nil
}
