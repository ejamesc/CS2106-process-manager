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
		p.Release(e.Value.(*RCB).RID)
	}

	PIDs[p.PID] = nil, false
}

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
		p.Status.Type = "blocked_a"
		p.Status.List = r.Waiting_List
	}
	Scheduler()
}

func (p *PCB) Release(rid string) {
	r := getRCB(rid)
	rcbListRemove(r, p.Other_Resources)
	if r.Waiting_List.Len() == 0 {
		r.Status = "free"
	} else {
		r.Waiting_List.Remove(r.Waiting_List.Front()) // remove front
		p.Status.Type = "ready_a"
		p.Status.List = Ready_List
		listRLInsert(p)
	}
	Scheduler()
}

// timeout function
func (p *PCB) Time_out() {
	listRLInsert(Curr) // place pointer to Curr running p back into RL
	Curr.Status.Type = "ready"
	Curr = nil
	Scheduler()
}

func Scheduler() {
	p := maxPriorityPCB()
	fmt.Println("Top process:", p.PID)
	if Curr == nil || Curr.Status.Type != "running" || Curr.Priority < p.Priority {
		if Curr.Status.Type != "running" {
			fmt.Printf("Process %s blocked; ", Curr.PID)
		}
		preempt(p, Curr)
	}
	// print state
	fmt.Printf("Process %s is running\n", Curr.PID)
	showRL()
}

// replaces curr with p
func preempt(p, prev *PCB) {
	if prev != nil {
		prev.Status.Type = "ready_a"
		if prev != Init { // edge case, init doesn't need to be re-added to RL
			listRLInsert(prev)
		}
	}

	Curr = p
	p.Status.Type = "running"
	listRLRemove(p)
}

// temp function
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
			fmt.Println("process terminated")
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
	PIDs["init"] = Init

	Ready_List.PushFront(list.New())
	Ready_List.PushFront(list.New())
	Ready_List.PushFront(list.New())

	listRLInsert(Init)

	Resource_List.PushFront(&RCB{"R1", "free", list.New()})
	Resource_List.PushFront(&RCB{"R2", "free", list.New()})
	Resource_List.PushFront(&RCB{"R3", "free", list.New()})

	IO.PushFront(IO_RCB{list.New()})
	fmt.Println(" ... done\nProcess init is running")
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
	case ins == "req" && len(cmds) == 2:
		Curr.Request(cmds[1])
	case ins == "rel" && len(cmds) == 2:
		Curr.Release(cmds[1])
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

func rcbListRemove(r *RCB, ls *list.List) {
	for e := ls.Front(); e != nil; e = e.Next() {
		if e.Value.(*RCB).RID == r.RID {
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

func getRCB(rid string) *RCB {
	for e := Resource_List.Front(); e != nil; e = e.Next() {
		if e.Value.(*RCB).RID == rid {
			return e.Value.(*RCB)
		}
	}
	return nil
}
