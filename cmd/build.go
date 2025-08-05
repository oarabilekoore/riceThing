package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/spf13/cobra"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	System        string    `json:"name"`
	Shell         string    `json:"shell"`
	Desktop       string    `json:"desktop"`
	Packages      []Package `json:"packages"`
	ConfigFolders []string  `json:"configs"`
}

var system, desktop, homeDir, shell string
var installedPackages, foldersToBundle []string

// flags
var (
	outDir         string
	noConfigs      bool
	noPackages     bool
	dotfilesCSV    string
	includeDotfiles bool
)

func init() {
	desktop = os.Getenv("XDG_SESSION_DESKTOP")
	homeDir = os.Getenv("HOME")
	shell = os.Getenv("SHELL")
	rootCmd.AddCommand(buildCmd)

	// flags
	buildCmd.Flags().StringVarP(&outDir, "out", "o", ".", "output directory for the bundle")
	buildCmd.Flags().BoolVar(&noConfigs, "no-configs", false, "do not copy ~/.config folders")
	buildCmd.Flags().BoolVar(&noPackages, "no-packages", false, "do not include installed packages in metadata")
	buildCmd.Flags().StringVar(&dotfilesCSV, "dotfiles", "", "comma-separated list of dotfiles to include (e.g. .bashrc,.profile)")
	buildCmd.Flags().BoolVar(&includeDotfiles, "include-dotfiles", false, "include a default set of common dotfiles (.bashrc, .profile)")
}

func getDistributionName() {
	// Read the system-wide /etc/os-release
	file, err := os.Open("/etc/os-release")
	if err != nil {
		fmt.Println("WARNING: Unable To Fetch Distribution Name")
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			system = strings.Trim(strings.SplitN(line, "=", 2)[1], `"`)
			return
		}
	}
}

func getInstalledPackages() {
	cmd := exec.Command("pacman", "-Q")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	if err := cmd.Start(); err != nil {
		panic(err)
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		installedPackages = append(installedPackages, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		panic(err)
	}

	if err := cmd.Wait(); err != nil {
		panic(err)
	}
}

func buildMetadataFile(meta riceThingMetadata) error {
	// Ensure output directory exists
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	outFile := filepath.Join(outDir, "ricemetadata.json")
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

func bundleRice(cmd *cobra.Command, args []string) {
	// collect list of ~/.config folders (only if we're copying configs)
	if !noConfigs {
		cfgPath := filepath.Join(homeDir, ".config")
		folders, err := os.ReadDir(cfgPath)
		if err != nil {
			fmt.Printf("ERROR: Unable to read config folder: %s\n", err)
			// continue — user might still want dotfiles or packages
		} else {
			for _, listOfFolders := range folders {
				if listOfFolders.IsDir() {
					foldersToBundle = append(foldersToBundle, listOfFolders.Name())
				}
			}
		}
	}

	// installed packages
	var parsedPackages []Package
	if !noPackages {
		getInstalledPackages()
		// Convert installedPackages from []string to []Package
		for _, pkg := range installedPackages {
			parts := strings.Fields(pkg)
			if len(parts) == 2 {
				parsedPackages = append(parsedPackages, Package{
					Name:    parts[0],
					Version: parts[1],
				})
			}
		}
		getDistributionName()
	} else {
		getDistributionName() // still useful even without packages
	}

	meta := riceThingMetadata{
		System:        system,
		Shell:         shell,
		Desktop:       desktop,
		Packages:      parsedPackages,
		ConfigFolders: foldersToBundle,
	}

	// write metadata file to outDir
	if err := buildMetadataFile(meta); err != nil {
		fmt.Printf("Error writing metadata: %s\n", err)
		return
	}

	// copy configs into outDir/.config/<folder>
	if !noConfigs {
		for _, folder := range foldersToBundle {
			src := filepath.Join(homeDir, ".config", folder)
			dest := filepath.Join(outDir, ".config", folder)

			if err := copyDir(src, dest); err != nil {
				fmt.Printf("❌ Failed to copy %s: %s\n", folder, err)
			} else {
				fmt.Printf("📦 Copied %s => %s\n", src, dest)
			}
		}
	} else {
		fmt.Println("Skipping ~/.config copy (--no-configs)")
	}

	// copy dotfiles if requested
	var dotfiles []string
	if includeDotfiles {
		// sensible defaults
		dotfiles = append(dotfiles, ".bashrc", ".profile", ".bash_profile", ".zshrc")
	}
	if dotfilesCSV != "" {
		for _, f := range strings.Split(dotfilesCSV, ",") {
			f = strings.TrimSpace(f)
			if f != "" {
				dotfiles = append(dotfiles, f)
			}
		}
	}
	// de-duplicate
	seen := map[string]bool{}
	uniqueDotfiles := []string{}
	for _, d := range dotfiles {
		if !seen[d] {
			seen[d] = true
			uniqueDotfiles = append(uniqueDotfiles, d)
		}
	}

	for _, f := range uniqueDotfiles {
		src := filepath.Join(homeDir, f)
		dest := filepath.Join(outDir, f)

		// if it's a directory, copy directory; else copy file
		info, err := os.Lstat(src)
		if err != nil {
			fmt.Printf("⚠️  Skipping dotfile %s (not found)\n", src)
			continue
		}
		if info.IsDir() {
			if err := copyDir(src, dest); err != nil {
				fmt.Printf("❌ Failed to copy dir %s: %s\n", src, err)
			} else {
				fmt.Printf("📦 Copied dir %s => %s\n", src, dest)
			}
		} else {
			if err := copyFile(src, dest); err != nil {
				fmt.Printf("❌ Failed to copy file %s: %s\n", src, err)
			} else {
				fmt.Printf("📦 Copied file %s => %s\n", src, dest)
			}
		}
	}

	// done
	fmt.Println("Build complete.")
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

