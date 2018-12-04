package server

import "container/list"

type BlocksBulkList struct {
	l *list.List
}

func NewBlocksBulkList() *BlocksBulkList {
	return &BlocksBulkList{
		l: list.New(),
	}
}

func (l *BlocksBulkList) AddBlocks(start uint64, insertBlocks []*WalletBlock) {
	blocks := make([]*WalletBlock, len(insertBlocks))
	copy(blocks, insertBlocks)

	inserted := false
	for e := l.l.Front(); e != nil; {
		bulk := e.Value.(*blocksBulk)

		if bulk.nextHeight() >= start {
			e, start, blocks = l.doAddBlocks(e, start, blocks)
			inserted = true
			continue
		}

		e = e.Next()
	}

	if !inserted {
		l.doAddBlocks(nil, start, blocks)
	}
}

func (l *BlocksBulkList) RemoveBlocks(start uint64, count int) {
	for e := l.l.Front(); e != nil; {
		bulk := e.Value.(*blocksBulk)
		if bulk.nextHeight() >= start {
			e = l.doRemoveBlocks(e, start, count)
			continue
		}

		e = e.Next()
	}
}

func (l *BlocksBulkList) TrimBlocks(topHeight uint64) {
	for e := l.l.Front(); e != nil; {
		bulk := e.Value.(*blocksBulk)
		if bulk.nextHeight() <= topHeight {
			e = e.Next()
			continue
		}

		e = l.trimBulk(e, topHeight)
	}
}

func (l *BlocksBulkList) BlocksAvailable(start uint64) int {
	var e *list.Element
	for e = l.l.Front(); e != nil; e = e.Next() {
		bulk := e.Value.(*blocksBulk)

		if bulk.nextHeight() > start {
			break
		}
	}

	if e == nil {
		return 0
	}

	bulk := e.Value.(*blocksBulk)
	if bulk.start > start {
		return 0
	}

	return int(bulk.nextHeight() - start)
}

func (l *BlocksBulkList) GetBlocks(start uint64, maxCount int) []*WalletBlock {
	var e *list.Element
	for e = l.l.Front(); e != nil; e = e.Next() {
		bulk := e.Value.(*blocksBulk)

		if bulk.nextHeight() > start {
			break
		}
	}

	if e == nil {
		return nil
	}

	bulk := e.Value.(*blocksBulk)
	if bulk.start > start {
		return nil
	}

	s := start - bulk.start

	f := int(s) + maxCount
	if len(bulk.blocks) < f {
		f = len(bulk.blocks)
	}
	return bulk.blocks[s:f]
}

// Returns the first place (`start height`, `blocks count`) where to fetch blocks from
// If list is empty returns 0, 0
// If there's no missing blocks returns `next height`, 0
func (l *BlocksBulkList) FindMissingBlocks() (uint64, int) {
	var start uint64
	var count int

	e := l.l.Front()
	if e == nil {
		return start, count
	}

	bulk := e.Value.(*blocksBulk)
	start = bulk.nextHeight()

	for e = e.Next(); e != nil; e = e.Next() {
		bulk = e.Value.(*blocksBulk)

		count = int(bulk.start - start)
		if !bulk.empty() {
			break
		}
	}

	if l.l.Back().Value.(*blocksBulk).empty() {
		count = 0
	}

	return start, count
}

func (l *BlocksBulkList) doAddBlocks(e *list.Element, start uint64, blocks []*WalletBlock) (*list.Element, uint64, []*WalletBlock) {
	if e == nil {
		l.l.PushBack(makeBulk(start, blocks))
		return nil, 0, nil
	}

	bulk := e.Value.(*blocksBulk)
	if start < bulk.start {
		// new bulk starts earlier than current one
		if len(blocks) < int(bulk.start-start) {
			// there's space left till next bulk, no need to merge
			l.l.InsertBefore(makeBulk(start, blocks), e)
			return nil, 0, nil
		}

		// bulks intersect we need to merge them
		newb := makeBulk(start, blocks[0:bulk.start-start])
		newb.blocks = append(newb.blocks, bulk.blocks...)

		el := l.l.InsertBefore(newb, e)
		l.l.Remove(e)
		return el, bulk.start, blocks[bulk.start-start:]
	}

	s := int(bulk.nextHeight() - start)
	if len(blocks) < s {
		return nil, 0, nil
	}

	nextElem := e.Next()
	if nextElem == nil {
		bulk.blocks = append(bulk.blocks, blocks[s:]...)
		return nil, 0, nil
	}

	nextBulk := nextElem.Value.(*blocksBulk)
	if start+uint64(len(blocks)) < nextBulk.start {
		bulk.blocks = append(bulk.blocks, blocks[s:]...)
		return nil, 0, nil
	}

	// filling the gap between bulks
	bulk.blocks = append(bulk.blocks, blocks[s:nextBulk.start-start]...)

	// stealing blocks from the next bulk
	bulk.blocks = append(bulk.blocks, nextBulk.blocks...)
	l.l.Remove(nextElem)

	newBlocks := blocks[nextBulk.start-start:]
	newStart := nextBulk.start

	return e, newStart, newBlocks
}

func (l *BlocksBulkList) doRemoveBlocks(e *list.Element, start uint64, count int) *list.Element {
	if e == nil {
		return nil
	}

	bulk := e.Value.(*blocksBulk)

	if start < bulk.start {
		if bulk.start-start >= uint64(count) {
			return nil
		}

		trim := count - int(bulk.start-start)
		if len(bulk.blocks) <= trim {
			next := e.Next()
			l.l.Remove(e)
			return next
		}

		bulk.start = start + uint64(count)
		bulk.blocks = bulk.blocks[trim:]
		return nil
	}

	if start+uint64(count) >= bulk.nextHeight() {
		bulk.blocks = bulk.blocks[:start-bulk.start]
		next := e.Next()
		if len(bulk.blocks) == 0 {
			l.l.Remove(e)
		}

		return next
	}

	startInd := (uint64(len(bulk.blocks)) + start + uint64(count)) - bulk.nextHeight()
	newBulk := makeBulk(start+uint64(count), bulk.blocks[startInd:])

	bulk.blocks = bulk.blocks[:start-bulk.start]
	return l.l.InsertAfter(newBulk, e)
}

func (l *BlocksBulkList) trimBulk(e *list.Element, topHeight uint64) *list.Element {
	bulk := e.Value.(*blocksBulk)
	if bulk.start > topHeight {
		next := e.Next()
		l.l.Remove(e)
		return next
	}

	// top height block should leave
	bulk.blocks = bulk.blocks[0 : topHeight-bulk.start+1]
	return e.Next()
}

type blocksBulk struct {
	start  uint64
	blocks []*WalletBlock
}

func makeBulk(start uint64, blocks []*WalletBlock) *blocksBulk {
	return &blocksBulk{
		start:  start,
		blocks: blocks,
	}
}

func (b *blocksBulk) nextHeight() uint64 {
	return b.start + uint64(len(b.blocks))
}

func (b *blocksBulk) empty() bool {
	return len(b.blocks) == 0
}

// returns height of the last block in bulk, zero if bulk is empty
func (b *blocksBulk) topHeight() uint64 {
	if len(b.blocks) == 0 {
		return 0
	}

	return b.nextHeight() - 1
}
