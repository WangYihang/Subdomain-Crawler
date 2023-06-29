package model

type TaskState int

const (
	Todo    TaskState = 0
	Running TaskState = 1
	Done    TaskState = 2
)

type Task struct {
	Sld    string
	Domain string
	Manual bool // true means that the task is added by hand
	State  TaskState
}
