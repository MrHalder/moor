package forward

import (
	"context"
	"fmt"
	"io"
	"net"
	"testing"
	"time"
)

func findFreePort(t *testing.T) uint16 {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return uint16(port)
}

func startEchoServer(t *testing.T, port uint16) {
	t.Helper()
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("echo server listen: %v", err)
	}
	t.Cleanup(func() { l.Close() })

	go func() {
		for {
			conn, err := l.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				io.Copy(c, c) // echo
			}(conn)
		}
	}()
}

func TestForwardBasic(t *testing.T) {
	toPort := findFreePort(t)
	fromPort := findFreePort(t)

	startEchoServer(t, toPort)

	fwd := New(fromPort, toPort)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- fwd.Start(ctx)
	}()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Connect to the forwarded port
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", fromPort))
	if err != nil {
		t.Fatalf("connect to forwarded port: %v", err)
	}
	defer conn.Close()

	// Send data and verify echo
	msg := "hello moor"
	_, err = conn.Write([]byte(msg))
	if err != nil {
		t.Fatalf("write: %v", err)
	}

	buf := make([]byte, len(msg))
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, err = io.ReadFull(conn, buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	if string(buf) != msg {
		t.Errorf("expected %q, got %q", msg, string(buf))
	}

	// Check stats
	stats := fwd.GetStats()
	if stats.TotalConns != 1 {
		t.Errorf("expected 1 total conn, got %d", stats.TotalConns)
	}

	// Stop
	fwd.Stop()

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("forwarder error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("forwarder did not stop in time")
	}
}

func TestForwardPortInUse(t *testing.T) {
	port := findFreePort(t)

	// Occupy the port on 127.0.0.1 (same address the forwarder binds)
	l, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		t.Fatalf("occupy port: %v", err)
	}
	defer l.Close()

	fwd := New(port, 9999)
	err = fwd.Start(context.Background())
	if err == nil {
		t.Fatal("expected error when port is in use")
	}
}

func TestForwardStats(t *testing.T) {
	fwd := New(0, 0)
	stats := fwd.GetStats()
	if stats.ActiveConns != 0 || stats.TotalConns != 0 {
		t.Error("expected zero stats initially")
	}
}
