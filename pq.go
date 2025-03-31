///////////////////////////////////////////////
///////////////////////////////////////////////
// SHIVAM KAK
// 31 MARCH 2025
// DUST TASK
///////////////////////////////////////////////
// FILE: pq.go

// Pq.go handles the implementation of
// necessary functions of the priority queue
// defined in interface.go

// IT IS RECOMMENDED TO READ THE DESIGN.TXT
// BEFORE PROCEEDING TO READ THIS FILE.
///////////////////////////////////////////////
package throttler

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	// First by priority (lower niceness = higher priority)
	if pq[i].niceness != pq[j].niceness {
		return pq[i].niceness < pq[j].niceness
	}
	// Then by timestamp (FIFO for same priority)
	return pq[i].timestamp.Before(pq[j].timestamp)
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*requestItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	
	// avoid memory leak
	old[n-1] = nil    

	// for safety
	item.index = -1   
	
	*pq = old[0 : n-1]
	return item
}