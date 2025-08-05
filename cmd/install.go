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
	installCmd.Flags().BoolVarP(&skipConfigs, "skip-configs", "c", false, "skip copying config folders and dotfiles")
	rootCmd.AddCommand(installCmd)
}

func installThing(cmd *cobra.Command, args []string) {
	// ensure argument present (cobra.ExactArgs already enforces this, but this is defensive)
	if len(args) == 0 {
		fmt.Println("please provide a path to a riceThing folder or git uri")
		return
	}

	repoPath := args[0]

	// Try to stat the path provided. If it's not a directory this will error.
	info, err := os.Lstat(repoPath)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	if !info.IsDir() {
		fmt.Println("Provided path is not a directory.")
		return
	}

	// read metadata from repoPath
	metaPath := filepath.Join(repoPath, "ricemetadata.json")
	file, err := os.ReadFile(metaPath)
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
		fmt.Println("WARNING: Your desktop does not match the metadata's info — this may lead to a different experience than expected.")
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

	// Copy config folders and dotfiles unless skipped
	if !skipConfigs {
		// Copy config folders from repoPath/.config/<folder> => $HOME/.config/<folder>
		for _, folder := range parsedMetadata.ConfigFolders {
			src := filepath.Join(repoPath, ".config", folder)
			dst := filepath.Join(homeDir, ".config", folder)

			// If the source doesn't exist in the repo, warn and continue
			if _, err := os.Lstat(src); err != nil {
				fmt.Printf("⚠️  Skipping config %s — not found in repo at %s\n", folder, src)
				continue
			}

			if err := copyDir(src, dst); err != nil {
				fmt.Printf("❌ Failed to copy config %s: %s\n", folder, err)
			} else {
				fmt.Printf("📦 Copied config %s => %s\n", src, dst)
			}
		}

		// Also copy common shell/dotfiles from the repo root to the user's home
		dotfiles := []string{".bashrc", ".bash_profile", ".profile", ".zshrc", ".xprofile", ".xinitrc"}
		for _, f := range dotfiles {
			src := filepath.Join(repoPath, f)
			dst := filepath.Join(homeDir, f)

			info, err := os.Lstat(src)
			if err != nil {
				// not present in repo — skip silently
				continue
			}

			if info.IsDir() {
				if err := copyDir(src, dst); err != nil {
					fmt.Printf("❌ Failed to copy dotfile dir %s: %s\n", src, err)
				} else {
					fmt.Printf("📦 Copied directory %s => %s\n", src, dst)
				}
			} else {
				if err := copyFile(src, dst); err != nil {
					fmt.Printf("❌ Failed to copy dotfile %s: %s\n", src, err)
				} else {
					fmt.Printf("📦 Copied file %s => %s\n", src, dst)
				}
			}
		}
	} else {
		fmt.Println("Skipping copying config folders and dotfiles (--skip-configs)")
	}
}

