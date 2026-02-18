package engine

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/samaelod/nabu/types"
)

// Engine handles the simulation/replay of network events from a Config.
type Engine struct {
	Config    *types.Config
	Running   bool
	Stopped   bool
	Listeners map[int]net.Listener     // Map of Server IDs to listeners
	Clients   map[int]map[int]net.Conn // Map of [From -> [To -> Conn]]
	Mutex     sync.Mutex
	Log       *Logger
	ActiveEnd map[int]bool // Track which endpoints are running
	Status    map[int]types.EndpointStatus
	Ctx       context.Context    // Context for cancellation
	Cancel    context.CancelFunc // Cancel function

	activeCount   int                 // Number of active endpoints
	endpointMutex map[int]*sync.Mutex // Per-endpoint mutexes for connection ops

	timeout time.Duration // Connection timeout
	delay   time.Duration // Default delay between messages
}

// NewEngine creates a new simulation engine instance.
func NewEngine(cfg *types.Config, logPath string, logLines int, timeoutMs int, delayMs int) *Engine {
	ctx, cancel := context.WithCancel(context.Background())

	if logLines <= 0 {
		logLines = 1000
	}
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}
	if delayMs < 0 {
		delayMs = 0
	}

	return &Engine{
		Config:        cfg,
		Listeners:     make(map[int]net.Listener),
		Clients:       make(map[int]map[int]net.Conn),
		Log:           NewLogger(logPath, logLines),
		ActiveEnd:     make(map[int]bool),
		Status:        make(map[int]types.EndpointStatus),
		Ctx:           ctx,
		Cancel:        cancel,
		activeCount:   0,
		endpointMutex: make(map[int]*sync.Mutex),
		timeout:       time.Duration(timeoutMs) * time.Millisecond,
		delay:         time.Duration(delayMs) * time.Millisecond,
	}
}

// StartEndpoint starts the simulation for a single endpoint.
func (e *Engine) StartEndpoint(id int) {
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	if e.ActiveEnd[id] {
		return
	}

	// Create per-endpoint mutex if not exists
	if _, ok := e.endpointMutex[id]; !ok {
		e.endpointMutex[id] = &sync.Mutex{}
	}

	e.ActiveEnd[id] = true
	e.Running = true

	ep := e.findEndpoint(id)
	isServer := ep != nil && ep.Kind == "server"

	// Only increment activeCount for client endpoints (servers just listen)
	if !isServer {
		e.activeCount++
	}

	e.Status[id] = types.StatusRunning

	go e.runEndpoint(id, isServer)
}

func (e *Engine) findEndpoint(id int) *types.Endpoint {
	for i := range e.Config.Endpoints {
		if e.Config.Endpoints[i].ID == id {
			return &e.Config.Endpoints[i]
		}
	}
	return nil
}

func (e *Engine) runEndpoint(id int, isServer bool) {
	// Ensure messages are indexed
	if e.Config.MessagesByFrom == nil {
		e.Config.IndexMessages()
	}

	// Find endpoint config
	ep := e.findEndpoint(id)
	if ep == nil {
		e.log(fmt.Sprintf("Endpoint %d not found", id))
		if !isServer {
			e.finishEndpoint(id)
		}
		return
	}

	e.log(fmt.Sprintf("Starting endpoint %d (%s)...", id, ep.Kind))

	if ep.Kind == "server" {
		// Just start listener if not already
		e.Mutex.Lock()
		_, exists := e.Listeners[id]
		e.Mutex.Unlock()

		if !exists {
			if err := e.setupListener(*ep); err != nil {
				e.log(fmt.Sprintf("Error starting listener %d: %v", id, err))
				e.Mutex.Lock()
				e.Status[id] = types.StatusError
				e.Mutex.Unlock()
			} else {
				e.Mutex.Lock()
				e.Status[id] = types.StatusRunning // Listeners stay running
				e.Mutex.Unlock()
			}
		} else {
			e.log(fmt.Sprintf("Listener %d already active", id))
		}
		// Server creates no traffic on its own in this model (passive)
		// Unless configured to send? Current setupListeners handles accept loop.
		return
	}

	// Client Logic
	// Iterate through messages where From == id using indexed map
	startTime := time.Now()

	messages := e.Config.MessagesByFrom[id]
	for i, msg := range messages {
		// Check if context was cancelled
		select {
		case <-e.Ctx.Done():
			e.log(fmt.Sprintf("Endpoint %d stopped by user", id))
			e.Mutex.Lock()
			e.Status[id] = types.StatusIdle
			e.Mutex.Unlock()
			e.finishEndpoint(id)
			return
		default:
		}

		if !e.IsRunning(id) {
			break
		}

		// Calculate wait time
		// TDelta is accumulated? Or just delta from previous?
		// In a single-stream pcap, delta is from previous packet.
		// If we skip other people's packets, we should probably accumulate their deltas?
		// OR just assume TDelta is "time to wait before sending this packet".
		// Let's rely on TDelta as "delay before this packet".

		waitDuration := time.Duration(msg.TDelta) * time.Millisecond
		if waitDuration > 0 {
			select {
			case <-time.After(waitDuration):
			case <-e.Ctx.Done():
				e.log(fmt.Sprintf("Endpoint %d stopped by user", id))
				e.Mutex.Lock()
				e.Status[id] = types.StatusIdle
				e.Mutex.Unlock()
				e.finishEndpoint(id)
				return
			}
		}

		// Currently elapsed (for logging)
		elapsed := time.Since(startTime).Milliseconds()

		// Execute Action
		err := e.executeMessage(msg)
		if err != nil {
			e.log(fmt.Sprintf("[+%dms] Error msg %d: %v", elapsed, i, err))
			e.Mutex.Lock()
			e.Status[id] = types.StatusError
			e.Mutex.Unlock()
		}
	}

	e.log(fmt.Sprintf("Endpoint %d finished trace.", id))
	e.finishEndpoint(id)
}

