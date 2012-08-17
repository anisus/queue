/*
	Package queue implements a non-blocking concurrent first-in-first-out queue

	The algorithm, the same as the one used in ConcurrentLinkedQueue in Java,
	is described in the paper:

		Simple, Fast, and Practical Non-Blocking and Blocking 
		Concurrent Queue Algorithms \Lambda 
		Maged M. Michael Michael L. Scott 
		Department of Computer Science 
		University of Rochester 
		Rochester, NY 14627-0226 
		fmichael,scottg@cs.rochester.edu
		http://www.cs.rochester.edu/u/scott/papers/1996_PODC_queues.pdf

	Modifications

	The original paper uses a counter and a CAS2 instruction to
	avoid the ABA problem. Since CAS2 instructions is not available in Go,
	the package uses a hack similar to that used in Microsoft Invisible Computing:

		http://research.microsoft.com/en-us/um/redmond/projects/invisible/src/queue/queue.c.htm
*/
package queue

import (
	"sync/atomic"
	"unsafe"
)

type node_t struct {
	value interface{}
	next  unsafe.Pointer
}

type Queue struct {
	head, tail unsafe.Pointer
}

// New returns an initialized queue
func New() *Queue {
	q := new(Queue)
	// Creating an initial node
	node := unsafe.Pointer(&node_t{nil, unsafe.Pointer(q)})

	// Both head and tail point to the initial node
	q.head = node
	q.tail = node
	return q
}

// Enqueue inserts the value at the tail of the queue
func (q *Queue) Enqueue(value interface{}) {
	node := new(node_t) // Allocate a new node from the free list
	node.value = value  // Copy enqueued value into node
	node.next = unsafe.Pointer(q)
	for { // Keep trying until Enqueue is done
		tail := atomic.LoadPointer(&q.tail)

		// Try to link in new node
		if atomic.CompareAndSwapPointer(&(*node_t)(tail).next, unsafe.Pointer(q), unsafe.Pointer(node)) {
			// Enqueue is done.  Try to swing tail to the inserted node.
			atomic.CompareAndSwapPointer(&q.tail, tail, unsafe.Pointer(node))
			return
		}

		// Try to swing tail to the next node as the tail was not pointing to the last node
		atomic.CompareAndSwapPointer(&q.tail, tail, (*node_t)(tail).next)
	}
}

// Dequeue returns the value at the head of the queue and true, or if the queue is empty, it returns a nil value and false
func (q *Queue) Dequeue() (value interface{}, ok bool) {
	for {
		head := atomic.LoadPointer(&q.head)			   // Read head pointer
		tail := atomic.LoadPointer(&q.tail)			   // Read tail pointer
		next := atomic.LoadPointer(&(*node_t)(head).next) // Read head.next
		if head != q.head {							   // Check head, tail, and next consistency
			continue // Not consistent. Try again
		}

		if head == tail { // Is queue empty or tail failing behind
			if next == unsafe.Pointer(q) { // Is queue empty?
				return
			}
			// Try to swing tail to the next node as the tail was not pointing to the last node
			atomic.CompareAndSwapPointer(&q.tail, tail, next)
		} else {
			// Read value before CAS
			// Otherwise, another dequeue might free the next node
			value = (*node_t)(next).value
			// Try to swing Head to the next node
			if atomic.CompareAndSwapPointer(&q.head, head, next) {
				ok = true
				return
			}
			value = nil
		}
	}
	return // Dummy return
}
