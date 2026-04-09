package forward

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
)

// Forwarder forwards TCP traffic from one local port to another.
type Forwarder struct {
	FromPort    uint16
	ToPort      uint16
	listener    net.Listener
	activeConns atomic.Int64
	totalConns  atomic.Int64
	wg          sync.WaitGroup
	cancel      context.CancelFunc
}

// Stats holds forwarding statistics.
type Stats struct {
	ActiveConns int64  `json:"active_connections"`
	TotalConns  int64  `json:"total_connections"`
	FromPort    uint16 `json:"from_port"`
	ToPort      uint16 `json:"to_port"`
}

// New creates a Forwarder. Call Start to begin.
func New(from, to uint16) *Forwarder {
	return &Forwarder{
		FromPort: from,
		ToPort:   to,
	}
}

// Start begins listening on FromPort and forwarding to ToPort.
// Blocks until the context is cancelled or an error occurs.
func (f *Forwarder) Start(ctx context.Context) error {
	ctx, f.cancel = context.WithCancel(ctx)

	var err error
	f.listener, err = net.Listen("tcp", fmt.Sprintf(":%d", f.FromPort))
	if err != nil {
		return fmt.Errorf("listen on port %d: %w", f.FromPort, err)
	}

	// Close listener when context is done
	go func() {
		<-ctx.Done()
		f.listener.Close()
	}()

	for {
		conn, err := f.listener.Accept()
		if err != nil {
			// Check if we were intentionally stopped
			select {
			case <-ctx.Done():
				f.wg.Wait()
				return nil
			default:
				return fmt.Errorf("accept: %w", err)
			}
		}

		f.wg.Add(1)
		f.activeConns.Add(1)
		f.totalConns.Add(1)

		go f.handleConn(ctx, conn)
	}
}

// Stop gracefully shuts down the forwarder.
func (f *Forwarder) Stop() {
	if f.cancel != nil {
		f.cancel()
	}
	f.wg.Wait()
}

// GetStats returns current forwarding statistics.
func (f *Forwarder) GetStats() Stats {
	return Stats{
		ActiveConns: f.activeConns.Load(),
		TotalConns:  f.totalConns.Load(),
		FromPort:    f.FromPort,
		ToPort:      f.ToPort,
	}
}

func (f *Forwarder) handleConn(ctx context.Context, src net.Conn) {
	defer f.wg.Done()
	defer f.activeConns.Add(-1)
	defer src.Close()

	dst, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", f.ToPort))
	if err != nil {
		return
	}
	defer dst.Close()

	done := make(chan struct{})

	// src -> dst
	go func() {
		io.Copy(dst, src)
		done <- struct{}{}
	}()

	// dst -> src
	go func() {
		io.Copy(src, dst)
		done <- struct{}{}
	}()

	// Wait for either direction to finish or context cancellation
	select {
	case <-done:
	case <-ctx.Done():
	}
}
