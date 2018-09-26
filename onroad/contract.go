package onroad

import (
	"container/heap"
	"github.com/pkg/errors"
	"github.com/vitelabs/go-vite/common/types"
	"github.com/vitelabs/go-vite/log15"
	"github.com/vitelabs/go-vite/producer"
	"github.com/vitelabs/go-vite/onroad/model"
	"github.com/vitelabs/go-vite/verifier"
	"sync"
)

//
type ContractWorker struct {
	manager *Manager

	uBlocksPool *model.OnroadBlocksPool
	verifier    *verifier.AccountVerifier

	gid                 types.Gid
	address             types.Address
	accEvent            producer.AccountStartEvent
	contractAddressList []*types.Address

	status      int
	statusMutex sync.Mutex

	// todo use sync.Cond
	isSleep                bool
	newOnroadTxAlarm       chan struct{}
	breaker                chan struct{}
	stopDispatcherListener chan struct{}

	contractTasks []*ContractTaskProcessor

	priorityToQueue      *model.PriorityToQueue
	priorityToQueueMutex sync.RWMutex

	blackList      map[types.Hash]bool // map[Hash(from, to)]bool
	blackListMutex sync.RWMutex

	lastAddrIndex int

	log log15.Logger
}

func NewContractWorker(manager *Manager, accEvent producer.AccountStartEvent) (*ContractWorker, error) {

	addressList, err := manager.onroadBlocksPool.GetAddrListByGid(accEvent.Gid)
	if err != nil {
		return nil, err
	}
	if len(addressList) <= 0 {
		return nil, errors.New("newContractWorker addressList nil")
	}

	return &ContractWorker{
		manager:                manager,
		uBlocksPool:            manager.onroadBlocksPool,
		verifier:               verifier.NewAccountVerifier(nil, nil, nil), // todo
		gid:                    accEvent.Gid,
		address:                accEvent.Address,
		accEvent:               accEvent,
		contractAddressList:    addressList,
		status:                 Create,
		isSleep:                false,
		newOnroadTxAlarm:       make(chan struct{}),
		breaker:                make(chan struct{}),
		stopDispatcherListener: make(chan struct{}),
		contractTasks:          make([]*ContractTaskProcessor, CONTRACT_TASK_SIZE),
		blackList:              make(map[types.Hash]bool),
		log:                    log15.New("worker", "c", "addr", accEvent.Address, "gid", accEvent.Gid),
	}, nil

}

func (w *ContractWorker) Start() {
	w.log.Info("Start()", "current status", w.status)
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status != Start {

		w.uBlocksPool.AddContractLis(w.gid, func() {
			w.NewOnroadTxAlarm()
		})

		for i, v := range w.contractTasks {
			v = NewContractTaskProcessor(w, i, w.dispatchTask)
			v.Start()
		}

		go w.waitingNewBlock()

		w.status = Start
	} else {
		// awake it in order to run at least once
		w.NewOnroadTxAlarm()
	}
	w.log.Info("end start")
}

func (w *ContractWorker) Stop() {
	w.log.Info("Stop()", "current status", w.status)
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	if w.status != Stop {

		w.breaker <- struct{}{}

		w.uBlocksPool.RemoveContractLis(w.gid)
		w.isSleep = true
		close(w.newOnroadTxAlarm)

		<-w.stopDispatcherListener
		close(w.stopDispatcherListener)

		w.log.Info("stop all task")
		wg := new(sync.WaitGroup)
		for _, v := range w.contractTasks {
			wg.Add(1)
			go func() {
				v.Stop()
				wg.Done()
			}()
		}
		wg.Wait()
		w.log.Info("all task stopped")
		w.status = Stop
	}
	w.log.Info("stopped")
}

func (w *ContractWorker) Close() error {
	w.Stop()
	return nil
}

func (w ContractWorker) Status() int {
	w.statusMutex.Lock()
	defer w.statusMutex.Unlock()
	return w.status
}

