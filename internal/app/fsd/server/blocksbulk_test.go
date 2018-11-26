package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func genTestBlocks(startHeight uint64, count int) []*WalletBlock {
	blocks := make([]*WalletBlock, count)
	for i := 0; i < count; i++ {
		blocks[i] = &WalletBlock{
			Timestamp: startHeight + uint64(i),
		}
	}

	return blocks
}

func getTestBulk(startHeight uint64, count int) *blocksBulk {
	return &blocksBulk{
		start:  startHeight,
		blocks: genTestBlocks(startHeight, count),
	}
}

func TestBlocksBulkInsert(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 10))

	assert.Equal(t, 1, bbl.l.Len())
	bb := bbl.l.Front().Value.(*blocksBulk)

	assert.Equal(t, getTestBulk(10, 10), bb)
}

func TestBlocksBulkInsertIntersectEnd(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 10))
	bbl.AddBlocks(19, genTestBlocks(19, 2))

	assert.Equal(t, 1, bbl.l.Len())
	bb := bbl.l.Front().Value.(*blocksBulk)

	assert.Equal(t, getTestBulk(10, 11), bb)
}

func TestBlocksBulkInsertIntersectStart(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 10))
	bbl.AddBlocks(8, genTestBlocks(8, 10))

	assert.Equal(t, 1, bbl.l.Len())
	e := bbl.l.Front()

	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(8, 12), bb)
}

func TestBlocksBulkInsertInside(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 10))
	bbl.AddBlocks(12, genTestBlocks(12, 2))

	assert.Equal(t, 1, bbl.l.Len())
	e := bbl.l.Front()

	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(10, 10), bb)
}

func TestBlocksBulkInsertSame(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 10))
	bbl.AddBlocks(10, genTestBlocks(10, 10))

	assert.Equal(t, 1, bbl.l.Len())
	e := bbl.l.Front()

	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(10, 10), bb)
}

func TestBlocksBulkInsertTwo(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 10))
	bbl.AddBlocks(30, genTestBlocks(30, 10))

	assert.Equal(t, 2, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(10, 10), bb)

	e = e.Next()
	bb = e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(30, 10), bb)
}

func TestBlocksBulkInsertThree(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(30, genTestBlocks(30, 7))
	bbl.AddBlocks(10, genTestBlocks(10, 5))
	bbl.AddBlocks(20, genTestBlocks(20, 6))

	assert.Equal(t, 3, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(10, 5), bb)

	e = e.Next()
	bb = e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(20, 6), bb)

	e = e.Next()
	bb = e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(30, 7), bb)
}

func TestBlocksBulkInsertConsequentiallyEnd(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 10))
	bbl.AddBlocks(20, genTestBlocks(20, 6))

	assert.Equal(t, 1, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(10, 16), bb)
}

func TestBlocksBulkInsertConsequentiallyStart(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(20, genTestBlocks(20, 6))
	bbl.AddBlocks(10, genTestBlocks(10, 10))

	assert.Equal(t, 1, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(10, 16), bb)
}

func TestBlocksBulkInsertOuter(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 10))
	bbl.AddBlocks(5, genTestBlocks(5, 40))

	assert.Equal(t, 1, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(5, 40), bb)
}

func TestBlocksBulkInsertBigBetweenTwo(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 10))
	bbl.AddBlocks(30, genTestBlocks(30, 10))
	bbl.AddBlocks(15, genTestBlocks(15, 20))

	assert.Equal(t, 1, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(10, 30), bb)
}

func TestBlocksBulkInsertBigMergeTwoFromLeft(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 5))
	bbl.AddBlocks(20, genTestBlocks(20, 5))
	bbl.AddBlocks(5, genTestBlocks(5, 25))

	assert.Equal(t, 1, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(5, 25), bb)
}

func TestBlocksBulkInsertMergeTwoBoundary(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 5))
	bbl.AddBlocks(20, genTestBlocks(20, 5))
	bbl.AddBlocks(5, genTestBlocks(5, 15))

	assert.Equal(t, 1, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(5, 20), bb)
}

func TestBlocksBulkInsertOnEmpty(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 0))
	bbl.AddBlocks(10, genTestBlocks(10, 10))

	assert.Equal(t, 1, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(10, 10), bb)
}

