// queue_test.go
package queue

import (
	"fmt"
	"runtime"
	"sync/atomic"
	"testing"
)

var counter int32
var q = New()

const NUM_ENQUEUE = 5000000

// queuer starts a goroutine that concurrently queues and enqueues shared incremental values until NUM_ENQUEUE is reached
func queuer(id int, dequeued chan int, done chan int) {
	go func() {
		runtime.LockOSThread()
		for {
			// Enqueuing a value
			c := int(atomic.AddInt32(&counter, 1))
			if c > NUM_ENQUEUE {
				break
			}

			q.Enqueue(c)

			// Dequeuing a value
			v, ok := q.Dequeue()
			if !ok {
				fmt.Println("Dequeue returned false")
			} else {
				dequeued <- v.(int)
			}
		}
		done <- id
	}()
}

// Test_Queue starts equal number of goroutines as there are CPU's, each goroutine Enqueueing and Dequeueing
// concurrently an increamental value until NUM_ENQUEUE has been reached.
// The test is passed if the dequeued values are the same as the enqueued
func Test_Queue(t *testing.T) {
	queuers := runtime.NumCPU()
	runtime.GOMAXPROCS(queuers)
	fmt.Printf("CPU's: %d\n", queuers)

	// Create the channel that will check read results. Buffer overkill
	dequeued := make(chan int, NUM_ENQUEUE)
	// Create the channel that will retrieve done message from the queuers
	done := make(chan int, queuers)
	// An array keeping count of the number of times each value is dequeued
	valcount := make([]int, NUM_ENQUEUE)

	for i := 0; i < queuers; i++ {
		// Starting up a queuer
		queuer(i, dequeued, done)
	}

	for i := 0; i < NUM_ENQUEUE; i++ {
		// Checking if any queuers are done
		/*for j:=0; j<queuers; j++ {
			select {
				case <-l_done[j]: fmt.Printf("Queuer %d is done\n", j)
				default:
			}
		}*/

		select {
		case v := <-dequeued:
			if v < 1 || v > NUM_ENQUEUE {
				t.Errorf("Value %d is out of range", v)
				break
			}
			valcount[v-1]++
		}
	}

	for i := 0; i < NUM_ENQUEUE; i++ {
		switch {
		case valcount[i] == 0:
			t.Errorf("Value %d never returned", i)
		case valcount[i] > 1:
			t.Errorf("Value %d returned %d times", i, valcount[i])
		}
	}

	// Checking if any queuers are done
	for j := 0; j < queuers; j++ {
		select {
		case id := <-done:
			fmt.Printf("Queuer %d is done\n", id)
		}
	}
}
