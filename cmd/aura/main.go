package main

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ivannovak/aura/pkg/logger"
	"github.com/ivannovak/aura/pkg/version"
)

//go:embed embed/docker-compose.yml embed/docker-compose.example.yml
//go:embed embed/setup.sh embed/setup-loopback.sh embed/setup-resolver.sh embed/setup-mkcert.sh
//go:embed embed/add-cert.sh embed/uninstall-resolver.sh embed/uninstall-loopback.sh
//go:embed embed/coredns/Corefile
var embeddedFS embed.FS

const (
	// Domain configuration
	auraTLD = ".aura"

	// Network configuration
	loopbackIP = "127.0.0.2"

	// Container names
	containerCaddy   = "aura-caddy"
	containerWhoami  = "aura-whoami"
	containerCoredns = "aura-coredns"
	containerPrefix  = "aura-" // Prefix for filtering containers

	// Docker resources
	networkName       = "aura-proxy"
	volumeCaddyData   = "aura_caddy_data"
	volumeCaddyConfig = "aura_caddy_config"

	// File permissions
	filePermScript = 0600
	dirPermDefault = 0755

	// Directory names
	dirCerts        = "certs"
	dirCertsDomains = "certs/domains"
	dirCoredns      = "coredns"

	// Filenames
	fileCorefile = "Corefile"
)

var (
	domainLabelRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

	auraDir string

	// Log configuration flags
	logLevel  string
	logFormat string

	// ANSI color codes
	teal  = "\033[96m"
	reset = "\033[0m"

	asciiTitle = teal + `
  █████╗ ██╗   ██╗██████╗  █████╗
 ██╔══██╗██║   ██║██╔══██╗██╔══██╗
 ███████║██║   ██║██████╔╝███████║
 ██╔══██║██║   ██║██╔══██╗██╔══██║
 ██║  ██║╚██████╔╝██║  ██║██║  ██║
 ╚═╝  ╚═╝ ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═╝
` + reset
)

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Unable to determine home directory: %v\n", err)
		os.Exit(1)
	}
	auraDir = filepath.Join(home, ".aura")
}

