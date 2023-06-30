package model

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type TaskMap struct {
	tasks   map[string]*Task
	numDone int
	numAll  int
	mu      *sync.Mutex
	wg      *sync.WaitGroup
	f       *os.File
}

func CreateTaskMap(sld string) (*TaskMap, error) {
	// Create folder
	outputFilepath := filepath.Join(Opts.OutputFolder, fmt.Sprintf("%s.txt", sld))
	os.MkdirAll(filepath.Dir(outputFilepath), 0755)

	// Create file
	f, err := os.OpenFile(outputFilepath, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	// Create task map
	return &TaskMap{
		tasks: make(map[string]*Task),
		mu:    &sync.Mutex{},
		wg:    &sync.WaitGroup{},
		f:     f,
	}, nil
}

func (r *TaskMap) AddTask(domain string, sld string, manual bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tasks[domain]; !exists {
		r.tasks[domain] = &Task{
			Domain: domain,
			Sld:    sld,
			Manual: manual,
			State:  Todo,
		}
		r.wg.Add(1)
		r.numAll++
	}
}

func (r *TaskMap) DoneWithSuccess(domain string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	task := r.tasks[domain]
	task.State = Done
	r.numDone++
	r.wg.Done()

	r.f.WriteString(fmt.Sprintf("%s\n", domain))
}

func (r *TaskMap) DoneWithFail(domain string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	task := r.tasks[domain]
	task.State = Done
	r.numDone++
	r.wg.Done()

	if !task.Manual {
		r.f.WriteString(fmt.Sprintf("%s\n", domain))
	}
}

func (r *TaskMap) Wait() {
	r.wg.Wait()
	r.f.Close()
}

func (r *TaskMap) GetState() (done, all int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.numDone, len(r.tasks)
}

func (r *TaskMap) GetTask() (*Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, task := range r.tasks {
		if task.State == Todo {
			task.State = Running
			return task, nil
		}
	}
	return nil, fmt.Errorf("no more tasks")
}

func (r *TaskMap) CheckDone() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	return r.numDone == r.numAll
}