func TestBlocksBulkInsertAfterEmpty(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 0))
	bbl.AddBlocks(11, genTestBlocks(11, 10))

	assert.Equal(t, 2, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(10, 0), bb)

	e = e.Next()
	bb = e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(11, 10), bb)
}

func TestBlocksBulkInsertBeforeEmpty(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 0))
	bbl.AddBlocks(9, genTestBlocks(9, 1))

	assert.Equal(t, 1, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(9, 1), bb)
}

func TestBlocksBulkInsertOnEmptyLeft(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 0))
	bbl.AddBlocks(5, genTestBlocks(5, 10))

	assert.Equal(t, 1, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(5, 10), bb)
}

func TestBlocksBulkInsertTwoEmpty(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 0))
	bbl.AddBlocks(9, genTestBlocks(9, 0))

	assert.Equal(t, 2, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(9, 0), bb)

	e = e.Next()
	bb = e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(10, 0), bb)
}

func TestBlocksBulkInsertMergeThreeBulks(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 10))
	bbl.AddBlocks(30, genTestBlocks(30, 10))
	bbl.AddBlocks(20, genTestBlocks(20, 10))

	assert.Equal(t, 1, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(10, 30), bb)
}

func TestBlocksBulkInsertMergeFiveBulks(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(50, genTestBlocks(50, 10))
	bbl.AddBlocks(10, genTestBlocks(10, 10))
	bbl.AddBlocks(30, genTestBlocks(30, 10))

	bbl.AddBlocks(40, genTestBlocks(40, 10))
	bbl.AddBlocks(20, genTestBlocks(20, 10))

	assert.Equal(t, 1, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(10, 50), bb)
}

func TestBlocksBulkRemoveExact(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(50, genTestBlocks(50, 10))
	bbl.RemoveBlocks(50, 10)

	assert.Equal(t, 0, bbl.l.Len())
}

func TestBlocksBulkRemoveBigger(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(50, genTestBlocks(50, 10))
	bbl.RemoveBlocks(45, 20)

	assert.Equal(t, 0, bbl.l.Len())
}

func TestBlocksBulkRemoveNothingLeft(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(50, genTestBlocks(50, 10))
	bbl.RemoveBlocks(45, 5)

	assert.Equal(t, 1, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(50, 10), bb)
}

func TestBlocksBulkRemoveNothingRight(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(50, genTestBlocks(50, 10))
	bbl.RemoveBlocks(60, 5)

	assert.Equal(t, 1, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(50, 10), bb)
}

func TestBlocksBulkRemoveLeft(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(50, genTestBlocks(50, 10))
	bbl.RemoveBlocks(45, 10)

	assert.Equal(t, 1, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(55, 5), bb)
}

func TestBlocksBulkRemoveRight(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(50, genTestBlocks(50, 10))
	bbl.RemoveBlocks(55, 10)

	assert.Equal(t, 1, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(50, 5), bb)
}

func TestBlocksBulkRemoveMiddle(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(50, genTestBlocks(50, 10))
	bbl.RemoveBlocks(52, 5)

	assert.Equal(t, 2, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(50, 2), bb)

	e = e.Next()
	bb = e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(57, 3), bb)
}

func TestBlocksBulkRemoveMiddleTwo(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 20))
	bbl.RemoveBlocks(13, 5)
	bbl.RemoveBlocks(20, 5)

	assert.Equal(t, 3, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(10, 3), bb)

	e = e.Next()
	bb = e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(18, 2), bb)

	e = e.Next()
	bb = e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(25, 5), bb)
}

func TestBlocksBulkRemoveIntersecting(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 10))
	bbl.AddBlocks(25, genTestBlocks(25, 10))

	bbl.RemoveBlocks(15, 15)

	assert.Equal(t, 2, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(10, 5), bb)

	e = e.Next()
	bb = e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(30, 5), bb)
}

func TestBlocksBulkRemoveAcrossThree(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 10))
	bbl.AddBlocks(25, genTestBlocks(25, 10))
	bbl.AddBlocks(40, genTestBlocks(40, 10))

	bbl.RemoveBlocks(15, 30)

	assert.Equal(t, 2, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(10, 5), bb)

	e = e.Next()
	bb = e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(45, 5), bb)
}

func TestBlocksBulkTrimOneBulk(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 10))
	bbl.TrimBlocks(9)

	assert.Equal(t, 0, bbl.l.Len())
}