func (e *Engine) finishEndpoint(id int) {
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	delete(e.ActiveEnd, id)
	e.activeCount--
	if e.activeCount < 0 {
		e.activeCount = 0
	}

	if e.Status[id] != types.StatusError {
		e.Status[id] = types.StatusCompleted
	}

	// Only set Running to false if no more endpoints are active
	if e.activeCount == 0 {
		e.Running = false
	}
}

// GetStatus returns the current status of an endpoint safely
func (e *Engine) GetStatus(id int) types.EndpointStatus {
	e.Mutex.Lock()
	defer e.Mutex.Unlock()
	return e.Status[id]
}

func (e *Engine) IsRunning(id int) bool {
	e.Mutex.Lock()
	defer e.Mutex.Unlock()
	return e.ActiveEnd[id]
}

// StopEndpoint stops a running endpoint
func (e *Engine) StopEndpoint(id int) {
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	if !e.ActiveEnd[id] {
		return
	}

	isServer := false
	if ep := e.findEndpoint(id); ep != nil {
		isServer = ep.Kind == "server"
	}

	delete(e.ActiveEnd, id)

	// Only decrement activeCount for client endpoints (servers don't use it)
	if !isServer {
		e.activeCount--
		if e.activeCount < 0 {
			e.activeCount = 0
		}
	}

	if e.Status[id] == types.StatusRunning {
		e.Status[id] = types.StatusIdle
	}

	// Only set Running to false if no more client endpoints are active
	if e.activeCount == 0 {
		e.Running = false
	}

	// Close listener if exists
	if ln, ok := e.Listeners[id]; ok {
		ln.Close()
		delete(e.Listeners, id)
	}

	// Close client connections for this endpoint
	if conns, ok := e.Clients[id]; ok {
		for _, conn := range conns {
			conn.Close()
		}
		delete(e.Clients, id)
	}

	e.log(fmt.Sprintf("Endpoint %d stopped", id))
}

// StopAll stops all running endpoints
func (e *Engine) StopAll() {
	e.Mutex.Lock()
	for id := range e.ActiveEnd {
		delete(e.ActiveEnd, id)
		if e.Status[id] == types.StatusRunning {
			e.Status[id] = types.StatusIdle
		}

		// Close listener if exists
		if ln, ok := e.Listeners[id]; ok {
			ln.Close()
			delete(e.Listeners, id)
		}

		// Close client connections for this endpoint
		if conns, ok := e.Clients[id]; ok {
			for _, conn := range conns {
				conn.Close()
			}
			delete(e.Clients, id)
		}
	}
	e.activeCount = 0
	e.Running = false
	e.Mutex.Unlock()

	// Cancel context to stop any running goroutines
	e.Cancel()
	e.log("All endpoints stopped")

	// Create new context for future runs
	e.Ctx, e.Cancel = context.WithCancel(context.Background())
}

