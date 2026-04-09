package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ashutosh/moor/internal/forward"
	"github.com/spf13/cobra"
)

var forwardCmd = &cobra.Command{
	Use:   "forward <from-port> <to-port>",
	Short: "Forward traffic from one local port to another",
	Long:  "Start a TCP forwarder that proxies all traffic from <from-port> to <to-port> on localhost.",
	Args:  cobra.ExactArgs(2),
	RunE:  runForward,
}

func init() {
	rootCmd.AddCommand(forwardCmd)
}

func runForward(cmd *cobra.Command, args []string) error {
	fromPort, err := parsePort(args[0])
	if err != nil {
		return fmt.Errorf("from-port: %w", err)
	}
	toPort, err := parsePort(args[1])
	if err != nil {
		return fmt.Errorf("to-port: %w", err)
	}

	if fromPort == toPort {
		return fmt.Errorf("from-port and to-port cannot be the same (%d)", fromPort)
	}

	fwd := forward.New(fromPort, toPort)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		errCh <- fwd.Start(ctx)
	}()

	fmt.Printf("Forwarding :%d -> :%d (Ctrl+C to stop)\n", fromPort, toPort)

	// Status ticker
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sigCh:
			fmt.Println("\nStopping forwarder...")
			fwd.Stop()
			stats := fwd.GetStats()
			fmt.Printf("Forwarded %d total connections\n", stats.TotalConns)
			return nil
		case err := <-errCh:
			return err
		case <-ticker.C:
			stats := fwd.GetStats()
			fmt.Printf("  active: %d  total: %d\n", stats.ActiveConns, stats.TotalConns)
		}
	}
}
