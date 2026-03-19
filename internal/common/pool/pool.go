package pool

import (
	"context"
	"fmt"
	"runtime"
	"sync"

	"github.com/alitto/pond/v2"
)

var (
	poolInitOnce sync.Once
	poolConfig   PoolConfig
)

var (
	WorkerPool         pond.Pool
	MmWorkerPool       pond.Pool
	SmWorkerPool       pond.Pool
	GnbWorkerPool      pond.Pool
	SctpNgapWorkerPool pond.Pool
)

type PoolConfig struct {
	MaxCapPool        int
	MaxSctpNgapWorker int
	MmWorkers         int
	SmWorkers         int
	GnbWorkers        int
}

func autoDetectPoolSizes(nUEs int, nGnbs int) PoolConfig {
	numCPU := runtime.NumCPU()

	baseWorkersPerCore := 1500

	totalPool := numCPU * baseWorkersPerCore
	if totalPool < 1000 {
		totalPool = 1000
	}
	if totalPool > 50000 {
		totalPool = 50000
	}

	if nUEs > 0 {
		requiredWorkers := (nUEs * 3) + (nGnbs * 100)

		if requiredWorkers > totalPool {
			totalPool = requiredWorkers
			if totalPool > 100000 {
				totalPool = 100000
			}
		}
	}

	sctpWorkers := totalPool / 6

	minSctpWorkers := nGnbs * 50
	if minSctpWorkers < 200 {
		minSctpWorkers = 200
	}
	if sctpWorkers < minSctpWorkers {
		sctpWorkers = minSctpWorkers
	}

	if sctpWorkers >= totalPool/2 {
		sctpWorkers = totalPool / 3
	}

	remainingWorkers := totalPool - sctpWorkers

	mmWorkers := (remainingWorkers * 40) / 100
	smWorkers := (remainingWorkers * 35) / 100
	gnbWorkers := remainingWorkers - mmWorkers - smWorkers

	return PoolConfig{
		MaxCapPool:        totalPool,
		MaxSctpNgapWorker: sctpWorkers,
		MmWorkers:         mmWorkers,
		SmWorkers:         smWorkers,
		GnbWorkers:        gnbWorkers,
	}
}

func InitWorkerPool(ctx context.Context, maxPool, nSctpWorker, nUEs, nGnbs int) PoolConfig {
	poolInitOnce.Do(func() {
		if maxPool <= 0 {
			poolConfig = autoDetectPoolSizes(nUEs, nGnbs)
			fmt.Printf("[Pool] Auto-detected: Total=%d, SCTP=%d, MM=%d, SM=%d, GNB=%d (CPUs=%d)\n",
				poolConfig.MaxCapPool, poolConfig.MaxSctpNgapWorker,
				poolConfig.MmWorkers, poolConfig.SmWorkers, poolConfig.GnbWorkers,
				runtime.NumCPU())
		} else {
			poolConfig.MaxCapPool = maxPool

			if nSctpWorker <= 0 {
				poolConfig.MaxSctpNgapWorker = maxPool / 6

				minSctp := nGnbs * 50
				if minSctp < 200 {
					minSctp = 200
				}
				if poolConfig.MaxSctpNgapWorker < minSctp {
					poolConfig.MaxSctpNgapWorker = minSctp
				}
			} else {
				if nSctpWorker >= maxPool {
					fmt.Printf("[Pool] WARNING: nSctpWorker(%d) >= maxPool(%d), auto-adjusting to %d\n",
						nSctpWorker, maxPool, maxPool/3)
					poolConfig.MaxSctpNgapWorker = maxPool / 3
				} else {
					poolConfig.MaxSctpNgapWorker = nSctpWorker
				}
			}

			remainingWorkers := poolConfig.MaxCapPool - poolConfig.MaxSctpNgapWorker
			if remainingWorkers <= 0 {
				panic(fmt.Sprintf("Invalid pool config: maxPool=%d, sctpWorkers=%d leaves no workers for protocol processing",
					poolConfig.MaxCapPool, poolConfig.MaxSctpNgapWorker))
			}

			poolConfig.MmWorkers = (remainingWorkers * 40) / 100
			poolConfig.SmWorkers = (remainingWorkers * 35) / 100
			poolConfig.GnbWorkers = remainingWorkers - poolConfig.MmWorkers - poolConfig.SmWorkers

			fmt.Printf("[Pool] Manual config: Total=%d, SCTP=%d, MM=%d, SM=%d, GNB=%d\n",
				poolConfig.MaxCapPool, poolConfig.MaxSctpNgapWorker,
				poolConfig.MmWorkers, poolConfig.SmWorkers, poolConfig.GnbWorkers)
		}

		WorkerPool = pond.NewPool(poolConfig.MaxCapPool, pond.WithContext(ctx))

		MmWorkerPool = WorkerPool.NewSubpool(poolConfig.MmWorkers)
		SmWorkerPool = WorkerPool.NewSubpool(poolConfig.SmWorkers)
		GnbWorkerPool = WorkerPool.NewSubpool(poolConfig.GnbWorkers)
		SctpNgapWorkerPool = WorkerPool.NewSubpool(poolConfig.MaxSctpNgapWorker)
	})

	return poolConfig
}

func GetMaxSctpNgapWorker() int {
	return poolConfig.MaxSctpNgapWorker
}

func GetPoolStats() map[string]interface{} {
	if WorkerPool == nil {
		return nil
	}

	return map[string]interface{}{
		"total_capacity": poolConfig.MaxCapPool,
		"total_running":  WorkerPool.RunningWorkers(),
		"mm_capacity":    poolConfig.MmWorkers,
		"mm_running":     MmWorkerPool.RunningWorkers(),
		"sm_capacity":    poolConfig.SmWorkers,
		"sm_running":     SmWorkerPool.RunningWorkers(),
		"gnb_capacity":   poolConfig.GnbWorkers,
		"gnb_running":    GnbWorkerPool.RunningWorkers(),
		"sctp_capacity":  poolConfig.MaxSctpNgapWorker,
		"sctp_running":   SctpNgapWorkerPool.RunningWorkers(),
	}
}
