package util

import "container/heap"

//比較するために必要な情報をItemに持たせる
type Item struct {
	Value    interface{} // The value of the item; arbitrary.
	Priority int         // The priority of the item in the queue.
	// The index is needed by update and is maintained by the heap.Interface methods.
	index int // The index of the item in the heap.
}

func (i *Item) GetValue() interface{} {
	return i.Value
}

// A PriorityQueue implements heap.Interface and holds Items.
type PriorityQueueContent []*Item

func (pq PriorityQueueContent) Len() int { return len(pq) }

func (pq PriorityQueueContent) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return pq[i].Priority > pq[j].Priority
}

func (pq PriorityQueueContent) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueueContent) Push(x interface{}) {
	n := len(*pq)
	item := x.(*Item)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueueContent) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]

	return item
}

func GeneratePriorityQueue() *PriorityQueue {
	q := make(PriorityQueueContent, 0)
	return &PriorityQueue{
		Queue: q,
	}
}

type PriorityQueue struct {
	Queue PriorityQueueContent
}

func (p *PriorityQueue) Push(i interface{}) {
	heap.Push(&p.Queue, i)
}
func (p *PriorityQueue) Pop() interface{} {
	in := heap.Pop(&p.Queue)
	item := in.(*Item)
	return item.GetValue() //Itemを返してもItemhの生存期間はPop()関数の中なので呼び出し下ではメモリから消えているっぽい？なので値だけ返す
}