var rootCmd = &cobra.Command{
	Use:   "aura",
	Short: "Aura - Local HTTPS development proxy",
	Long: asciiTitle + "\n     Local HTTPS development proxy\n\n" +
		"Aura provides a Docker-based reverse proxy with automatic HTTPS for local development using the .aura TLD.",
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install Aura proxy system",
	Long:  `Sets up the Aura proxy system including loopback address, mkcert, and Docker configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		logger.Info("Installing Aura proxy system")
		fmt.Println("🚀 Installing Aura proxy system...")

		// Create .aura directory
		logger.Debug("Creating aura directory", "path", auraDir)
		if err := os.MkdirAll(auraDir, dirPermDefault); err != nil {
			return fmt.Errorf("failed to create aura directory: %w", err)
		}

		// Copy configuration files
		logger.Info("Copying configuration files")
		if err := copyConfigs(); err != nil {
			return fmt.Errorf("failed to copy configs: %w", err)
		}

		// Run setup script
		setupScript := filepath.Join(auraDir, "setup.sh")
		logger.Info("Running setup script", "script", setupScript)
		if err := runCommand("bash", setupScript); err != nil {
			return fmt.Errorf("setup failed: %w", err)
		}

		logger.Info("Aura proxy installed successfully")
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
		logger.Info("Starting Aura proxy")
		fmt.Println("🟢 Starting Aura proxy...")

		if err := runCommandInDir(auraDir, "docker", "compose", "up", "-d"); err != nil {
			return fmt.Errorf("failed to start proxy: %w", err)
		}

		logger.Info("Aura proxy started successfully")
		fmt.Println("✅ Aura proxy started!")
		fmt.Println("   Test: https://whoami.aura")
		return nil
	},
}

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop Aura proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger.Info("Stopping Aura proxy")
		fmt.Println("🔴 Stopping Aura proxy...")

		if err := runCommandInDir(auraDir, "docker", "compose", "down"); err != nil {
			return fmt.Errorf("failed to stop proxy: %w", err)
		}

		logger.Info("Aura proxy stopped successfully")
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
		if !strings.HasSuffix(domain, auraTLD) {
			domain += auraTLD
		}

		logger.Debug("Validating domain", "domain", domain)
		if err := validateDomain(domain); err != nil {
			return fmt.Errorf("invalid domain: %w", err)
		}

		logger.Info("Generating certificate", "domain", domain)
		fmt.Printf("🔐 Generating certificate for %s...\n", domain)

		certScript := filepath.Join(auraDir, "add-cert.sh")
		if err := runCommand("bash", certScript, domain); err != nil {
			return fmt.Errorf("failed to generate certificate: %w", err)
		}

		logger.Info("Certificate generated successfully", "domain", domain)
		fmt.Printf("✅ Certificate generated for %s\n", domain)
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show Aura proxy status",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger.Info("Checking Aura proxy status")
		fmt.Println("🔍 Checking Aura proxy status...")

		// Check if containers are running
		filter := fmt.Sprintf("name=%s", containerPrefix)
		output, err := exec.Command("docker", "ps", "--filter", filter, "--format", "table {{.Names}}\t{{.Status}}").Output()
		if err != nil {
			logger.Warn("Failed to query Docker", "error", err)
			fmt.Println("❌ Aura proxy is not running")
			return nil
		}

		if len(output) > 0 {
			logger.Debug("Found running containers", "count", len(strings.Split(string(output), "\n"))-1)
			fmt.Println("✅ Aura proxy is running")
			fmt.Println(string(output))
		} else {
			logger.Info("No Aura containers running")
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
		dockerArgs = append(dockerArgs, containerCaddy)

		return runCommand("docker", dockerArgs...)
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall Aura proxy system",
	Long:  `Removes Aura proxy system including DNS configuration, loopback address, and all files.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("⚠️  This will completely remove Aura proxy:")
		fmt.Println("   - Stop and remove Docker containers")
		fmt.Println("   - Remove DNS resolver configuration")
		fmt.Printf("   - Remove loopback address (%s)\n", loopbackIP)
		fmt.Println("   - Remove all certificates")
		fmt.Println("   - Remove ~/.aura directory")
		fmt.Print("\nAre you sure? (y/N): ")

		var response string
		if _, err := fmt.Scanln(&response); err != nil {
			fmt.Println("Canceled")
			return nil
		}
		if response != "y" && response != "Y" {
			fmt.Println("Canceled")
			return nil
		}

		// Stop containers and remove volumes
		fmt.Println("\n🧹 Stopping containers...")
		if err := runCommandInDir(auraDir, "docker", "compose", "down", "-v"); err != nil {
			fmt.Printf("Warning: failed to stop containers: %v\n", err)
		}

		// Remove DNS resolver configuration
		uninstallResolverScript := filepath.Join(auraDir, "uninstall-resolver.sh")
		if _, err := os.Stat(uninstallResolverScript); err == nil {
			fmt.Println("🧹 Removing DNS resolver configuration...")
			if err := runCommand("bash", uninstallResolverScript); err != nil {
				fmt.Printf("Warning: failed to cleanup DNS resolver: %v\n", err)
			}
		}

		// Remove loopback address configuration
		uninstallLoopbackScript := filepath.Join(auraDir, "uninstall-loopback.sh")
		if _, err := os.Stat(uninstallLoopbackScript); err == nil {
			fmt.Println("🧹 Removing loopback address configuration...")
			if err := runCommand("bash", uninstallLoopbackScript); err != nil {
				fmt.Printf("Warning: failed to cleanup loopback address: %v\n", err)
			}
		}

		// Remove Docker network
		fmt.Println("🧹 Removing Docker network...")
		if err := runCommand("docker", "network", "rm", networkName); err != nil {
			// Network might not exist, that's okay
			fmt.Println("Note: Docker network already removed or doesn't exist")
		}

		// Remove Docker volumes
		fmt.Println("🧹 Removing Docker volumes...")
		_ = runCommand("docker", "volume", "rm", volumeCaddyData)
		_ = runCommand("docker", "volume", "rm", volumeCaddyConfig)

		// Remove directory
		fmt.Println("🧹 Removing ~/.aura directory...")
		if err := os.RemoveAll(auraDir); err != nil {
			return fmt.Errorf("failed to remove aura directory: %w", err)
		}

		fmt.Println("\n✅ Aura proxy completely uninstalled!")
		fmt.Println("\nOptional: To remove the CLI binary:")
		fmt.Println("  sudo rm /usr/local/bin/aura")
		return nil
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		verbose, _ := cmd.Flags().GetBool("verbose")

		if verbose {
			fmt.Printf("Aura v%s\n", version.Version)
			fmt.Printf("  Git Commit: %s\n", version.GitCommit)
			fmt.Printf("  Build Date: %s\n", version.BuildDate)
			fmt.Printf("  Go Version: %s\n", version.GoVersion)
			fmt.Printf("  Platform:   %s\n", version.Platform)
		} else {
			fmt.Printf("aura version %s\n", version.Version)
		}
	},
}

