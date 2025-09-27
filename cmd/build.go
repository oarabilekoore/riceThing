package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "build a rice configuration you can share and use later.",
	Run:   bundleRice,
}

type Package struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type riceThingMetadata struct {
	DistributionName string    `json:"distro"`
	Shell            string    `json:"shell"`
	Desktop          string    `json:"de"`
	Packages         []Package `json:"packages"`
	ConfigFolders    []string  `json:"configs"`
}

var distributionName, desktopEnvironment, homeDir, shell string
var bundleAll bool
var outDir string

func init() {
	desktopEnvironment = os.Getenv("XDG_SESSION_DESKTOP")
	homeDir = os.Getenv("HOME")
	shell = os.Getenv("SHELL")

	// Add flags
	buildCmd.Flags().BoolVarP(&bundleAll, "all", "a", false, "Bundle all config folders and all installed packages")
	buildCmd.Flags().StringVarP(&outDir, "out", "o", ".", "Output directory for the rice bundle")

	rootCmd.AddCommand(buildCmd)

	file, err := os.Open("/etc/os-release") //get the distribution name
	if err != nil {
		fmt.Println("WARNING: Unable To Fetch Distribution Name")
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			distributionName = strings.Trim(strings.SplitN(line, "=", 2)[1], `"`)
			return
		}
	}
}

func checkOrCreateRiceThingConfig() ([]string, []string, []string, error) {
	riceThingDir := filepath.Join(homeDir, ".config", "ricething")
	includeFile := filepath.Join(riceThingDir, "include")
	packagesFile := filepath.Join(riceThingDir, "packages")

	// Check if ricething directory exists
	if _, err := os.Stat(riceThingDir); os.IsNotExist(err) {
		// Create the directory
		if err := os.MkdirAll(riceThingDir, 0755); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create ricething config directory: %v", err)
		}

		// Create include file with example content
		if err := createExampleIncludeFile(includeFile); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create include file: %v", err)
		}

		// Create packages file with example content
		if err := createExamplePackagesFile(packagesFile); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create packages file: %v", err)
		}

		fmt.Printf("📁 Created ricething config directory at: %s\n", riceThingDir)
		fmt.Printf("📝 Please populate the following files:\n")
		fmt.Printf("   • %s - List folders/files to include (one per line)\n", includeFile)
		fmt.Printf("   • %s - List packages to include (one per line)\n", packagesFile)
		fmt.Printf("   • The include file supports paths relative to home directory\n")
		fmt.Printf("   • Use 'DOTFILES' entry to include common dotfiles\n")
		fmt.Printf("   • Use -a flag to bundle everything automatically\n")
		fmt.Printf("\nRun the build command again after populating these files.\n")
		os.Exit(0)
	}

	// Read include file
	var folders []string
	var dotfiles []string
	if _, err := os.Stat(includeFile); err == nil {
		lines, err := readLinesFromFile(includeFile)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to read include file: %v", err)
		}

		for _, line := range lines {
			if line == "DOTFILES" {
				// Add common dotfiles
				dotfiles = append(dotfiles, ".bashrc", ".profile", ".bash_profile", ".zshrc", ".xprofile", ".xinitrc")
			} else if strings.HasPrefix(line, ".") && !strings.Contains(line, "/") {
				dotfiles = append(dotfiles, line)
			} else {
				folders = append(folders, line)
			}
		}

		// IMPORTANT: If include file exists but is empty (after removing comments),
		// return empty slices - don't fall back to bundling everything
		if len(lines) == 0 {
			fmt.Println("📝 Include file is empty - no configs will be bundled")
			fmt.Println("   Add entries to ~/.config/ricething/include or use -a flag to bundle everything")
		}

	} else {
		fmt.Printf("⚠️  Include file not found at %s, creating example file\n", includeFile)
		if err := createExampleIncludeFile(includeFile); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create include file: %v", err)
		}
		fmt.Println("📝 Please populate the include file and run build again")
		// Return empty slices, don't bundle everything by default
		return []string{}, []string{}, []string{}, nil
	}

	var customPackages []string
	if _, err := os.Stat(packagesFile); err == nil {
		customPackages, err = readLinesFromFile(packagesFile)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to read packages file: %v", err)
		}
	} else {
		fmt.Printf("⚠️  Packages file not found at %s, creating example file\n", packagesFile)
		if err := createExamplePackagesFile(packagesFile); err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create packages file: %v", err)
		}
		// Return empty packages list if file doesn't exist
	}

	return folders, customPackages, dotfiles, nil
}