func TestBlocksBulkTrimTwoBulks(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 10))
	bbl.AddBlocks(25, genTestBlocks(25, 10))
	bbl.TrimBlocks(9)

	assert.Equal(t, 0, bbl.l.Len())
}

func TestBlocksBulkTrimBulkInMiddle(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 10))
	bbl.TrimBlocks(15)

	assert.Equal(t, 1, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(10, 6), bb)
}

func TestBlocksBulkTrimMultipleBulksInMiddle(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 10))
	bbl.AddBlocks(10, genTestBlocks(30, 10))
	bbl.TrimBlocks(15)

	assert.Equal(t, 1, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(10, 6), bb)
}

func TestBlocksBulkTrimStartHeight(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 10))
	bbl.TrimBlocks(10)

	assert.Equal(t, 1, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(10, 1), bb)
}

func TestBlocksBulkTrimEmpty(t *testing.T) {
	bbl := NewBlocksBulkList()

	bbl.AddBlocks(10, genTestBlocks(10, 0))
	bbl.TrimBlocks(10)

	assert.Equal(t, 1, bbl.l.Len())

	e := bbl.l.Front()
	bb := e.Value.(*blocksBulk)
	assert.Equal(t, getTestBulk(10, 0), bb)
}

func TestBlocksBulkBlocksAvailableEmpty(t *testing.T) {
	bbl := NewBlocksBulkList()
	assert.Equal(t, 0, bbl.BlocksAvailable(0))
}

func TestBlocksBulkBlocksAvailableBefore(t *testing.T) {
	bbl := NewBlocksBulkList()
	bbl.AddBlocks(10, genTestBlocks(10, 10))

	assert.Equal(t, 0, bbl.BlocksAvailable(9))
}

func TestBlocksBulkBlocksAvailableEmptyBulk(t *testing.T) {
	bbl := NewBlocksBulkList()
	bbl.AddBlocks(10, genTestBlocks(10, 0))

	assert.Equal(t, 0, bbl.BlocksAvailable(10))
}

func TestBlocksBulkBlocksAvailableStartBulk(t *testing.T) {
	bbl := NewBlocksBulkList()
	bbl.AddBlocks(10, genTestBlocks(10, 10))

	assert.Equal(t, 10, bbl.BlocksAvailable(10))
}

func TestBlocksBulkBlocksAvailableMiddleBulk(t *testing.T) {
	bbl := NewBlocksBulkList()
	bbl.AddBlocks(10, genTestBlocks(10, 10))

	assert.Equal(t, 5, bbl.BlocksAvailable(15))
}

func TestBlocksBulkBlocksAvailableNextHeight(t *testing.T) {
	bbl := NewBlocksBulkList()
	bbl.AddBlocks(10, genTestBlocks(10, 10))

	assert.Equal(t, 0, bbl.BlocksAvailable(20))
}

func TestBlocksBulkBlocksAvailableBetweenBulks(t *testing.T) {
	bbl := NewBlocksBulkList()
	bbl.AddBlocks(10, genTestBlocks(10, 10))
	bbl.AddBlocks(10, genTestBlocks(30, 10))

	assert.Equal(t, 0, bbl.BlocksAvailable(20))
}

func TestBlocksBulkGetBlocksEmpty(t *testing.T) {
	bbl := NewBlocksBulkList()
	blocks := bbl.GetBlocks(10, 10)

	assert.Equal(t, 0, len(blocks))
}

func TestBlocksBulkGetBlocksBefore(t *testing.T) {
	bbl := NewBlocksBulkList()
	bbl.AddBlocks(10, genTestBlocks(10, 10))

	blocks := bbl.GetBlocks(5, 2)
	assert.Equal(t, 0, len(blocks))
}

func TestBlocksBulkGetBlocksLeftIntersect(t *testing.T) {
	bbl := NewBlocksBulkList()
	bbl.AddBlocks(10, genTestBlocks(10, 10))

	blocks := bbl.GetBlocks(5, 10)
	assert.Nil(t, blocks)
}

func TestBlocksBulkGetBlocksAfter(t *testing.T) {
	bbl := NewBlocksBulkList()
	bbl.AddBlocks(10, genTestBlocks(10, 10))

	blocks := bbl.GetBlocks(20, 10)
	assert.Equal(t, 0, len(blocks))
}