func (w *ContractWorker) dispatchTask(index int) *model.FromItem {
	//w.log.Info("dispatchTask", "index", index)
	w.priorityToQueueMutex.Lock()
	defer w.priorityToQueueMutex.Unlock()

	if w.priorityToQueue.Len() == 0 {
		w.log.Info("priorityToQueue empty now get from db")
		for !w.FetchNewFromDb() {
			if w.priorityToQueue.Len() != 0 {
				break
			}
		}
	}

	if w.priorityToQueue.Len() == 0 {
		w.log.Info("priorityToQueue empty cache and db")
		return nil
	}

	tItem := heap.Pop(w.priorityToQueue).(*model.ToItem)
	priorityFromQueue := tItem.Value
	for j := 0; j < priorityFromQueue.Len(); j++ {
		fItem := heap.Pop(priorityFromQueue).(*model.FromItem)
		return fItem
	}
	return nil
}

func (w *ContractWorker) NewOnroadTxAlarm() {
	if w.isSleep {
		w.newOnroadTxAlarm <- struct{}{}
	}
}

func (w *ContractWorker) waitingNewBlock() {
	w.log.Info("waitingNewBlock")
LOOP:
	for {
		w.isSleep = false
		if w.Status() == Stop {
			break
		}

		if w.priorityToQueue.Len() != 0 {
			for _, v := range w.contractTasks {
				v.WakeUp()
				// todo mutex
				if w.priorityToQueue.Len() == 0 {
					break
				}
			}
		}

		w.isSleep = true
		select {
		case <-w.newOnroadTxAlarm:
			w.log.Info("start awake")
		case <-w.breaker:
			w.log.Info("worker broken")
			break LOOP
		}
	}

	w.log.Info("waitingNewBlock end called")
	w.stopDispatcherListener <- struct{}{}
	w.log.Info("waitingNewBlock end")
}

// true means traverse all contract address
func (w *ContractWorker) FetchNewFromDb() bool {
	snapshotHash := w.accEvent.SnapshotHash
	i := w.lastAddrIndex
	// fetch CONTRACT_TASK_SIZE contract address at most
	for ; i < len(w.contractAddressList) && (i-w.lastAddrIndex) < CONTRACT_TASK_SIZE; i++ {
		count := 0
		nextIndex := 0
		for {
			blockList, err := w.uBlocksPool.GetOnroadBlocks(uint64(nextIndex), 1, uint64(CONTRACT_FETCH_SIZE), w.contractAddressList[i])
			if blockList == nil {
				break
			}
			if err != nil {
				w.log.Error("FetchNewFromDb.GetOnroadBlocks", "error", err)
				break
			}
			for _, v := range blockList {
				if !w.isInBlackList(v.AccountAddress, v.ToAddress) {
					fromQuota := w.manager.uAccess.GetAccountQuota(v.AccountAddress, snapshotHash)
					toQuota := w.manager.uAccess.GetAccountQuota(v.ToAddress, snapshotHash)
					w.priorityToQueue.InsertNew(v, toQuota, fromQuota)
					count++
				}
			}
			if count < CONTRACT_FETCH_SIZE {
				nextIndex++
			} else {
				break
			}
		}
	}
	w.lastAddrIndex = i
	if w.lastAddrIndex >= len(w.contractAddressList) {
		w.lastAddrIndex = 0
		return true
	}
	return false
}

// fixme 把from去掉
func (w *ContractWorker) addIntoBlackList(from types.Address, to types.Address) {
	w.log.Info("addIntoBlackList", "from", from, "to", to)
	key := types.DataListHash(from.Bytes(), to.Bytes())
	w.blackListMutex.Lock()
	defer w.blackListMutex.Unlock()
	w.blackList[key] = true
}

func (w *ContractWorker) deleteBlackListItem(from types.Address, to types.Address) {
	w.log.Info("deleteBlackListItem", "from", from, "to", to)
	key := types.DataListHash(from.Bytes(), to.Bytes())
	w.blackListMutex.Lock()
	defer w.blackListMutex.Unlock()
	delete(w.blackList, key)
}

func (w *ContractWorker) isInBlackList(from types.Address, to types.Address) bool {
	key := types.DataListHash(from.Bytes(), to.Bytes())
	w.blackListMutex.RLock()
	defer w.blackListMutex.RUnlock()
	_, ok := w.blackList[key]
	if ok {
		w.log.Info("isInBlackList", "from", from, "to", to, "in", ok)
	}
	return ok
}