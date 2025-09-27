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
	Name string `json:"name"`
}

type riceThingMetadata struct {
	DistributionName        string    `json:"distro"`
	Shell                   string    `json:"shell"`
	Desktop       string    `json:"de"`
	Packages      []Package `json:"packages"`
	ConfigFolders []string  `json:"configs"`
}


var distributionName, desktopEnvironment, homeDir, shell string
var installedPackages, foldersToBundle []string


func init() {
	desktopEnvironment = os.Getenv("XDG_SESSION_DESKTOP")
	homeDir = os.Getenv("HOME")
	shell = os.Getenv("SHELL")
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


func buildMetadataFile(meta riceThingMetadata) error {
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

func bundleRice(cmd *cobra.Command, args []string) {

	var parsedPackages []Package


	// Convert installedPackages from []string to []Package
	for _, pkg := range installedPackages {
		parts := strings.Fields(pkg)
		if len(parts) == 2 {
			parsedPackages = append(parsedPackages, Package{
				Name:    parts[0],
				Version: parts[1],
			})
		}

	meta := riceThingMetadata{
		distributionName:        distributionName,
		Shell:         shell,
		Desktop:       desktopEnvironment,
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
