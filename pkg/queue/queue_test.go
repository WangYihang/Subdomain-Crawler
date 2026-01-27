package queue

import "testing"

func TestJobQueueEnqueueDequeue(t *testing.T) {
	q := NewJobQueue(10)
	task := Task{Domain: "example.com", Depth: 1, Root: "example.com"}

	if !q.Enqueue(task) {
		t.Errorf("Enqueue should succeed")
	}

	retrieved, ok := q.Dequeue()
	if !ok || retrieved.Domain != task.Domain {
		t.Errorf("Dequeue failed")
	}
}

func TestJobQueueFull(t *testing.T) {
	q := NewJobQueue(2)
	task := Task{Domain: "test.com"}

	q.Enqueue(task)
	q.Enqueue(task)

	if q.Enqueue(task) {
		t.Errorf("Enqueue should fail when full")
	}
}

func TestJobQueueClose(t *testing.T) {
	q := NewJobQueue(10)
	q.Enqueue(Task{Domain: "test.com"})
	q.Close()

	_, ok := q.Dequeue()
	if !ok {
		t.Errorf("Should dequeue remaining task")
	}

	_, ok = q.Dequeue()
	if ok {
		t.Errorf("Dequeue should fail after close")
	}
}

func TestResultQueueSendReceive(t *testing.T) {
	q := NewResultQueue(10)
	result := Result{
		Domain:     "example.com",
		Root:       "example.com",
		Subdomains: []string{"www.example.com"},
	}

	if !q.Send(result) {
		t.Errorf("Send should succeed")
	}

	retrieved, ok := q.Receive()
	if !ok || retrieved.Domain != result.Domain {
		t.Errorf("Receive failed")
	}
}
