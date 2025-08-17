package main

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

//go:embed templates/*
var templates embed.FS

var (
	version = "1.0.0"
	auraDir = filepath.Join(os.Getenv("HOME"), ".aura")
	
	// ANSI color codes
	lightBlue = "\033[94m"
	reset     = "\033[0m"
	
	asciiTitle = lightBlue + `
    ___   _   _ ____    ___  
   / _ \ | | | |  _ \  / _ \ 
  | |_| || | | | |_) || |_| |
  |  _  || |_| |  _ < |  _  |
  |_| |_| \___/|_| \_\|_| |_|
` + reset
)

var rootCmd = &cobra.Command{
	Use:   "aura",
	Short: "Aura - Local HTTPS development proxy",
	Long:  asciiTitle + "\n     Local HTTPS development proxy\n\nAura provides a Docker-based reverse proxy with automatic HTTPS for local development using the .aura TLD.",
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Aura proxy system",
	Long:  `Sets up the Aura proxy system including loopback address, mkcert, and Docker configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🚀 Installing Aura proxy system...")
		
		// Create .aura directory
		if err := os.MkdirAll(auraDir, 0755); err != nil {
			return fmt.Errorf("failed to create aura directory: %w", err)
		}

		// Copy configuration files
		if err := copyConfigs(); err != nil {
			return fmt.Errorf("failed to copy configs: %w", err)
		}

		// Run setup script
		setupScript := filepath.Join(auraDir, "setup.sh")
		if err := runCommand("bash", setupScript); err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}

		fmt.Println("✅ Aura proxy installed successfully!")
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Start the proxy: aura start")
		fmt.Println("  2. Test it: open https://whoami.aura")
		return nil
	},
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start Aura proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🟢 Starting Aura proxy...")
		if err := runCommandInDir(auraDir, "docker", "compose", "up", "-d"); err != nil {
			return fmt.Errorf("failed to start proxy: %w", err)
		}
		fmt.Println("✅ Aura proxy started!")
		fmt.Println("   Test: https://whoami.aura")
		return nil
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop Aura proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🔴 Stopping Aura proxy...")
		if err := runCommandInDir(auraDir, "docker", "compose", "down"); err != nil {
			return fmt.Errorf("failed to stop proxy: %w", err)
		}
		fmt.Println("✅ Aura proxy stopped")
		return nil
	},
}

var certCmd = &cobra.Command{
	Use:   "cert [domain]",
	Short: "Generate certificate for a domain",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		domain := args[0]
		if !strings.HasSuffix(domain, ".aura") {
			domain = domain + ".aura"
		}

		fmt.Printf("🔐 Generating certificate for %s...\n", domain)
		
		certScript := filepath.Join(auraDir, "add-cert.sh")
		if err := runCommand("bash", certScript, domain); err != nil {
			return fmt.Errorf("failed to generate certificate: %w", err)
		}
		
		fmt.Printf("✅ Certificate generated for %s\n", domain)
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Aura proxy status",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🔍 Checking Aura proxy status...")
		
		// Check if containers are running
		output, err := exec.Command("docker", "ps", "--filter", "name=aura-", "--format", "table {{.Names}}\t{{.Status}}").Output()
		if err != nil {
			fmt.Println("❌ Aura proxy is not running")
			return nil
		}
		
		if len(output) > 0 {
			fmt.Println("✅ Aura proxy is running\n")
			fmt.Println(string(output))
		} else {
			fmt.Println("❌ Aura proxy is not running")
			fmt.Println("   Start with: aura start")
		}
		return nil
	},
}

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show Aura proxy logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		follow, _ := cmd.Flags().GetBool("follow")
		
		dockerArgs := []string{"logs"}
		if follow {
			dockerArgs = append(dockerArgs, "-f")
		}
		dockerArgs = append(dockerArgs, "aura-caddy")
		
		return runCommand("docker", dockerArgs...)
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall Aura proxy system",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("⚠️  This will remove Aura proxy and all certificates.")
		fmt.Print("Are you sure? (y/N): ")
		
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled")
			return nil
		}

		// Stop containers
		runCommandInDir(auraDir, "docker", "compose", "down", "-v")
		
		// Remove directory
		if err := os.RemoveAll(auraDir); err != nil {
			return fmt.Errorf("failed to remove aura directory: %w", err)
		}
		
		fmt.Println("✅ Aura proxy uninstalled")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(certCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(uninstallCmd)
	
	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output")
	
	rootCmd.Version = version
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func runCommandInDir(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func copyConfigs() error {
	// This will be implemented to copy embedded files
	// For now, we'll copy from the current directory
	files := []string{
		"docker-compose.yml",
		"docker-compose.example.yml",
		"setup.sh",
		"setup-loopback.sh",
		"setup-mkcert.sh",
		"add-cert.sh",
	}
	
	for _, file := range files {
		src := filepath.Join(".", file)
		dst := filepath.Join(auraDir, file)
		
		input, err := os.ReadFile(src)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", file, err)
		}
		
		if err := os.WriteFile(dst, input, 0755); err != nil {
			return fmt.Errorf("failed to write %s: %w", file, err)
		}
	}
	
	// Create directories
	os.MkdirAll(filepath.Join(auraDir, "certs", "domains"), 0755)
	
	return nil
}