func TestBlocksBulkGetBlocksRightIntersect(t *testing.T) {
	bbl := NewBlocksBulkList()
	bbl.AddBlocks(10, genTestBlocks(10, 10))

	blocks := bbl.GetBlocks(15, 10)
	assert.Equal(t, genTestBlocks(15, 5), blocks)
}

func TestBlocksBulkGetBlocksExact(t *testing.T) {
	bbl := NewBlocksBulkList()
	bbl.AddBlocks(10, genTestBlocks(10, 10))

	blocks := bbl.GetBlocks(10, 10)
	assert.Equal(t, genTestBlocks(10, 10), blocks)
}

func TestBlocksBulkGetBlocksInside(t *testing.T) {
	bbl := NewBlocksBulkList()
	bbl.AddBlocks(10, genTestBlocks(10, 10))

	blocks := bbl.GetBlocks(12, 2)
	assert.Equal(t, genTestBlocks(12, 2), blocks)
}

func TestBlocksBulkGetBlocksZero(t *testing.T) {
	bbl := NewBlocksBulkList()
	bbl.AddBlocks(10, genTestBlocks(10, 10))

	blocks := bbl.GetBlocks(12, 0)
	assert.Equal(t, 0, len(blocks))
}

func TestBlocksBulkFindFreeSpaceEmpty(t *testing.T) {
	bbl := NewBlocksBulkList()
	start, count := bbl.FindMissingBlocks()
	assert.Equal(t, uint64(0), start)
	assert.Equal(t, 0, count)
}

func TestBlocksBulkFindFreeSpaceOneBulk(t *testing.T) {
	bbl := NewBlocksBulkList()
	bbl.AddBlocks(10, genTestBlocks(10, 10))

	start, count := bbl.FindMissingBlocks()
	assert.Equal(t, uint64(20), start)
	assert.Equal(t, 0, count)
}

func TestBlocksBulkFindFreeSpaceOneEmptyBulk(t *testing.T) {
	bbl := NewBlocksBulkList()
	bbl.AddBlocks(10, genTestBlocks(10, 0))

	start, count := bbl.FindMissingBlocks()
	assert.Equal(t, uint64(10), start)
	assert.Equal(t, 0, count)
}

func TestBlocksBulkFindFreeSpaceTwoBulks(t *testing.T) {
	bbl := NewBlocksBulkList()
	bbl.AddBlocks(10, genTestBlocks(10, 10))
	bbl.AddBlocks(30, genTestBlocks(30, 10))

	start, count := bbl.FindMissingBlocks()
	assert.Equal(t, uint64(20), start)
	assert.Equal(t, 10, count)
}

func TestBlocksBulkFindFreeSpaceEmptyBetween(t *testing.T) {
	bbl := NewBlocksBulkList()
	bbl.AddBlocks(10, genTestBlocks(10, 10))
	bbl.AddBlocks(25, genTestBlocks(25, 0))
	bbl.AddBlocks(30, genTestBlocks(30, 10))

	start, count := bbl.FindMissingBlocks()
	assert.Equal(t, uint64(20), start)
	assert.Equal(t, 10, count)
}

func TestBlocksBulkFindFreeSpaceMultipleEmptyBetween(t *testing.T) {
	bbl := NewBlocksBulkList()
	bbl.AddBlocks(10, genTestBlocks(10, 10))
	bbl.AddBlocks(23, genTestBlocks(23, 0))
	bbl.AddBlocks(26, genTestBlocks(26, 0))
	bbl.AddBlocks(30, genTestBlocks(30, 10))

	start, count := bbl.FindMissingBlocks()
	assert.Equal(t, uint64(20), start)
	assert.Equal(t, 10, count)
}

func TestBlocksBulkFindFreeSpaceMultipleEmpty(t *testing.T) {
	bbl := NewBlocksBulkList()
	bbl.AddBlocks(23, genTestBlocks(23, 0))
	bbl.AddBlocks(26, genTestBlocks(26, 0))

	start, count := bbl.FindMissingBlocks()
	assert.Equal(t, uint64(23), start)
	assert.Equal(t, 0, count)
}

func TestBlocksBulkFindFreeSpaceEmptyBefore(t *testing.T) {
	bbl := NewBlocksBulkList()
	bbl.AddBlocks(23, genTestBlocks(23, 0))
	bbl.AddBlocks(26, genTestBlocks(26, 10))

	start, count := bbl.FindMissingBlocks()
	assert.Equal(t, uint64(23), start)
	assert.Equal(t, 3, count)
}
