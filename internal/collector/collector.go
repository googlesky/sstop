package collector

import (
	"sort"
	"sync"
	"time"

	"github.com/googlesky/sstop/internal/geo"
	"github.com/googlesky/sstop/internal/model"
	"github.com/googlesky/sstop/internal/platform"
)

const (
	emaAlpha = 0.3
)

// socketTracker tracks per-socket bandwidth over time.
type socketTracker struct {
	prevBytesSent uint64
	prevBytesRecv uint64
	upEMA         *EMA
	downEMA       *EMA
	firstSeen     time.Time
	lastSeen      time.Time
}

// ifaceTracker tracks per-interface bandwidth.
type ifaceTracker struct {
	prevBytesSent uint64
	prevBytesRecv uint64
	upEMA         *EMA
	downEMA       *EMA
}

// Collector periodically polls the platform and produces Snapshots.
type Collector struct {
	platform platform.Platform
	interval time.Duration
	dns      *DNSCache

	mu           sync.Mutex
	sockets      map[platform.SocketKey]*socketTracker
	ifaces       map[string]*ifaceTracker
	procHistory  map[uint32]*RingBuffer // PID → bandwidth history
	totalHistory *RingBuffer            // system-wide rate history for header sparkline
	lastPoll     time.Time

	// Cumulative tracking (for exit summary + cumulative mode)
	sessionStart time.Time
	totalCumUp   uint64
	totalCumDown uint64
	cumByPID     map[uint32]*model.ProcessCumulative

	stopOnce   sync.Once
	stopCh     chan struct{}
	snapCh     chan model.Snapshot
	intervalCh chan time.Duration // dynamic interval changes
}

// New creates a new Collector.
func New(p platform.Platform, interval time.Duration) *Collector {
	return &Collector{
		platform:     p,
		interval:     interval,
		dns:          NewDNSCache(),
		sockets:      make(map[platform.SocketKey]*socketTracker),
		ifaces:       make(map[string]*ifaceTracker),
		procHistory:  make(map[uint32]*RingBuffer),
		totalHistory: NewRingBufferN(60), // 60 samples = 1 min at 1s interval
		sessionStart: time.Now(),
		cumByPID:     make(map[uint32]*model.ProcessCumulative),
		stopCh:       make(chan struct{}),
		snapCh:       make(chan model.Snapshot, 1),
		intervalCh:   make(chan time.Duration, 1),
	}
}

// Start begins periodic collection. Returns a channel that receives Snapshots.
func (c *Collector) Start() <-chan model.Snapshot {
	go c.loop()
	return c.snapCh
}

// Stop halts the collector and closes the snapshot channel.
func (c *Collector) Stop() {
	c.stopOnce.Do(func() {
		close(c.stopCh)
	})
}

// SetInterval changes the polling interval dynamically.
func (c *Collector) SetInterval(d time.Duration) {
	select {
	case c.intervalCh <- d:
	default:
		// Drop if channel full (previous change not consumed yet)
		select {
		case <-c.intervalCh:
		default:
		}
		c.intervalCh <- d
	}
}

// Interval returns the current polling interval.
func (c *Collector) Interval() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.interval
}

func (c *Collector) loop() {
	defer close(c.snapCh) // unblocks any WaitForSnapshot goroutine

	// Initial poll immediately
	c.poll()

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case newInterval := <-c.intervalCh:
			c.mu.Lock()
			c.interval = newInterval
			c.mu.Unlock()
			ticker.Reset(newInterval)
		case <-ticker.C:
			c.poll()
		}
	}
}

