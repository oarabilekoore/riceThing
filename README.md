# riceThing Documentation

> **riceThing** – Instantly package your Linux desktop configuration and installed packages into a portable, shareable bundle.

riceThing is a command-line tool that allows Linux users to easily package their desktop "rice" (customized desktop environment) into a shareable format and install rice configurations from others.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Commands](#commands)
  - [build](#build-command)
  - [install](#install-command)
- [Configuration](#configuration)
- [File Structure](#file-structure)
- [Examples](#examples)
- [Troubleshooting](#troubleshooting)
- [Contributing](#contributing)

## Installation

### Prerequisites

- Linux system with `pacman` package manager (Arch Linux, Manjaro, etc.)
- Go 1.16+ (for building from source)
- `sudo` privileges (for package installation during install command)

### From Source

```bash
git clone https://github.com/yourusername/riceThing.git
cd riceThing
go build -o ricething
sudo mv ricething /usr/local/bin/
```

## Quick Start

### Package Your Rice

Create a shareable package of your current desktop configuration:

```bash
# First run creates configuration files
ricething build -o my-rice-bundle

# Bundle everything automatically (all configs + all packages)
ricething build -a -o my-rice-bundle
```

### Install Someone's Rice

Install a rice configuration from a local directory:

```bash
ricething install ./downloaded-rice-bundle
```

## Commands

### `build` Command

Package your current Linux desktop configuration into a shareable bundle.

```bash
ricething build [flags]
```

#### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--out` | `-o` | Output directory for the bundle | `.` (current directory) |
| `--all` | `-a` | Bundle all config folders and all installed packages | `false` |

#### Configuration-Based Building

On first run, riceThing creates configuration files in `~/.config/ricething/`:

- **`include`** - Lists folders/files to include (one per line)
- **`packages`** - Lists specific packages to include (one per line)

The `include` file supports:
- Paths relative to home directory (e.g., `.config/awesome`, `Pictures/wallpapers`)
- Individual dotfiles (e.g., `.bashrc`, `.vimrc`)
- The special keyword `DOTFILES` to include common shell files

#### Bundle All Mode (`-a` flag)

When using `--all`, riceThing bypasses configuration files and includes:
- All directories in `~/.config/` (except `ricething`)
- All installed packages from `pacman -Q`
- Common dotfiles (`.bashrc`, `.profile`, `.bash_profile`, `.zshrc`, `.xprofile`, `.xinitrc`)

#### What Gets Packaged

- **System metadata**: Distribution name, desktop environment, shell
- **Packages**: From packages file or all installed (with `--all`)
- **Config folders**: From include file or all `.config` folders (with `--all`)
- **Dotfiles**: Specified in include file or common ones (with `--all`)
- **Extra files**: Any non-config paths specified in include file

#### Examples

```bash
# First run - creates config files and exits
ricething build

# Build using configuration files
ricething build -o ~/my-rice

# Bundle everything automatically
ricething build -a -o ~/complete-rice

# Build to current directory using config files
ricething build
```

### `install` Command

Install a rice configuration from a local directory.

```bash
ricething install <path> [flags]
```

#### Arguments

- `<path>`: Path to the rice bundle directory

#### Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--skip-pkgs` | `-p` | Skip installing packages | `false` |
| `--skip-configs` | `-c` | Skip copying config folders and dotfiles | `false` |

#### Installation Process

1. **Packages**: Installs packages using `sudo pacman -S --noconfirm`
2. **Config folders**: Copies from `bundle/.config/` to `~/.config/`
3. **Extra files**: Copies from `bundle/extra/` to appropriate home locations
4. **Dotfiles**: Copies common shell files from bundle root to home directory

#### Compatibility Checks

- **Desktop environment mismatch**: Shows warning but continues installation
- **Missing files**: Warns about missing files but continues
- **File conflicts**: Overwrites existing files without backup

#### Examples

```bash
# Full installation (packages + configs)
ricething install ./awesome-rice-bundle

# Install configs only (skip packages)
ricething install ./rice-bundle --skip-pkgs

# Install packages only (skip configs)
ricething install ./rice-bundle --skip-configs
```

## Configuration

### System Detection

riceThing automatically detects:
- **Distribution**: From `/etc/os-release` (ID field)
- **Desktop Environment**: From `$XDG_SESSION_DESKTOP`
- **Shell**: From `$SHELL` environment variable
- **Home Directory**: From `$HOME` environment variable

### riceThing Configuration Files

Located in `~/.config/ricething/`:

#### `include` file format:
```
# Comments start with #
.config/awesome
.config/kitty
Pictures/wallpapers
Documents/scripts
DOTFILES
.vimrc
.tmux.conf
```

#### `packages` file format:
```
# Comments start with #
firefox
discord
neovim
awesome
kitty
```

## File Structure

A riceThing bundle contains:

```
rice-bundle/
├── ricemetadata.json          # System and package metadata
├── .config/                   # Desktop configuration folders
│   ├── awesome/              # Window manager configs
│   ├── kitty/                # Terminal configs
│   └── ...
├── extra/                     # Non-config files (optional)
│   ├── Pictures/wallpapers/  # Custom paths from include file
│   └── Documents/scripts/
├── .bashrc                   # Dotfiles (if included)
├── .zshrc
├── .profile
├── .xinitrc
└── ...
```

### Metadata Format

The `ricemetadata.json` file contains:

```json
{
  "distro": "arch",
  "shell": "/bin/bash",
  "de": "awesome",
  "packages": [
    {
      "name": "firefox",
      "version": "119.0.1-1"
    },
    {
      "name": "custom-package",
      "version": "unknown"
    }
  ],
  "configs": [
    "awesome",
    "kitty",
    "rofi"
  ]
}
```

## Examples

### Complete Rice Workflow

1. **Customize your desktop** with preferred tools and themes

2. **Configure riceThing** (on first run):
   ```bash
   ricething build  # Creates config files
   # Edit ~/.config/ricething/include and ~/.config/ricething/packages
   ```

3. **Package your rice**:
   ```bash
   ricething build -o ~/my-awesome-rice
   ```

4. **Share your bundle** (upload to GitHub, etc.)

5. **Others can install**:
   ```bash
   git clone https://github.com/user/awesome-rice
   ricething install ./awesome-rice
   ```

### Bundle Everything Quickly

```bash
# Skip configuration, include everything
ricething build -a -o ~/complete-system-rice
```

### Selective Installation

```bash
# Test configs first
ricething install ./rice-bundle --skip-pkgs

# If satisfied, install packages
ricething install ./rice-bundle --skip-configs
```

### Custom Includes Example

Create `~/.config/ricething/include`:
```
.config/awesome
.config/kitty
.config/rofi
Pictures/wallpapers
Documents/scripts
DOTFILES
.vimrc
.tmux.conf
```

Create `~/.config/ricething/packages`:
```
firefox
discord
neovim
awesome
kitty-git
```

## Troubleshooting

### Common Issues

**"Unable To Fetch Distribution Name"**
- Ensure `/etc/os-release` exists and is readable
- riceThing continues without this info but shows a warning

**Package Installation Fails**
- Verify you have `sudo` privileges
- Check if packages exist in your repositories
- Some packages might have different names or be unavailable
- Packages marked as "unknown" version aren't installed on build system

**Config Copy Failures**
- Check file permissions in `~/.config`
- Ensure sufficient disk space
- Close applications that might be using config files

**Desktop Environment Mismatch**
- riceThing shows a warning but continues installation
- Different desktop environments may not be compatible
- Consider installing configs only first to test

**Configuration Files Not Found**
- Run `ricething build` once to create example configuration files
- Edit `~/.config/ricething/include` and `~/.config/ricething/packages`
- Use `--all` flag to skip configuration files entirely

### Current Limitations

- **Arch Linux only**: Currently hardcoded for `pacman` package manager
- **No git support**: Install command doesn't handle git URLs
- **No automatic backups**: Overwrites existing configs without backup
- **Path restrictions**: Only supports paths relative to home directory
- **No AUR support**: Only official repository packages

### Debug Tips

1. **Verify bundle structure**: Check that `ricemetadata.json` exists
2. **Test selective install**: Use `--skip-pkgs` or `--skip-configs` to isolate issues
3. **Check package names**: Ensure packages in bundle exist in your repositories
4. **File permissions**: Ensure write access to `~/.config` and home directory

### Getting Help

```bash
ricething --help
ricething build --help
ricething install --help
```

## Contributing

riceThing is open source under the MIT License. Contributions welcome!

### Potential Improvements

- Support for other package managers (apt, dnf, etc.)
- Git repository support for install command
- Automatic backup creation
- AUR package support
- Cross-distribution compatibility
- Package dependency resolution
- Selective file restoration

## License

MIT License - see LICENSE file for details.
