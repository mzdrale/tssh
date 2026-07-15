# tssh

`tssh` is a command-line tool for searching, grouping, and connecting to Teleport nodes with fuzzy search and interactive selection. It supports configuration profiles, local caching of nodes for fast search, and seamless integration with `tsh ssh`.

## Features

- Interactive fuzzy search for nodes by hostname
- Group nodes by hostname, environment, or service
- Local caching of nodes for fast startup and search
- Profile-based configuration via TOML
- Easy integration with Teleport (`tsh ssh`)
- Multi-platform builds (macOS, Linux, arm64, amd64)

## Installation

### Build from source

```sh
git clone https://github.com/mzdrale/tssh.git
cd tssh
make build
```

Binaries will be created in the project root, named like `tssh-<version>-<os>-<arch>`. 
If you need binary for specific OS/arch, for example for macOS Apple Silicon, which is arm64, run:

```sh
make build macos:arm64
```

### Requirements

- Go 1.22+
- [Teleport CLI (`tsh`)](https://goteleport.com/docs/cli-docs/)
- [tctl](https://goteleport.com/docs/management/admin/#tctl)

## Usage

### Configuration

Create a config file at `~/.config/tssh/config.toml`. Example:

```toml
[default]
    profile = "myprofile"

[profiles]
    [profiles.myprofile]
        proxy = "mycompany.teleport.sh"
        username = "your.username"

[environments]
    [environments.teleport_env_dev]
        color = "Green"
    [environments.teleport_env_staging]
        color = "Teal"
    [environments.teleport_env_prod]
        color = "Red"
```

### Fetch and cache nodes

```sh
tssh --update-nodes
```

or

```sh
tssh -u
```

This will fetch nodes from Teleport and save them to `~/.config/tssh/nodes.json`.

### Search and connect

```sh
tssh
```

- Type `/` and start typing a hostname to filter nodes.
- Select a node and press Enter to connect via `tsh ssh`.

### Command-line options

- `-u`, `--update-nodes` &nbsp;&nbsp; Update nodes from Teleport and cache them locally
- `-V`, `--version` &nbsp;&nbsp; Print version information

## Development

### Build for your platform

```sh
make build:macos:arm64
make build:macos:amd64
make build:linux:arm64
make build:linux:amd64
```

### Run locally

```sh
go run ./cmd
```

## Project Structure

```
cmd/         # Main CLI entrypoint and config loading
teleport/    # Teleport node logic, grouping, and helpers
config.toml  # Example configuration
Makefile     # Build targets
VERSION      # Version file
```

## License

MIT

---

**Author:** [mzdrale](https://github.com/mzdrale)
