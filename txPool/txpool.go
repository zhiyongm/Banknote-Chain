package txPool

import (
	"blockcooker/core"
	"sync"
	"time"
	"unsafe"
)

type TxPool struct {
	TxQueue  []*core.Transaction // transaction Queue
	lock     sync.Mutex
	capacity int        // 池的最大容量
	cond     *sync.Cond // 用于阻塞和唤醒的条件变量
}

// NewTxPool 创建一个新的交易池并指定容量
func NewTxPool(capacity int) *TxPool {
	if capacity <= 0 {
		// 如果容量无效，可以设置一个默认值或引发 panic
		// 这里我们选择 panic，因为一个无容量的池没有意义
		panic("txpool capacity must be positive")
	}
	pool := &TxPool{
		// 为了效率，预先分配切片的容量
		TxQueue:  make([]*core.Transaction, 0, capacity),
		capacity: capacity,
	}
	// 初始化条件变量，它需要与一个锁关联
	pool.cond = sync.NewCond(&pool.lock)
	return pool
}

// Add a transaction to the pool. It will block if the pool is full.
// 向交易池中添加一个交易，如果池已满，则会阻塞
func (txpool *TxPool) AddTx2Pool(tx *core.Transaction) {
	txpool.lock.Lock()
	defer txpool.lock.Unlock()

	// 使用 for 循环检查条件。这是官方推荐的做法，可以防止“虚假唤醒”
	for len(txpool.TxQueue) >= txpool.capacity {
		// 池已满，调用 Wait()。
		// Wait() 会自动释放锁，并让当前协程休眠。
		// 当被唤醒时，它会重新获取锁，然后循环会再次检查条件。
		txpool.cond.Wait()
	}

	if tx.Time.IsZero() {
		tx.Time = time.Now()
	}
	txpool.TxQueue = append(txpool.TxQueue, tx)
}

// Add a list of transactions to the pool. It will block if the pool is full.
// 向交易池中添加一组交易，如果池已满，则会阻塞
func (txpool *TxPool) AddTxs2Pool(txs []*core.Transaction) {
	txpool.lock.Lock()
	defer txpool.lock.Unlock()

	for _, tx := range txs {
		for len(txpool.TxQueue) >= txpool.capacity {
			txpool.cond.Wait()
		}
		if tx.Time.IsZero() {
			tx.Time = time.Now()
		}
		txpool.TxQueue = append(txpool.TxQueue, tx)
	}
}

// Add transactions into the pool head. It will block if the pool is full.
// 从头部向交易池中添加一组交易，如果池已满，则会阻塞
func (txpool *TxPool) AddTxs2Pool_Head(txs []*core.Transaction) {
	txpool.lock.Lock()
	defer txpool.lock.Unlock()

	for i := len(txs) - 1; i >= 0; i-- { // 为了保持原顺序，从后往前检查和添加
		tx := txs[i]
		for len(txpool.TxQueue) >= txpool.capacity {
			txpool.cond.Wait()
		}
		// prepend a single transaction
		txpool.TxQueue = append([]*core.Transaction{tx}, txpool.TxQueue...)
	}
}

// Pack transactions for a proposal. It notifies waiting goroutines after packing.
// 打包指定数量的交易，并在打包后通知等待的协程
func (txpool *TxPool) PackTxs(max_txs uint64) []*core.Transaction {
	txpool.lock.Lock()
	defer txpool.lock.Unlock()

	txNum := max_txs
	if uint64(len(txpool.TxQueue)) < txNum {
		txNum = uint64(len(txpool.TxQueue))
	}

	if txNum == 0 {
		return nil
	}

	txs_Packed := txpool.TxQueue[:txNum]
	// 创建一个新的切片以避免内存泄漏 (aliasing)
	remainingTxs := make([]*core.Transaction, len(txpool.TxQueue)-int(txNum))
	copy(remainingTxs, txpool.TxQueue[txNum:])
	txpool.TxQueue = remainingTxs

	// 交易被取出，池中有了空间，唤醒所有可能在等待的生产者
	txpool.cond.Broadcast()

	return txs_Packed
}

// Pack transaction for a proposal (use 'BlocksizeInBytes' to control). It notifies waiting goroutines after packing.
// 打包交易，直到超过指定的字节数，并在打包后通知等待的协程
func (txpool *TxPool) PackTxsWithBytes(max_bytes int) []*core.Transaction {
	txpool.lock.Lock()
	defer txpool.lock.Unlock()

	if len(txpool.TxQueue) == 0 {
		return nil
	}

	txNum := len(txpool.TxQueue)
	currentSize := 0
	for tx_idx, tx := range txpool.TxQueue {
		currentSize += int(unsafe.Sizeof(*tx))
		if currentSize > max_bytes {
			txNum = tx_idx
			break
		}
	}

	if txNum == 0 { // 如果第一个交易就超限了
		return nil
	}

	txs_Packed := txpool.TxQueue[:txNum]
	// 创建一个新的切片以避免内存泄漏 (aliasing)
	remainingTxs := make([]*core.Transaction, len(txpool.TxQueue)-txNum)
	copy(remainingTxs, txpool.TxQueue[txNum:])
	txpool.TxQueue = remainingTxs

	// 交易被取出，池中有了空间，唤醒所有可能在等待的生产者
	txpool.cond.Broadcast()

	return txs_Packed
}

// txpool get locked
func (txpool *TxPool) GetLocked() {
	txpool.lock.Lock()
}

// txpool get unlocked
// 注意：如果手动解锁，请确保在需要时手动调用 cond.Broadcast()
func (txpool *TxPool) GetUnlocked() {
	txpool.lock.Unlock()
}

// get the length of tx queue
// 获取交易池的长度
func (txpool *TxPool) GetTxQueueLen() int {
	txpool.lock.Lock()
	defer txpool.lock.Unlock()
	return len(txpool.TxQueue)
}
