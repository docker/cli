package container

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
)

type stats struct {
	mu sync.RWMutex
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

func collect(ctx context.Context, s *Stats, cli client.ContainerAPIClient, streamStats bool, waitFirst *sync.WaitGroup) { //nolint:gocyclo
	var getFirst bool

	defer func() {
		// if error happens, and we get nothing of stats, release wait group whatever
		if !getFirst {
			getFirst = true
			waitFirst.Done()
		}
	}()

	response, err := cli.ContainerStats(ctx, s.Container, client.ContainerStatsOptions{
		Stream:                streamStats,
		IncludePreviousSample: !streamStats, // collect previous CPU value for the first result when not streaming.
	})
	if err != nil {
		s.SetError(err)
		return
	}

	u := make(chan error, 1)
	go func() {
		defer response.Body.Close()
		dec := json.NewDecoder(response.Body)
		for {
			if ctx.Err() != nil {
				return
			}
			var v container.StatsResponse
			if err := dec.Decode(&v); err != nil {
				dec = json.NewDecoder(io.MultiReader(dec.Buffered(), response.Body))
				u <- err
				if err == io.EOF {
					break
				}
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Daemon versions before v29 did not return per-stats OSType;
			// fall back to using the daemon's OSType.
			if v.OSType == "" {
				v.OSType = daemonOSType
			}

			if daemonOSType == "windows" {
				netRx, netTx := calculateNetwork(v.Networks)
				s.SetStatistics(StatsEntry{
					Name:          v.Name,
					ID:            v.ID,
					CPUPercentage: calculateCPUPercentWindows(&v),
					Memory:        float64(v.MemoryStats.PrivateWorkingSet),
					NetworkRx:     netRx,
					NetworkTx:     netTx,
					BlockRead:     float64(v.StorageStats.ReadSizeBytes),
					BlockWrite:    float64(v.StorageStats.WriteSizeBytes),
				})
			} else {
				memUsage := calculateMemUsageUnixNoCache(v.MemoryStats)
				netRx, netTx := calculateNetwork(v.Networks)
				blkRead, blkWrite := calculateBlockIO(v.BlkioStats)
				s.SetStatistics(StatsEntry{
					Name:             v.Name,
					ID:               v.ID,
					CPUPercentage:    calculateCPUPercentUnix(v.PreCPUStats, v.CPUStats),
					Memory:           memUsage,
					MemoryPercentage: calculateMemPercentUnixNoCache(float64(v.MemoryStats.Limit), memUsage),
					MemoryLimit:      float64(v.MemoryStats.Limit),
					NetworkRx:        netRx,
					NetworkTx:        netTx,
					BlockRead:        float64(blkRead),
					BlockWrite:       float64(blkWrite),
					PidsCurrent:      v.PidsStats.Current,
				})
			}
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
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				continue
			}
			// if this is the first stat you get, release WaitGroup
			if !getFirst {
				getFirst = true
				waitFirst.Done()
			}
		case <-ctx.Done():
			s.SetError(ctx.Err())
			return
		}
		if !streamStats {
			return
		}
	}
}

func calculateCPUPercentUnix(previousCPU container.CPUStats, curCPUStats container.CPUStats) float64 {
	var (
		cpuPercent = 0.0
		// calculate the change for the cpu usage of the container in between readings
		cpuDelta = float64(curCPUStats.CPUUsage.TotalUsage) - float64(previousCPU.CPUUsage.TotalUsage)
		// calculate the change for the entire system between readings
		systemDelta = float64(curCPUStats.SystemUsage) - float64(previousCPU.SystemUsage)
		onlineCPUs  = float64(curCPUStats.OnlineCPUs)
	)

	if onlineCPUs == 0.0 {
		onlineCPUs = float64(len(curCPUStats.CPUUsage.PercpuUsage))
	}
	if systemDelta > 0.0 && cpuDelta > 0.0 {
		cpuPercent = (cpuDelta / systemDelta) * onlineCPUs * 100.0
	}
	return cpuPercent
}

func calculateCPUPercentWindows(v *container.StatsResponse) float64 {
	// Max number of 100ns intervals between the previous time read and now
	possIntervals := uint64(v.Read.Sub(v.PreRead).Nanoseconds()) // Start with number of ns intervals
	possIntervals /= 100                                         // Convert to number of 100ns intervals
	possIntervals *= uint64(v.NumProcs)                          // Multiply by the number of processors

	// Percentage avoiding divide-by-zero
	if possIntervals > 0 {
		intervalsUsed := v.CPUStats.CPUUsage.TotalUsage - v.PreCPUStats.CPUUsage.TotalUsage
		return float64(intervalsUsed) / float64(possIntervals) * 100.0
	}
	return 0.00
}

func calculateBlockIO(blkio container.BlkioStats) (uint64, uint64) {
	var blkRead, blkWrite uint64
	for _, bioEntry := range blkio.IoServiceBytesRecursive {
		if len(bioEntry.Op) == 0 {
			continue
		}
		switch bioEntry.Op[0] {
		case 'r', 'R':
			blkRead += bioEntry.Value
		case 'w', 'W':
			blkWrite += bioEntry.Value
		}
	}
	return blkRead, blkWrite
}

func calculateNetwork(network map[string]container.NetworkStats) (float64, float64) {
	var rx, tx float64

	for _, v := range network {
		rx += float64(v.RxBytes)
		tx += float64(v.TxBytes)
	}
	return rx, tx
}

// calculateMemUsageUnixNoCache calculate memory usage of the container.
// Cache is intentionally excluded to avoid misinterpretation of the output.
//
// On cgroup v1 host, the result is `mem.Usage - mem.Stats["total_inactive_file"]` .
// On cgroup v2 host, the result is `mem.Usage - mem.Stats["inactive_file"] `.
//
// This definition is consistent with cadvisor and containerd/CRI.
// * https://github.com/google/cadvisor/commit/307d1b1cb320fef66fab02db749f07a459245451
// * https://github.com/containerd/cri/commit/6b8846cdf8b8c98c1d965313d66bc8489166059a
//
// On Docker 19.03 and older, the result was `mem.Usage - mem.Stats["cache"]`.
// See https://github.com/moby/moby/issues/40727 for the background.
func calculateMemUsageUnixNoCache(mem container.MemoryStats) float64 {
	// cgroup v1
	if v, isCgroup1 := mem.Stats["total_inactive_file"]; isCgroup1 && v < mem.Usage {
		return float64(mem.Usage - v)
	}
	// cgroup v2
	if v := mem.Stats["inactive_file"]; v < mem.Usage {
		return float64(mem.Usage - v)
	}
	return float64(mem.Usage)
}

func calculateMemPercentUnixNoCache(limit float64, usedNoCache float64) float64 {
	// MemoryStats.Limit will never be 0 unless the container is not running and we haven't
	// got any data from cgroup
	if limit != 0 {
		return usedNoCache / limit * 100.0
	}
	return 0
}
