package server

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/exantech/monero-fastsync/internal/pkg/logging"
	"github.com/exantech/monero-fastsync/internal/pkg/utils"
)

type jobsQueue struct {
	lock             *sync.Mutex
	cond             *sync.Cond
	jobs             []*job
	scanner          Scanner
	db               DbWorker
	stopped          bool
	wg               sync.WaitGroup
	blockchainHeight uint64 // atomic
	topUpdater       *bcHeightUpdater
	workerBlocks     int
	resultBlocks     int
	jobLifetime      time.Duration
}

type job struct {
	wallet           utils.WalletEntry
	startHeight      uint64
	blocks           []WalletBlock
	lock             *sync.Mutex
	cond             *sync.Cond
	err              error
	inProgress       bool
	lastQuery        time.Time
	blockchainHeight uint64
	stopJob          bool
}

func NewJobsQueue(scanner Scanner, db DbWorker, workerBlocks int, resultBlocks int, jobLifetime time.Duration) *jobsQueue {
	jq := &jobsQueue{
		lock:         new(sync.Mutex),
		jobs:         make([]*job, 0, 100),
		scanner:      scanner,
		stopped:      false,
		db:           db,
		workerBlocks: workerBlocks,
		resultBlocks: resultBlocks,
		jobLifetime:  jobLifetime,
	}

	jq.topUpdater = newBcHeightUpdater(&jq.blockchainHeight, jq.db, 30*time.Second)
	jq.cond = sync.NewCond(jq.lock)
	return jq
}

func (q *jobsQueue) StartWorkers(count int) error {
	err := q.topUpdater.updateTopBlockInfo()
	if err != nil {
		logging.Log.Errorf("Failed to start workers, error on updating top block height: %s", err.Error())
		return err
	}

	logging.Log.Debugf("Top block height retrieved: %d", atomic.LoadUint64(&q.blockchainHeight))

	logging.Log.Debugf("Starting %d workers", count)
	for i := 0; i < count; i++ {
		w := &worker{q, q.scanner, q.db, q.workerBlocks}

		go func() {
			q.wg.Add(1)
			defer q.wg.Done()

			w.run()
		}()
	}

	logging.Log.Debugf("Workers started")

	go func() {
		q.wg.Add(1)
		defer q.wg.Done()

		q.topUpdater.runLoop()
	}()

	return nil
}

func (q *jobsQueue) Stop() {
	logging.Log.Info("Stopping updater...")
	q.topUpdater.stop()
	logging.Log.Info("Updater stopped")

	q.lock.Lock()
	q.stopped = true
	q.cond.Broadcast()
	q.lock.Unlock()

	logging.Log.Info("Waiting for workers...")
	q.wg.Wait()
	logging.Log.Info("Workers stopped")
}

func (q *jobsQueue) AddJob(wallet utils.WalletEntry, startHeight uint64) *blocksListener {
	q.lock.Lock()
	defer q.lock.Unlock()

	for _, j := range q.jobs {
		if j.wallet.Keys.SpendPublicKey == wallet.Keys.SpendPublicKey && j.wallet.Keys.ViewSecretKey == wallet.Keys.ViewSecretKey {
			j.updateJob(time.Now(), atomic.LoadUint64(&q.blockchainHeight))
			q.cond.Signal()
			return &blocksListener{j, startHeight, q.resultBlocks}
		}
	}

	newJob := q.addNewJob(wallet, startHeight)
	q.cond.Signal()
	return &blocksListener{newJob, startHeight, q.resultBlocks}
}

// must be locked from outside
func (q *jobsQueue) addNewJob(wallet utils.WalletEntry, startHeight uint64) *job {
	newJob := newJob(wallet, startHeight)
	newJob.setBcHeight(atomic.LoadUint64(&q.blockchainHeight))

	q.jobs = append(q.jobs, newJob)

	return newJob
}

func (q *jobsQueue) waitJob() (*job, bool) {
	q.lock.Lock()
	defer q.lock.Unlock()

	var freeJob *job
	for {
		freeJob = q.findFreeJob()
		if q.stopped || freeJob != nil {
			break
		}

		q.cond.Wait()
	}

	if q.stopped {
		return nil, true
	}

	freeJob.inProgress = true
	return freeJob, false
}

// must be locked from outside
func (q *jobsQueue) findFreeJob() *job {
	for _, j := range q.jobs {
		if !j.inProgress && j.currentHeightLocked() < atomic.LoadUint64(&q.blockchainHeight) && time.Now().Sub(j.lastQuery) < q.jobLifetime {
			return j
		}
	}

	return nil
}

func newJob(wallet utils.WalletEntry, startHeight uint64) *job {
	j := &job{
		wallet:      wallet,
		startHeight: startHeight,
		blocks:      make([]WalletBlock, 0, 5000),
		lock:        new(sync.Mutex),
		inProgress:  false,
		lastQuery:   time.Now(),
	}

	j.cond = sync.NewCond(j.lock)
	return j
}

type blocksListener struct {
	job        *job
	returnFrom uint64
	maxBlocks  int
}

