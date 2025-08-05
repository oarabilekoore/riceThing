package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	skipPkgs    bool
	skipConfigs bool
)

var installCmd = &cobra.Command{
	Use:   "install <path-or-repo>",
	Short: "install riceThing compatible repositories by directing towards a git uri or folder.",
	Args:  cobra.ExactArgs(1),
	Run:   installThing,
}

func init() {
	// flags: skip package installation, skip copying configs
	installCmd.Flags().BoolVarP(&skipPkgs, "skip-pkgs", "p", false, "skip installing packages")
	installCmd.Flags().BoolVarP(&skipConfigs, "skip-configs", "c", false, "skip copying config folders")
	rootCmd.AddCommand(installCmd)
}

func installThing(cmd *cobra.Command, args []string) {
	// ensure argument present (cobra.ExactArgs already enforces this, but this is defensive)
	if len(args) == 0 {
		fmt.Println("please provide a path to a riceThing folder or git uri")
		return
	}

	// Try to stat the path provided. If it's not a directory this will error.
	info, err := os.Lstat(args[0])
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	if info.IsDir() {
		file, err := os.ReadFile(filepath.Join(args[0], "ricemetadata.json"))
		if err != nil {
			fmt.Println("Error reading metadata file:", err)
			return
		}

		var parsedMetadata riceThingMetadata
		if err := json.Unmarshal(file, &parsedMetadata); err != nil {
			fmt.Println("Error parsing metadata file:", err)
			return
		}

		if parsedMetadata.Desktop != desktop {
			fmt.Println("riceThing cannot continue if your desktop doesn't match metadata info")
			return
		}

		// Install packages unless user requested to skip them
		if !skipPkgs {
			for _, pkg := range parsedMetadata.Packages {
				fmt.Printf("Installing package: %s\n", pkg.Name)

				pacmanCmd := exec.Command("sudo", "pacman", "-S", "--noconfirm", pkg.Name)
				pacmanCmd.Stdout = os.Stdout
				pacmanCmd.Stderr = os.Stderr
				pacmanCmd.Stdin = os.Stdin // needed for sudo password prompt

				if err := pacmanCmd.Run(); err != nil {
					fmt.Printf("Failed to install %s: %v\n", pkg.Name, err)
				}
			}
		} else {
			fmt.Println("Skipping package installation (--skip-pkgs)")
		}

		// Copy config folders unless skipped
		if !skipConfigs {
			for _, folder := range parsedMetadata.ConfigFolders {
				dst := filepath.Join(homeDir, folder)
				if err := copyDir(folder, dst); err != nil {
					fmt.Printf("❌ Failed to copy %s: %s\n", folder, err)
				} else {
					fmt.Printf("📦 Copied %s => %s\n", folder, dst)
				}
			}
		} else {
			fmt.Println("Skipping copying config folders (--skip-configs)")
		}
	} else {
		fmt.Println("Provided path is not a directory.")
	}
}
