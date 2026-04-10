package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/MrHalder/moor/internal/config"
	"github.com/MrHalder/moor/internal/envfile"
	"github.com/spf13/cobra"
)

var (
	reserveDesc    string
	reserveEnvFile string
	reserveFromEnv string
)

var reserveCmd = &cobra.Command{
	Use:   "reserve <port> <project>",
	Short: "Reserve a port for a project",
	Long:  "Reserve a port for a project, or use --from-env to auto-reserve from a .env file.",
	RunE:  runReserve,
}

func init() {
	reserveCmd.Flags().StringVar(&reserveDesc, "description", "", "description of what uses this port")
	reserveCmd.Flags().StringVar(&reserveEnvFile, "env-file", "", "path to .env file for this project")
	reserveCmd.Flags().StringVar(&reserveFromEnv, "from-env", "", "auto-reserve ports found in a .env file")
	rootCmd.AddCommand(reserveCmd)
}

func runReserve(cmd *cobra.Command, args []string) error {
	if reserveFromEnv != "" {
		return runReserveFromEnv(reserveFromEnv)
	}

	if len(args) != 2 {
		return fmt.Errorf("requires <port> <project> arguments (or use --from-env)")
	}

	port, err := parsePort(args[0])
	if err != nil {
		return err
	}
	project := args[1]

	if len(project) > 100 {
		return fmt.Errorf("project name must not exceed 100 characters")
	}
	if reserveDesc != "" && len(reserveDesc) > 256 {
		return fmt.Errorf("description must not exceed 256 characters")
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	existing := cfg.FindReservation(port)
	if existing != nil {
		fmt.Printf("Updating reservation for port %d (was: %s)\n", port, existing.Project)
	}

	r := config.Reservation{
		Port:        port,
		Project:     project,
		Description: reserveDesc,
		EnvFile:     reserveEnvFile,
	}

	cfg = cfg.AddReservation(r)

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Printf("Reserved port %d for '%s'\n", port, project)
	if reserveDesc != "" {
		fmt.Printf("  description: %s\n", reserveDesc)
	}
	return nil
}

func runReserveFromEnv(envPath string) error {
	ports, err := envfile.Parse(envPath)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", envPath, err)
	}

	if len(ports) == 0 {
		fmt.Printf("No port variables found in %s\n", envPath)
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	absPath, err := filepath.Abs(envPath)
	if err != nil {
		absPath = envPath
	}
	project := filepath.Base(filepath.Dir(absPath))

	for _, p := range ports {
		r := config.Reservation{
			Port:        p.Value,
			Project:     project,
			Description: fmt.Sprintf("from %s (%s)", filepath.Base(envPath), p.Key),
			EnvFile:     absPath,
		}
		cfg = cfg.AddReservation(r)
		fmt.Printf("Reserved port %d for '%s' (%s=%d)\n", p.Value, project, p.Key, p.Value)
	}

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	return nil
}