//nolint:gochecknoinits // init required for cobra command registration
func init() {
	// Add global logging flags
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info",
		"Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text",
		"Log format (text, json)")

	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(certCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(uninstallCmd)
	rootCmd.AddCommand(versionCmd)

	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output")
	versionCmd.Flags().BoolP("verbose", "v", false, "Show detailed version information")

	rootCmd.Version = version.Version
}

func main() {
	// Initialize logger before executing commands
	cobra.OnInitialize(initLogger)

	if err := rootCmd.Execute(); err != nil {
		logger.Error("Command failed", "error", err)
		os.Exit(1)
	}
}

func initLogger() {
	level := logger.LevelInfo
	switch logLevel {
	case "debug":
		level = logger.LevelDebug
	case "info":
		level = logger.LevelInfo
	case "warn":
		level = logger.LevelWarn
	case "error":
		level = logger.LevelError
	}

	logger.Init(logger.Config{
		Level:  level,
		Format: logFormat,
		Output: os.Stdout,
	})
}

func runCommand(name string, args ...string) error {
	logger.Debug("Executing command", "name", name, "args", args)

	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		logger.Error("Command failed", "name", name, "args", args, "error", err)
		return err
	}

	logger.Debug("Command completed", "name", name)
	return nil
}

func runCommandInDir(dir, name string, args ...string) error {
	logger.Debug("Executing command in directory", "dir", dir, "name", name, "args", args)

	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		logger.Error("Command failed", "dir", dir, "name", name, "args", args, "error", err)
		return err
	}

	logger.Debug("Command completed", "name", name)
	return nil
}

func validateDomain(domain string) error {
	baseDomain := strings.TrimSuffix(domain, auraTLD)

	if len(domain) > 253 {
		return fmt.Errorf("domain name too long (max 253 characters)")
	}

	if baseDomain == "" {
		return fmt.Errorf("domain name cannot be empty")
	}

	if strings.Contains(domain, "..") {
		return fmt.Errorf("invalid domain: contains path traversal")
	}

	if strings.Contains(domain, "\x00") {
		return fmt.Errorf("invalid domain: contains null byte")
	}

	labels := strings.Split(baseDomain, ".")
	for _, label := range labels {
		if len(label) == 0 {
			return fmt.Errorf("invalid domain: empty label")
		}
		if len(label) > 63 {
			return fmt.Errorf("invalid domain: label exceeds 63 characters")
		}
		if !domainLabelRegex.MatchString(label) {
			return fmt.Errorf("invalid domain label '%s': must contain only lowercase letters, numbers, and hyphens", label)
		}
	}

	return nil
}

func copyConfigs() error {
	files := []string{
		"docker-compose.yml",
		"docker-compose.example.yml",
		"setup.sh",
		"setup-loopback.sh",
		"setup-resolver.sh",
		"setup-mkcert.sh",
		"add-cert.sh",
		"uninstall-resolver.sh",
		"uninstall-loopback.sh",
	}

	for _, file := range files {
		data, err := embeddedFS.ReadFile("embed/" + file)
		if err != nil {
			return fmt.Errorf("failed to read embedded %s: %w", file, err)
		}

		dst := filepath.Join(auraDir, file)
		if err := os.WriteFile(dst, data, filePermScript); err != nil {
			return fmt.Errorf("failed to write %s: %w", file, err)
		}
	}

	// Create directories
	if err := os.MkdirAll(filepath.Join(auraDir, dirCertsDomains), dirPermDefault); err != nil {
		return fmt.Errorf("failed to create certs directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(auraDir, dirCoredns), dirPermDefault); err != nil {
		return fmt.Errorf("failed to create coredns directory: %w", err)
	}

	// Copy CoreDNS configuration
	corefilePath := filepath.Join("embed", dirCoredns, fileCorefile)
	corefileData, err := embeddedFS.ReadFile(corefilePath)
	if err != nil {
		return fmt.Errorf("failed to read embedded Corefile: %w", err)
	}

	corefileDst := filepath.Join(auraDir, dirCoredns, fileCorefile)
	if err := os.WriteFile(corefileDst, corefileData, filePermScript); err != nil {
		return fmt.Errorf("failed to write Corefile: %w", err)
	}

	return nil
}