func createExampleIncludeFile(filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	content := `# List folders and files to include (paths relative to home directory)
# Each line should contain either:
# - A path relative to home (e.g., .config/awesome, Pictures/wallpapers)
# - A dotfile name (e.g., .bashrc, .vimrc)
# - The special keyword 'DOTFILES' to include common dotfiles

# Example entries:
# .config/awesome
# .config/kitty
# .config/rofi
# Pictures/wallpapers
# Documents/scripts
# DOTFILES
# .vimrc
# .tmux.conf
`
	_, err = file.WriteString(content)
	return err
}

func createExamplePackagesFile(filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	content := `# List packages to include in the rice bundle
# Each line should contain a package name

# Example entries:
# firefox
# discord
# neovim
# awesome
# kitty
`
	_, err = file.WriteString(content)
	return err
}

func readLinesFromFile(filepath string) ([]string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") { // Skip empty lines and comments
			lines = append(lines, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func getInstalledPackages() ([]Package, error) {
	cmd := exec.Command("pacman", "-Q")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get installed packages: %v", err)
	}

	var packages []Package
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			packages = append(packages, Package{
				Name:    parts[0],
				Version: parts[1],
			})
		}
	}

	return packages, nil
}

func buildMetadataFile(meta riceThingMetadata, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil { // ensure output directory exists
		return err
	}

	outFile := filepath.Join(outDir, "ricemetadata.json") //create metadata file
	file, err := os.Create(outFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(meta); err != nil {
		return err
	}

	fmt.Printf("✅ Metadata written to %s\n", outFile)
	return nil
}

func getAllConfigFolders() ([]string, error) {
	configDir := filepath.Join(homeDir, ".config")
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return nil, err
	}

	var folders []string
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "ricething" {
			// Add as .config/foldername for consistency with new path format
			folders = append(folders, filepath.Join(".config", entry.Name()))
		}
	}
	return folders, nil
}

