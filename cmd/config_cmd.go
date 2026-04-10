package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/ashutosh/moor/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage moor configuration",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}
		fmt.Print(string(data))
		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print config file location",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(config.ConfigPath())
	},
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open config in $EDITOR",
	RunE: func(cmd *cobra.Command, args []string) error {
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}

		editorPath, err := exec.LookPath(editor)
		if err != nil {
			return fmt.Errorf("editor %q not found in PATH: %w", editor, err)
		}

		// Ensure config exists
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if err := config.Save(cfg); err != nil {
			return err
		}

		path := config.ConfigPath()
		c := exec.Command(editorPath, path)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

var configResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset config to defaults",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := config.Reset(); err != nil {
			return fmt.Errorf("resetting config: %w", err)
		}
		fmt.Printf("Config reset to defaults at %s\n", config.ConfigPath())
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configPathCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configResetCmd)
	rootCmd.AddCommand(configCmd)
}
