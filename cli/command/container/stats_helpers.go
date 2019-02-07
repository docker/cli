package container

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type stats struct {
	mu sync.Mutex
	cs []*Stats
}

// daemonOSType is set once we have at least one stat for a container
// from the daemon. It is used to ensure we print the right header based
// on the daemon platform.
var daemonOSType string

func (s *stats) add(cs *Stats) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.isKnownContainer(cs.Container); !exists {
		s.cs = append(s.cs, cs)
		return true
	}
	return false
}

func (s *stats) remove(id string) {
	s.mu.Lock()
	if i, exists := s.isKnownContainer(id); exists {
		s.cs = append(s.cs[:i], s.cs[i+1:]...)
	}
	s.mu.Unlock()
}

func (s *stats) isKnownContainer(cid string) (int, bool) {
	for i, c := range s.cs {
		if c.Container == cid {
			return i, true
		}
	}
	return -1, false
}

func collect(ctx context.Context, s *Stats, cli client.APIClient, streamStats bool, waitFirst *sync.WaitGroup) {
	logrus.Debugf("collecting stats for %s", s.Container)
	var (
		getFirst       bool
		previousCPU    uint64
		previousSystem uint64
		u              = make(chan error, 1)
	)

	defer func() {
		// if error happens and we get nothing of stats, release wait group whatever
		if !getFirst {
			getFirst = true
			waitFirst.Done()
		}
	}()

	response, err := cli.ContainerStats(ctx, s.Container, streamStats)
	if err != nil {
		s.SetError(err)
		return
	}
	defer response.Body.Close()

	dec := json.NewDecoder(response.Body)
	go func() {
		for {
			var (
				v                      			*types.StatsJSON
				memPercent, cpuPercent 			float64
				mem, memLimit          			float64
				blkRead, blkWrite      			uint64 // Only used on Linux
				pidsStatsCurrent       			uint64
				memoryAutoRange, cpuAutoRange   string
			)

			if err := dec.Decode(&v); err != nil {
				dec = json.NewDecoder(io.MultiReader(dec.Buffered(), response.Body))
				u <- err
				if err == io.EOF {
					break
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

			daemonOSType = response.OSType

			if daemonOSType != "windows" {
				previousCPU = v.PreCPUStats.CPUUsage.TotalUsage
				previousSystem = v.PreCPUStats.SystemUsage
				cpuPercent = calculateCPUPercentUnix(previousCPU, previousSystem, v)
				blkRead, blkWrite = calculateBlockIO(v.BlkioStats)
				mem = calculateMemUsageUnixNoCache(v.MemoryStats)
				memLimit = float64(v.MemoryStats.Limit)
				memPercent = calculateMemPercentUnixNoCache(memLimit, mem)
				pidsStatsCurrent = v.PidsStats.Current
				memoryAutoRange = formatMemoryAutoRange(v.AutoRange["memoryAR"])
				cpuAutoRange = formatCPUAutoRange(v.AutoRange["cpuAR"])
			} else {
				cpuPercent = calculateCPUPercentWindows(v)
				blkRead = v.StorageStats.ReadSizeBytes
				blkWrite = v.StorageStats.WriteSizeBytes
				mem = float64(v.MemoryStats.PrivateWorkingSet)
			}
			netRx, netTx := calculateNetwork(v.Networks)
			s.SetStatistics(StatsEntry{
				Name:             v.Name,
				ID:               v.ID,
				CPUPercentage:    cpuPercent,
				Memory:           mem,
				MemoryPercentage: memPercent,
				MemoryLimit:      memLimit,
				NetworkRx:        netRx,
				NetworkTx:        netTx,
				BlockRead:        float64(blkRead),
				BlockWrite:       float64(blkWrite),
				PidsCurrent:      pidsStatsCurrent,
				AutoRange:		  memoryAutoRange + cpuAutoRange,
			})
			u <- nil
			if !streamStats {
				return
			}
		}
	}()
	for {
		select {
		case <-time.After(2 * time.Second):
			// zero out the values if we have not received an update within
			// the specified duration.
			s.SetErrorAndReset(errors.New("timeout waiting for stats"))
			// if this is the first stat you get, release WaitGroup
			if !getFirst {
				getFirst = true
				waitFirst.Done()
			}
		case err := <-u:
			s.SetError(err)
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			// if this is the first stat you get, release WaitGroup
			if !getFirst {
				getFirst = true
				waitFirst.Done()
			}
		}
		if !streamStats {
			return
		}
	}
}

func formatCPUAutoRange(ar map[string]string) string {
	numCPU, exist := ar["numCPU"]
	if !exist {
		numCPU = "--"
	}

	percentOpti, exist := ar["percentOpti"]
	if !exist {
		percentOpti = "--"
	}

	usageOpti, exist := ar["usageOpti"]
	if !exist {
		usageOpti = "--"
	}

	return fmt.Sprintf("%s / %s%% / %s", numCPU, percentOpti, usageOpti)
}

func formatMemoryAutoRange(ar map[string]string) string {
	newMinRange, exist := ar["nmin"]
	if !exist {
		newMinRange = "--"
	}

	newMaxRange, exist := ar["nmax"]
	if !exist {
		newMaxRange = "--"
	}

	sugMin, exist := ar["sugmin"]
	if !exist {
		sugMin = "--"
	}

	sugMax, exist := ar["sugmax"]
	if !exist {
		sugMax = "--"
	}

	return fmt.Sprintf("%s/%s  %s/%s | ", addUnit(newMinRange), addUnit(newMaxRange), addUnit(sugMin), addUnit(sugMax))
}

func addUnit(value string) string {
	lval := len(value)

	if lval > 9 {
		return value[0:lval-9] + "." + value[lval-9:lval-7] + "GiB"
	} else if lval > 6 {
		return value[0:lval-6] + "." + value[lval-6:lval-4] + "MiB"
	} else if lval > 3 {
		return value[0:lval-3] + "." + value[lval-3:lval-1] + "KiB"
	} else if lval > 0 && value != "--" {
		return value + "%"
	}
	
	return value
}

func calculateCPUPercentUnix(previousCPU, previousSystem uint64, v *types.StatsJSON) float64 {
	var (
		cpuPercent = 0.0
		// calculate the change for the cpu usage of the container in between readings
		cpuDelta = float64(v.CPUStats.CPUUsage.TotalUsage) - float64(previousCPU)
		// calculate the change for the entire system between readings
		systemDelta = float64(v.CPUStats.SystemUsage) - float64(previousSystem)
		onlineCPUs  = float64(v.CPUStats.OnlineCPUs)
	)

	if onlineCPUs == 0.0 {
		onlineCPUs = float64(len(v.CPUStats.CPUUsage.PercpuUsage))
	}
	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * onlineCPUs * 100.0
	}
	return cpuPercent
}

func calculateCPUPercentWindows(v *types.StatsJSON) float64 {
	// Max number of 100ns intervals between the previous time read and now
	possIntervals := uint64(v.Read.Sub(v.PreRead).Nanoseconds()) // Start with number of ns intervals
	possIntervals /= 100                                         // Convert to number of 100ns intervals
	possIntervals *= uint64(v.NumProcs)                          // Multiple by the number of processors

	// Intervals used
	intervalsUsed := v.CPUStats.CPUUsage.TotalUsage - v.PreCPUStats.CPUUsage.TotalUsage

	// Percentage avoiding divide-by-zero
	if possIntervals > 0 {
		return float64(intervalsUsed) / float64(possIntervals) * 100.0
	}
	return 0.00
}

func calculateBlockIO(blkio types.BlkioStats) (uint64, uint64) {
	var blkRead, blkWrite uint64
	for _, bioEntry := range blkio.IoServiceBytesRecursive {
		switch strings.ToLower(bioEntry.Op) {
		case "read":
			blkRead = blkRead + bioEntry.Value
		case "write":
			blkWrite = blkWrite + bioEntry.Value
		}
	}
	return blkRead, blkWrite
}

func calculateNetwork(network map[string]types.NetworkStats) (float64, float64) {
	var rx, tx float64

	for _, v := range network {
		rx += float64(v.RxBytes)
		tx += float64(v.TxBytes)
	}
	return rx, tx
}

// calculateMemUsageUnixNoCache calculate memory usage of the container.
// Page cache is intentionally excluded to avoid misinterpretation of the output.
func calculateMemUsageUnixNoCache(mem types.MemoryStats) float64 {
	return float64(mem.Usage - mem.Stats["cache"])
}

func calculateMemPercentUnixNoCache(limit float64, usedNoCache float64) float64 {
	// MemoryStats.Limit will never be 0 unless the container is not running and we haven't
	// got any data from cgroup
	if limit != 0 {
		return usedNoCache / limit * 100.0
	}
	return 0
}