func (c *Collector) poll() {
	now := time.Now()

	sockets, ifaces, err := c.platform.Collect()
	if err != nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	dt := now.Sub(c.lastPoll).Seconds()
	if dt <= 0 {
		dt = 1.0
	}
	isFirstPoll := c.lastPoll.IsZero()
	c.lastPoll = now

	// Track which socket keys are active this poll
	activeKeys := make(map[platform.SocketKey]bool)

	// Per-process aggregation
	type procData struct {
		info     model.ProcessInfo
		conns    []model.Connection
		listen   []model.ListenPort
		upRate   float64
		downRate float64
	}
	procs := make(map[uint32]*procData)

	getProc := func(pid uint32, name, cmdline string) *procData {
		pd, ok := procs[pid]
		if !ok {
			pd = &procData{
				info: model.ProcessInfo{
					PID:     pid,
					Name:    name,
					Cmdline: cmdline,
				},
			}
			procs[pid] = pd
		}
		return pd
	}

	for i := range sockets {
		s := &sockets[i]
		key := platform.MakeSocketKey(s)
		activeKeys[key] = true

		tracker, exists := c.sockets[key]
		if !exists {
			tracker = &socketTracker{
				prevBytesSent: s.BytesSent,
				prevBytesRecv: s.BytesRecv,
				upEMA:         NewEMA(emaAlpha),
				downEMA:       NewEMA(emaAlpha),
				firstSeen:     now,
			}
			c.sockets[key] = tracker
		}

		var upRate, downRate float64
		if !isFirstPoll && exists {
			deltaSent := safeDelta(s.BytesSent, tracker.prevBytesSent)
			deltaRecv := safeDelta(s.BytesRecv, tracker.prevBytesRecv)
			rawUp := float64(deltaSent) / dt
			rawDown := float64(deltaRecv) / dt
			upRate = tracker.upEMA.Update(rawUp)
			downRate = tracker.downEMA.Update(rawDown)

			// Cumulative tracking
			c.totalCumUp += deltaSent
			c.totalCumDown += deltaRecv
			if s.PID != 0 {
				pc, ok := c.cumByPID[s.PID]
				if !ok {
					pc = &model.ProcessCumulative{PID: s.PID, Name: s.ProcessName}
					c.cumByPID[s.PID] = pc
				}
				pc.BytesUp += deltaSent
				pc.BytesDown += deltaRecv
				if pc.Name == "" {
					pc.Name = s.ProcessName
				}
			}
		}

		tracker.prevBytesSent = s.BytesSent
		tracker.prevBytesRecv = s.BytesRecv
		tracker.lastSeen = now

		// Aggregate into process
		pd := getProc(s.PID, s.ProcessName, s.Cmdline)

		if s.State == model.StateListen {
			pd.listen = append(pd.listen, model.ListenPort{
				Proto: s.Proto,
				IP:    s.SrcIP,
				Port:  s.SrcPort,
			})
		} else {
			pd.conns = append(pd.conns, model.Connection{
				Proto:      s.Proto,
				SrcIP:      s.SrcIP,
				SrcPort:    s.SrcPort,
				DstIP:      s.DstIP,
				DstPort:    s.DstPort,
				State:      s.State,
				UpRate:     upRate,
				DownRate:   downRate,
				Age:        now.Sub(tracker.firstSeen),
				RemoteHost: c.dns.Resolve(s.DstIP),
				Service:    model.ServiceName(s.DstPort, s.SrcPort),
			})
		}
		pd.upRate += upRate
		pd.downRate += downRate
	}

	// Clean up stale socket trackers (not seen for 30s)
	staleThreshold := now.Add(-30 * time.Second)
	for key, tracker := range c.sockets {
		if !activeKeys[key] && tracker.lastSeen.Before(staleThreshold) {
			delete(c.sockets, key)
		}
	}

	// Process interface stats
	var ifaceStats []model.InterfaceStats
	var totalUp, totalDown float64

	for _, iface := range ifaces {
		tracker, exists := c.ifaces[iface.Name]
		if !exists {
			tracker = &ifaceTracker{
				prevBytesSent: iface.BytesSent,
				prevBytesRecv: iface.BytesRecv,
				upEMA:         NewEMA(emaAlpha),
				downEMA:       NewEMA(emaAlpha),
			}
			c.ifaces[iface.Name] = tracker
		}

		var upRate, downRate float64
		if !isFirstPoll && exists {
			deltaSent := safeDelta(iface.BytesSent, tracker.prevBytesSent)
			deltaRecv := safeDelta(iface.BytesRecv, tracker.prevBytesRecv)
			rawUp := float64(deltaSent) / dt
			rawDown := float64(deltaRecv) / dt
			upRate = tracker.upEMA.Update(rawUp)
			downRate = tracker.downEMA.Update(rawDown)
			totalUp += upRate
			totalDown += downRate
		}

		tracker.prevBytesSent = iface.BytesSent
		tracker.prevBytesRecv = iface.BytesRecv

		ifaceStats = append(ifaceStats, model.InterfaceStats{
			Name:      iface.Name,
			BytesRecv: iface.BytesRecv,
			BytesSent: iface.BytesSent,
			RecvRate:  downRate,
			SendRate:  upRate,
		})
	}

	// Build process summaries + update history
	activePIDs := make(map[uint32]bool)
	var processes []model.ProcessSummary
	for _, pd := range procs {
		pid := pd.info.PID
		activePIDs[pid] = true

		// Update sparkline history
		hist, ok := c.procHistory[pid]
		if !ok {
			hist = &RingBuffer{}
			c.procHistory[pid] = hist
		}
		hist.Push(pd.upRate + pd.downRate)

		// Populate cumulative bytes from tracking
		var cumUp, cumDown uint64
		if pc, ok := c.cumByPID[pid]; ok {
			cumUp = pc.BytesUp
			cumDown = pc.BytesDown
		}

		containerID, serviceName := readCgroup(pid)

		ps := model.ProcessSummary{
			PID:         pid,
			PPID:        readPPID(pid),
			Name:        pd.info.Name,
			Cmdline:     pd.info.Cmdline,
			UpRate:      pd.upRate,
			DownRate:    pd.downRate,
			Connections: pd.conns,
			ListenPorts: pd.listen,
			ConnCount:   len(pd.conns),
			ListenCount: len(pd.listen),
			CumUp:       cumUp,
			CumDown:     cumDown,
			ContainerID: containerID,
			ServiceName: serviceName,
			RateHistory: hist.Samples(),
		}
		processes = append(processes, ps)
	}

	// Clean up history for processes that disappeared
	for pid := range c.procHistory {
		if !activePIDs[pid] {
			delete(c.procHistory, pid)
		}
	}

	// Aggregate remote hosts across all processes
	type hostAgg struct {
		ip        string
		rawIP     []byte
		hostname  string
		upRate    float64
		downRate  float64
		connCount int
		procNames map[string]bool
	}
	hostMap := make(map[string]*hostAgg)
	for _, pd := range procs {
		for _, conn := range pd.conns {
			if conn.DstIP == nil {
				continue
			}
			ipKey := conn.DstIP.String()
			ha, ok := hostMap[ipKey]
			if !ok {
				ha = &hostAgg{
					ip:        ipKey,
					rawIP:     make([]byte, len(conn.DstIP)),
					procNames: make(map[string]bool),
				}
				copy(ha.rawIP, conn.DstIP)
				ha.hostname = conn.RemoteHost
				hostMap[ipKey] = ha
			}
			ha.upRate += conn.UpRate
			ha.downRate += conn.DownRate
			ha.connCount++
			if pd.info.Name != "" {
				ha.procNames[pd.info.Name] = true
			}
		}
	}

	var remoteHosts []model.RemoteHostSummary
	for _, ha := range hostMap {
		var prNames []string
		for name := range ha.procNames {
			prNames = append(prNames, name)
		}
		sort.Strings(prNames)
		country := geo.Lookup(ha.rawIP)
		remoteHosts = append(remoteHosts, model.RemoteHostSummary{
			Host:      ha.hostname,
			IP:        ha.rawIP,
			Country:   country.Format(),
			UpRate:    ha.upRate,
			DownRate:  ha.downRate,
			ConnCount: ha.connCount,
			Processes: prNames,
		})
	}

	// Sort remote hosts by total rate descending
	sort.Slice(remoteHosts, func(i, j int) bool {
		return (remoteHosts[i].UpRate + remoteHosts[i].DownRate) >
			(remoteHosts[j].UpRate + remoteHosts[j].DownRate)
	})

	// Aggregate all listening ports system-wide
	var listenPorts []model.ListenPortEntry
	for _, pd := range procs {
		for _, lp := range pd.listen {
			listenPorts = append(listenPorts, model.ListenPortEntry{
				Proto:   lp.Proto,
				IP:      lp.IP,
				Port:    lp.Port,
				PID:     pd.info.PID,
				Process: pd.info.Name,
				Cmdline: pd.info.Cmdline,
			})
		}
	}
	// Sort by port number, then proto
	sort.Slice(listenPorts, func(i, j int) bool {
		if listenPorts[i].Port != listenPorts[j].Port {
			return listenPorts[i].Port < listenPorts[j].Port
		}
		return listenPorts[i].Proto < listenPorts[j].Proto
	})

	// Update total rate history for header sparkline
	c.totalHistory.Push(totalUp + totalDown)

	snap := model.Snapshot{
		Timestamp:        now,
		Processes:        processes,
		Interfaces:       ifaceStats,
		RemoteHosts:      remoteHosts,
		ListenPorts:      listenPorts,
		TotalUp:          totalUp,
		TotalDown:        totalDown,
		TotalRateHistory: c.totalHistory.Samples(),
	}

	// Non-blocking send — drop oldest if consumer is slow
	select {
	case c.snapCh <- snap:
	default:
		select {
		case <-c.snapCh:
		default:
		}
		select {
		case c.snapCh <- snap:
		default:
		}
	}
}