func bundleRice(cmd *cobra.Command, args []string) {
	// Output directory from flag (defaults to current directory)
	var folders []string
	var customPackages []string
	var dotfiles []string
	var allPackages []Package
	var err error

	if bundleAll {
		// Bundle everything mode
		fmt.Println("🚀 Bundle all mode: including all config folders and installed packages")

		// Get all config folders
		folders, err = getAllConfigFolders()
		if err != nil {
			fmt.Printf("Warning: Failed to get config folders: %v\n", err)
			folders = []string{}
		}

		// Include common dotfiles
		dotfiles = []string{".bashrc", ".profile", ".bash_profile", ".zshrc", ".xprofile", ".xinitrc"}

		// Get all installed packages
		allPackages, err = getInstalledPackages()
		if err != nil {
			fmt.Printf("Warning: Failed to get installed packages: %v\n", err)
			allPackages = []Package{}
		}

	} else {
		// Use ricething config files
		folders, customPackages, dotfiles, err = checkOrCreateRiceThingConfig()
		if err != nil {
			fmt.Printf("Error with ricething config: %v\n", err)
			return
		}

		// Only use packages from packages file (don't get all installed packages)
		if len(customPackages) > 0 {
			for _, pkgName := range customPackages {
				// Try to get the version if package is installed
				cmd := exec.Command("pacman", "-Q", pkgName)
				if output, err := cmd.Output(); err == nil {
					parts := strings.Fields(strings.TrimSpace(string(output)))
					if len(parts) >= 2 {
						allPackages = append(allPackages, Package{
							Name:    parts[0],
							Version: parts[1],
						})
					}
				} else {
					// Package not installed, add without version
					allPackages = append(allPackages, Package{
						Name:    pkgName,
						Version: "unknown",
					})
				}
			}
		}
	}

	fmt.Printf("📦 Folders/files to include: %v\n", folders)
	if len(dotfiles) > 0 {
		fmt.Printf("📦 Dotfiles to include: %v\n", dotfiles)
	}
	fmt.Printf("📦 Total packages to include: %d\n", len(allPackages))

	// Extract just the folder names for metadata (remove path prefixes for compatibility)
	var metadataFolders []string
	for _, folder := range folders {
		if strings.HasPrefix(folder, ".config/") {
			// For .config folders, just store the folder name
			metadataFolders = append(metadataFolders, strings.TrimPrefix(folder, ".config/"))
		} else {
			// For other paths, store the full path
			metadataFolders = append(metadataFolders, folder)
		}
	}

	meta := riceThingMetadata{
		DistributionName: distributionName,
		Shell:            shell,
		Desktop:          desktopEnvironment,
		Packages:         allPackages,
		ConfigFolders:    metadataFolders,
	}

	// write metadata file to outDir
	if err := buildMetadataFile(meta, outDir); err != nil {
		fmt.Printf("Error writing metadata: %s\n", err)
		return
	}

	// Copy folders/files
	for _, folder := range folders {
		src := filepath.Join(homeDir, folder)

		// Determine destination based on source path
		var dest string
		if strings.HasPrefix(folder, ".config/") {
			// For .config folders, maintain the .config structure in output
			dest = filepath.Join(outDir, folder)
		} else {
			// For other paths, place them in a special directory to be sorted by install
			dest = filepath.Join(outDir, "extra", folder)
		}

		// Check if source exists
		if _, err := os.Stat(src); os.IsNotExist(err) {
			fmt.Printf("⚠️  Skipping %s - not found\n", folder)
			continue
		}

		// Check if it's a directory or file
		info, err := os.Stat(src)
		if err != nil {
			fmt.Printf("⚠️  Skipping %s - cannot stat: %v\n", folder, err)
			continue
		}

		if info.IsDir() {
			if err := copyDir(src, dest); err != nil {
				fmt.Printf("❌ Failed to copy directory %s: %s\n", folder, err)
			} else {
				fmt.Printf("📁 Copied directory %s => %s\n", src, dest)
			}
		} else {
			if err := copyFile(src, dest); err != nil {
				fmt.Printf("❌ Failed to copy file %s: %s\n", folder, err)
			} else {
				fmt.Printf("📄 Copied file %s => %s\n", src, dest)
			}
		}
	}

	// Copy dotfiles to root of bundle
	for _, dotfile := range dotfiles {
		src := filepath.Join(homeDir, dotfile)
		dest := filepath.Join(outDir, dotfile)

		if _, err := os.Stat(src); os.IsNotExist(err) {
			fmt.Printf("⚠️  Skipping %s - file not found in home directory\n", dotfile)
			continue
		}

		if err := copyFile(src, dest); err != nil {
			fmt.Printf("❌ Failed to copy %s: %s\n", dotfile, err)
		} else {
			fmt.Printf("📄 Copied %s => %s\n", src, dest)
		}
	}

	fmt.Println("✅ Build complete.")
}

func copyDir(src, dest string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Create destination dir
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, destPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, destPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(srcFile, destFile string) error {
	// ensure destination folder exists
	if err := os.MkdirAll(filepath.Dir(destFile), 0o755); err != nil {
		return err
	}

	src, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer src.Close()

	dest, err := os.Create(destFile)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, src)
	return err
}
