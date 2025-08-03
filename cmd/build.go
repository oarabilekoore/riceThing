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
	Desktop       string    `json:"desktop"`
	Packages      []Package `json:"packages"`
	ConfigFolders []string  `json:"configs"`
}

var system, desktop, homeDir string
var installedPackages, foldersToBundle []string

func init() {
	desktop = os.Getenv("XDG_SESSION_DESKTOP")
	homeDir = os.Getenv("HOME")
	rootCmd.AddCommand(buildCmd)
}

func getDistributionName() {
	file, err := os.Open(homeDir + "/etc/os-release")
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

func buildMetadataFile() {
	// Convert installedPackages from []string to []Package
	var parsedPackages []Package
	for _, pkg := range installedPackages {
		parts := strings.Fields(pkg)
		if len(parts) == 2 {
			parsedPackages = append(parsedPackages, Package{
				Name:    parts[0],
				Version: parts[1],
			})
		}
	}

	meta := riceThingMetadata{
		System:        system,
		Desktop:       desktop,
		Packages:      parsedPackages,
		ConfigFolders: foldersToBundle,
	}

	outFile := filepath.Join("ricemetadata.json")
	file, err := os.Create(outFile)
	if err != nil {
		fmt.Printf("Error creating JSON file: %s\n", err)
		return
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(meta); err != nil {
		fmt.Printf("Error encoding JSON: %s\n", err)
		return
	}

	fmt.Printf("✅ Metadata written to %s\n", outFile)
}

func bundleRice(cmd *cobra.Command, args []string) {
	folders, err := os.ReadDir(homeDir + "/.config")
	if err != nil {
		fmt.Printf("ERROR: Unable to read config folder:  %s", err)
		os.Exit(1)
	}

	for _, listOfFolders := range folders {
		if listOfFolders.IsDir() {
			foldersToBundle = append(foldersToBundle, listOfFolders.Name())
		}
	}

	getInstalledPackages()
	getDistributionName()
	buildMetadataFile()

	for _, folder := range foldersToBundle {
		src := filepath.Join(homeDir, ".config", folder)
		dest := filepath.Join(".", folder)

		if err := copyDir(src, dest); err != nil {
			fmt.Printf("❌ Failed to copy %s: %s\n", folder, err)
		} else {
			fmt.Printf("📦 Copied %s => %s\n", src, dest)
		}
	}

}

func copyDir(src, dest string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Create destination dir
	if err := os.MkdirAll(dest, 0755); err != nil {
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