// setupListener starts a single listener
func (e *Engine) setupListener(ep types.Endpoint) error {
	e.Mutex.Lock()
	defer e.Mutex.Unlock()

	addr := net.JoinHostPort(ep.Address, strconv.Itoa(ep.Port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("endpoint %d (%s): %w", ep.ID, addr, err)
	}
	e.Listeners[ep.ID] = ln
	e.log(fmt.Sprintf("Endpoint %d listening on %s", ep.ID, addr))

	// Create connection map entry for this server
	if _, ok := e.Clients[ep.ID]; !ok {
		e.Clients[ep.ID] = make(map[int]net.Conn)
	}

	// Start accept loop — keep accepted connections alive and drain data
	go func(id int, listener net.Listener, ctx context.Context) {
		for {
			conn, err := listener.Accept()
			if err != nil {
				// Check if closed by context
				select {
				case <-ctx.Done():
					return
				default:
					// Log error only if not intentionally closed
					e.log(fmt.Sprintf("Endpoint %d listener error: %v", id, err))
				}
				return
			}
			e.log(fmt.Sprintf("Endpoint %d accepted connection from %s", id, conn.RemoteAddr()))
			// Drain incoming data in background to prevent kernel buffer from filling
			go func(c net.Conn, drainCtx context.Context) {
				defer c.Close()
				buf := make([]byte, 4096)
				for {
					select {
					case <-drainCtx.Done():
						return
					default:
						c.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
						n, err := c.Read(buf)
						if err != nil {
							return
						}
						if n == 0 {
							return
						}
					}
				}
			}(conn, ctx)
		}
	}(ep.ID, ln, e.Ctx)

	return nil
}

// setupListeners removed in favor of single setupListener

func (e *Engine) executeMessage(msg types.Message) error {
	fromID := msg.From

	// Get endpoint-specific mutex for connection operations
	e.Mutex.Lock()
	epMu, ok := e.endpointMutex[fromID]
	if !ok {
		epMu = &sync.Mutex{}
		e.endpointMutex[fromID] = epMu
	}
	e.Mutex.Unlock()

	switch msg.Kind {
	case "syn":
		// Initiate connection
		// Find 'To' endpoint
		var target *types.Endpoint
		for _, ep := range e.Config.Endpoints {
			if ep.ID == msg.To {
				target = &ep
				break
			}
		}
		if target == nil {
			return fmt.Errorf("target endpoint %d not found", msg.To)
		}

		addr := net.JoinHostPort(target.Address, strconv.Itoa(target.Port))
		e.log(fmt.Sprintf("Connecting %d -> %d (%s)...", fromID, msg.To, addr))

		// Network I/O outside of mutex
		conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err != nil {
			return fmt.Errorf("connect failed: %w", err)
		}

		// Store connection under endpoint mutex
		epMu.Lock()
		if e.Clients[fromID] == nil {
			e.Clients[fromID] = make(map[int]net.Conn)
		}
		e.Clients[fromID][msg.To] = conn
		epMu.Unlock()

		e.log(fmt.Sprintf("Connected %d -> %d", fromID, msg.To))

	case "data", "psh", "push":
		// Send Data
		// Check if we have a connection from 'From' to 'To'
		epMu.Lock()
		conn, ok := e.Clients[fromID][msg.To]
		epMu.Unlock()

		if !ok {
			e.log(fmt.Sprintf("simulating data %d -> %d (no active conn)", fromID, msg.To))
			return nil
		}

		// data hex decode
		data, err := hex.DecodeString(msg.Value)
		if err != nil {
			return fmt.Errorf("invalid hex payload: %v", err)
		}

		if len(data) > 0 {
			_, err := conn.Write(data)
			if err != nil {
				return fmt.Errorf("write failed: %w", err)
			}
			e.log(fmt.Sprintf("Sent %d bytes %d -> %d", len(data), fromID, msg.To))
		}

	case "fin":
		// Close connection
		epMu.Lock()
		conn, ok := e.Clients[fromID][msg.To]
		if ok {
			conn.Close()
			delete(e.Clients[fromID], msg.To)
		}
		epMu.Unlock()

		if ok {
			e.log(fmt.Sprintf("Closed connection %d -> %d", fromID, msg.To))
		}

	default:
	}

	return nil
}

func (e *Engine) log(msg string) {
	ts := time.Now().Format("15:04:05")
	line := fmt.Sprintf("[%s] %s", ts, msg)
	e.Log.Write(line)
}
