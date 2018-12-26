package server

import (
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
	jj               *jobJanitor
}

type job struct {
	wallet           utils.WalletEntry
	blocks           *BlocksBulkList
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

	jq.jj = newJobJanitor(jq, jobLifetime)
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

	go func() {
		q.wg.Add(1)
		defer q.wg.Done()

		q.jj.runLoop()
	}()

	return nil
}

func (q *jobsQueue) Stop() {
	logging.Log.Info("Stopping updater...")
	q.topUpdater.stop()
	logging.Log.Info("Updater stopped")

	logging.Log.Info("Stopping job janitor...")
	q.jj.stop()
	logging.Log.Info("Job janitor stopped")

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

	topHeight := atomic.LoadUint64(&q.blockchainHeight)
	for _, j := range q.jobs {
		if j.wallet.Keys.SpendPublicKey == wallet.Keys.SpendPublicKey && j.wallet.Keys.ViewSecretKey == wallet.Keys.ViewSecretKey {
			j.updateJob(time.Now(), topHeight, startHeight)
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

func (q *jobsQueue) jobDone(job *job) {
	q.lock.Lock()
	defer q.lock.Unlock()

	i := 0
	for _, j := range q.jobs {
		if j == job {
			break
		}

		i++
	}

	// move job to the end of the queue
	q.jobs = append(q.jobs[0:i], q.jobs[i+1:]...)
	q.jobs = append(q.jobs, job)

	job.lock.Lock()
	job.inProgress = false
	job.lock.Unlock()

	q.cond.Signal()
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
		bcHeight := atomic.LoadUint64(&q.blockchainHeight)
		nextBlock, _ := j.FindMissingBlocks()

		synced := j.BlocksAvailable(bcHeight) != 0 && nextBlock >= bcHeight

		if !j.inProgress && !synced && time.Now().Sub(j.lastQuery) < q.jobLifetime {
			return j
		}
	}

	return nil
}

func newJob(wallet utils.WalletEntry, startHeight uint64) *job {
	j := &job{
		wallet:     wallet,
		blocks:     NewBlocksBulkList(),
		lock:       new(sync.Mutex),
		inProgress: false,
		lastQuery:  time.Now(),
	}

	j.blocks.AddBlocks(startHeight, []*WalletBlock{})

	j.cond = sync.NewCond(j.lock)
	return j
}

type blocksListener struct {
	job        *job
	returnFrom uint64
	maxBlocks  int
}

func (l *blocksListener) Wait() ([]*WalletBlock, error) {
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

		w.processJob(job)
		w.queue.jobDone(job)
	}
}

func (w *worker) processJob(job *job) {
	top, err := w.db.GetTopScannedHeightInfo(job.wallet.Id)
	if err != nil {
		job.setError(err) //TODO: turn error off after use!
		return
	}

	// in case if chain split occurred we trim top detached blocks
	job.trimHeight(top.Height)

	job.wallet.ScannedHeight = top.Height

	start, count := job.FindMissingBlocks()
	if count > w.maxBlocks || count == 0 {
		count = w.maxBlocks
	}

	blocks, err := w.scanner.GetBlocks(start, job.wallet, count)
	if err != nil {
		job.setError(err) //TODO: turn error off after use!
		return
	}

	job.setBlocks(start, blocks)
}

func (j *job) waitBlocks(from uint64, maxCount int) ([]*WalletBlock, error) {
	j.lock.Lock()
	defer j.lock.Unlock()

	// There is the situation when wallet's blocks are not fully scanned till blockchain height
	// and a wallet can get in response just 1 block, which means end of synchronization.
	// Therefore unless we scanned blocks till blockchain height we have to send at least 2 blocks to a wallet.
	minCount := 4
	next, _ := j.blocks.FindMissingBlocks()
	if next >= j.blockchainHeight {
		minCount = 1
	}

	for j.blocks.BlocksAvailable(from) < minCount && j.err == nil {
		j.cond.Wait()
	}

	if j.err != nil {
		err := j.err
		j.err = nil
		return nil, err
	}

	return j.blocks.GetBlocks(from, maxCount), nil
}

func (j *job) trimHeight(height uint64) {
	j.lock.Lock()
	defer j.lock.Unlock()

	j.blocks.TrimBlocks(height)
}

func (j *job) FindMissingBlocks() (uint64, int) {
	j.lock.Lock()
	defer j.lock.Unlock()

	return j.blocks.FindMissingBlocks()
}

func (j *job) BlocksAvailable(start uint64) int {
	j.lock.Lock()
	defer j.lock.Unlock()

	return j.blocks.BlocksAvailable(start)
}

func (j *job) setError(err error) {
	j.lock.Lock()
	defer j.lock.Unlock()

	j.err = err
	j.cond.Broadcast()
}

func (j *job) setBlocks(start uint64, blocks []*WalletBlock) {
	j.lock.Lock()
	defer j.lock.Unlock()

	j.blocks.AddBlocks(start, blocks)
	j.cond.Broadcast()
}

func (j *job) setBcHeight(bcHeight uint64) {
	j.lock.Lock()
	defer j.lock.Unlock()

	j.blockchainHeight = bcHeight
}

func (j *job) updateJob(lastQuery time.Time, bcHeight uint64, startHeight uint64) {
	j.lock.Lock()
	defer j.lock.Unlock()

	j.lastQuery = lastQuery
	j.blockchainHeight = bcHeight
	j.blocks.AddBlocks(startHeight, []*WalletBlock{})
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

type jobJanitor struct {
	jq          *jobsQueue
	stopCh      chan struct{}
	jobLifetime time.Duration
	interval    time.Duration
}

func newJobJanitor(jq *jobsQueue, jobLifetime time.Duration) *jobJanitor {
	return &jobJanitor{
		jq:          jq,
		jobLifetime: jobLifetime,
		interval:    jobLifetime,
		stopCh:      make(chan struct{}),
	}
}

func (jj *jobJanitor) runLoop() {
	ticker := time.Tick(jj.interval)
	for {
		select {
		case <-ticker:
			jj.clean()
		case <-jj.stopCh:
			logging.Log.Debug("Stop signal received, stopping job janitor loop")
			return
		}
	}
}

func (jj *jobJanitor) stop() {
	jj.stopCh <- struct{}{}
}

func (jj *jobJanitor) clean() {
	jj.jq.lock.Lock()
	defer jj.jq.lock.Unlock()

	fresh := make([]*job, 0, len(jj.jq.jobs))
	for _, j := range jj.jq.jobs {
		if time.Now().Sub(j.lastQuery) < jj.jobLifetime {
			fresh = append(fresh, j)
		}
	}

	jj.jq.jobs = fresh
}
