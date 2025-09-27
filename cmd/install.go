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

	if parsedMetadata.Desktop != desktopEnvironment {
		fmt.Println("WARNING: Your desktop does not match the metadata's info — this may lead to a different experience than expected.")
	}

	// Install packages unless user requested to skip them
	if !skipPkgs {
		fmt.Printf("📦 Installing %d packages...\n", len(parsedMetadata.Packages))
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
		// Copy traditional .config folders from repoPath/.config/<folder> => $HOME/.config/<folder>
		configPath := filepath.Join(repoPath, ".config")
		if _, err := os.Stat(configPath); err == nil {
			entries, err := os.ReadDir(configPath)
			if err == nil {
				for _, entry := range entries {
					if entry.IsDir() {
						src := filepath.Join(configPath, entry.Name())
						dst := filepath.Join(homeDir, ".config", entry.Name())

						if err := copyDir(src, dst); err != nil {
							fmt.Printf("❌ Failed to copy config %s: %s\n", entry.Name(), err)
						} else {
							fmt.Printf("📁 Copied config %s => %s\n", src, dst)
						}
					}
				}
			}
		}

		// Copy extra folders/files from repoPath/extra/ to appropriate locations
		extraPath := filepath.Join(repoPath, "extra")
		if _, err := os.Stat(extraPath); err == nil {
			fmt.Println("📁 Found extra files/folders, installing to appropriate locations...")

			err := filepath.Walk(extraPath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				// Skip the extra directory itself
				if path == extraPath {
					return nil
				}

				// Get relative path from extra directory
				relPath, err := filepath.Rel(extraPath, path)
				if err != nil {
					return err
				}

				// Determine destination
				dst := filepath.Join(homeDir, relPath)

				// Create parent directory if it doesn't exist
				if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
					fmt.Printf("❌ Failed to create directory %s: %v\n", filepath.Dir(dst), err)
					return nil
				}

				if info.IsDir() {
					// Create directory if it doesn't exist
					if err := os.MkdirAll(dst, info.Mode()); err != nil {
						fmt.Printf("❌ Failed to create directory %s: %v\n", dst, err)
					} else {
						fmt.Printf("📁 Created directory %s\n", dst)
					}
				} else {
					// Copy file
					if err := copyFile(path, dst); err != nil {
						fmt.Printf("❌ Failed to copy file %s: %v\n", relPath, err)
					} else {
						fmt.Printf("📄 Copied file %s => %s\n", path, dst)
					}
				}

				return nil
			})

			if err != nil {
				fmt.Printf("Error walking extra directory: %v\n", err)
			}
		}

		// Copy common shell/dotfiles from the repo root to the user's home
		dotfiles := []string{".bashrc", ".bash_profile", ".profile", ".zshrc", ".xprofile", ".xinitrc", ".vimrc", ".tmux.conf"}
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
					fmt.Printf("📁 Copied directory %s => %s\n", src, dst)
				}
			} else {
				if err := copyFile(src, dst); err != nil {
					fmt.Printf("❌ Failed to copy dotfile %s: %s\n", src, err)
				} else {
					fmt.Printf("📄 Copied file %s => %s\n", src, dst)
				}
			}
		}

		fmt.Println("✅ Installation complete!")
	} else {
		fmt.Println("Skipping copying config folders and dotfiles (--skip-configs)")
	}
}