// SessionStats returns cumulative session statistics.
func (c *Collector) SessionStats() model.SessionStats {
	c.mu.Lock()
	defer c.mu.Unlock()

	stats := model.SessionStats{
		Duration:  time.Since(c.sessionStart),
		TotalUp:   c.totalCumUp,
		TotalDown: c.totalCumDown,
	}

	// Collect all process cumulatives
	all := make([]model.ProcessCumulative, 0, len(c.cumByPID))
	for _, pc := range c.cumByPID {
		all = append(all, *pc)
	}

	// Sort by total bytes descending
	sort.Slice(all, func(i, j int) bool {
		return (all[i].BytesUp + all[i].BytesDown) > (all[j].BytesUp + all[j].BytesDown)
	})

	// Top 5
	if len(all) > 5 {
		all = all[:5]
	}
	stats.TopProcess = all

	return stats
}

// CumulativeByPID returns cumulative bytes for a specific PID.
func (c *Collector) CumulativeByPID(pid uint32) (up, down uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if pc, ok := c.cumByPID[pid]; ok {
		return pc.BytesUp, pc.BytesDown
	}
	return 0, 0
}

// safeDelta handles counter wraps (uint64 overflow).
func safeDelta(current, previous uint64) uint64 {
	if current >= previous {
		return current - previous
	}
	// Counter wrapped — return 0 to avoid a spike.
	// Real wraps of uint64 byte counters are essentially impossible.
	return 0
}