func (l *blocksListener) Wait() ([]WalletBlock, error) {
	return l.job.waitBlocks(l.returnFrom, l.maxBlocks)
}

type worker struct {
	queue     *jobsQueue
	scanner   Scanner
	db        DbWorker
	maxBlocks int
}

func (w *worker) run() {
	for {
		job, stop := w.queue.waitJob()
		if stop {
			return
		}

		top, err := w.db.GetTopScannedHeightInfo(job.wallet.Id)
		if err != nil {
			job.setError(err) //TODO: turn error off after use!
			return
		}

		// in case if chain split occurred we trim top detached blocks
		job.trimHeight(top.Height)

		job.wallet.ScannedHeight = top.Height

		blocks, err := w.scanner.GetBlocks(job.nextHeight(), job.wallet, w.maxBlocks)
		if err != nil {
			job.setError(err) //TODO: turn error off after use!
			return
		}

		job.setBlocks(blocks)
	}
}

func (j *job) waitBlocks(from uint64, maxCount int) ([]WalletBlock, error) {
	if from < j.startHeight {
		logging.Log.Warningf("A job called to wait blocks from %d while it's start height %d. "+
			"This situation is not handled", from, j.startHeight)

		return nil, errors.New("not implemented")
	}

	j.lock.Lock()
	defer j.lock.Unlock()

	// There is the situation when wallet's blocks are not fully scanned till blockchain height
	// and a wallet can get in response just 1 block, which means end of synchronization.
	// Therefore unless we scanned blocks till blockchain height we have to send at least 2 blocks to a wallet.
	minCount := 2
	if j.currentHeightLocked() == j.blockchainHeight {
		minCount = 1
	}
	minSize := int(from - j.startHeight + uint64(minCount))

	for len(j.blocks) < minSize && j.err == nil {
		j.cond.Wait()
	}

	if j.err != nil {
		return nil, j.err
	}

	count := maxCount
	if len(j.blocks)-minSize+1 < count {
		count = len(j.blocks) - minSize + 1
	}

	return j.blocks[minSize-minCount : minSize+count-1], nil
}

func (j *job) trimHeight(height uint64) {
	j.lock.Lock()
	defer j.lock.Unlock()

	if height < j.startHeight {
		j.blocks = make([]WalletBlock, 0, 5000)
		j.startHeight = height
		return
	}

	cachedHeight := j.startHeight + uint64(len(j.blocks)) - 1
	if cachedHeight > height {
		j.blocks = j.blocks[0 : cachedHeight-j.startHeight+1]
	}
}

func (j *job) nextHeight() uint64 {
	j.lock.Lock()
	defer j.lock.Unlock()

	return j.startHeight + uint64(len(j.blocks))
}

func (j *job) currentHeight() uint64 {
	j.lock.Lock()
	defer j.lock.Unlock()

	return j.currentHeightLocked()
}

func (j *job) currentHeightLocked() uint64 {
	h := j.startHeight + uint64(len(j.blocks))
	if h == 0 {
		return h
	}

	return h - 1
}

func (j *job) setError(err error) {
	j.lock.Lock()
	defer j.lock.Unlock()

	j.inProgress = false // XXX: the function assumed to be called just before releasing the job
	j.err = err
	j.cond.Broadcast()
}

func (j *job) setBlocks(blocks []WalletBlock) {
	j.lock.Lock()
	defer j.lock.Unlock()

	j.inProgress = false // XXX: the function assumed to be called just before releasing the job
	j.blocks = append(j.blocks, blocks...)
	j.cond.Broadcast()
}

func (j *job) setBcHeight(bcHeight uint64) {
	j.lock.Lock()
	defer j.lock.Unlock()

	j.blockchainHeight = bcHeight
}

func (j *job) updateJob(lastQuery time.Time, bcHeight uint64) {
	j.lock.Lock()
	defer j.lock.Unlock()

	j.lastQuery = lastQuery
	j.blockchainHeight = bcHeight
}

type bcHeightUpdater struct {
	topHeight *uint64
	db        DbWorker
	interval  time.Duration
	stopCh    chan struct{}
}

func newBcHeightUpdater(topHeight *uint64, db DbWorker, interval time.Duration) *bcHeightUpdater {
	return &bcHeightUpdater{
		topHeight: topHeight,
		db:        db,
		interval:  interval,
		stopCh:    make(chan struct{}),
	}
}

func (u *bcHeightUpdater) runLoop() {
	ticker := time.Tick(u.interval)
	for {
		select {
		case <-ticker:
			u.updateTopBlockInfo()
		case <-u.stopCh:
			logging.Log.Debug("Stop signal received, stopping top block update loop")
			return
		}
	}
}

func (u *bcHeightUpdater) stop() {
	u.stopCh <- struct{}{}
}

func (u *bcHeightUpdater) updateTopBlockInfo() error {
	height, err := u.db.GetTopBlockHeight()
	if err != nil {
		logging.Log.Errorf("Failed to get top block height: %s", err.Error())
		return err
	}

	atomic.StoreUint64(u.topHeight, height)
	return nil
}
