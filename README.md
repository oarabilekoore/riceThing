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
ricething build -o my-rice-bundle
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
| `--no-configs` | | Skip copying `~/.config` folders | `false` |
| `--no-packages` | | Don't include installed packages in metadata | `false` |
| `--dotfiles` | | Comma-separated list of dotfiles to include | `""` |
| `--include-dotfiles` | | Include common dotfiles (.bashrc, .profile, etc.) | `false` |

#### What Gets Packaged

- **System metadata**: Distribution name, desktop environment, shell
- **Installed packages**: Complete list with versions (via `pacman -Q`)
- **Config folders**: All directories in `~/.config/`
- **Dotfiles**: Optional inclusion of dotfiles from home directory

#### Examples

```bash
# Basic build - packages everything to current directory
ricething build

# Build to specific directory with dotfiles
ricething build -o ~/my-rice --include-dotfiles

# Build with custom dotfiles
ricething build --dotfiles=".bashrc,.zshrc,.vimrc"

# Build without packages (configs only)
ricething build --no-packages

# Build packages only (no configs)
ricething build --no-configs
```

### `install` Command

Install a rice configuration from a local directory or bundle.

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

#### What Gets Installed

- **Packages**: Installs all packages listed in metadata using `pacman`
- **Config folders**: Copies from `bundle/.config/` to `~/.config/`
- **Dotfiles**: Automatically copies common shell files (.bashrc, .zshrc, .profile, .xinitrc, etc.)
- **Missing files**: Warns about missing files but continues installation

- **Warning for desktop mismatch**: Shows warning if desktop environments don't match (no longer blocks installation)
- Requires `sudo` for package installation
- Creates backups recommended (not automated)

#### Examples

```bash
# Full installation
ricething install ./awesome-rice-bundle

# Install configs only
ricething install ./rice-bundle --skip-pkgs

# Install packages only  
ricething install ./rice-bundle --skip-configs
```

## Configuration

riceThing automatically detects your system configuration:

- **Distribution**: Read from `/etc/os-release`
- **Desktop Environment**: From `$XDG_SESSION_DESKTOP`
- **Shell**: From `$SHELL` environment variable
- **Home Directory**: From `$HOME` environment variable

## File Structure

A riceThing bundle contains:

```
rice-bundle/
├── ricemetadata.json          # System and package metadata
├── .config/                   # Desktop configuration folders
│   ├── awesome/              # Window manager configs
│   ├── kitty/                # Terminal configs
│   └── ...
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
  "name": "arch",
  "shell": "/bin/bash",
  "desktop": "awesome",
  "packages": [
    {
      "name": "firefox",
      "version": "119.0.1-1"
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

1. **Customize your desktop** with your preferred:
   - Window manager/desktop environment
   - Terminal emulator
   - Text editors
   - Themes and icons

2. **Package your rice**:
   ```bash
   ricething build -o ~/my-awesome-rice --include-dotfiles
   ```

3. **Share your bundle** (upload to GitHub, etc.)

4. **Others can install**:
   ```bash
   git clone https://github.com/user/awesome-rice
   ricething install ./awesome-rice
   ```

### Selective Installation

```bash
# Try configs first, then packages if satisfied
ricething install ./rice-bundle --skip-pkgs
# If happy with configs:
ricething install ./rice-bundle --skip-configs
```

## Troubleshooting

### Common Issues

**"Unable To Fetch Distribution Name"**
- Ensure `/etc/os-release` exists and is readable
- riceThing continues without this info

**Package Installation Fails**
- Verify you have `sudo` privileges
- Check if packages exist in your repositories
- Some packages might have different names across distributions

**Config Copy Failures**
- Check file permissions in `~/.config`
- Ensure sufficient disk space
- Some applications might be running and locking config files

**Desktop Environment Mismatch**
- riceThing now shows a warning instead of blocking installation
- This allows more flexibility while still informing users of potential issues

### Current Limitations

- **Arch Linux only**: Currently hardcoded for `pacman`
- **No git support**: Install command doesn't handle git URLs yet  
- **No backups**: Overwrites existing configs without backup
- **Missing file handling**: Warns about missing files but continues (which is usually good)

### Getting Help

```bash
ricething --help
ricething build --help
ricething install --help
```

## License

MIT.
