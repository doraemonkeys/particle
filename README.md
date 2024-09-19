# Particle

Particle is a Go-based tool designed to enhance Syncthing's ignore patterns functionality. It addresses the need for more complex ignore rules, particularly for development projects using various programming languages and frameworks.

## Features

- Automatically generates `.stignore` files based on project structure
- Supports multiple programming languages and frameworks, including:
  - Rust
  - Node.js
  - Dart
  - Python (with Conda)
- Integrates with Syncthing's web API for seamless operation
- Offers both command-line and web-based interfaces

## Installation

To install Particle, use the following Go command:

```go
go install github.com/doraemonkeys/particle@latest
```

Or you can download the binary from [here](https://github.com/doraemonkeys/particle/releases/latest), and put it in your `PATH`.


## Usage

- Basic usage (local directory):

  ```bash
  particle -dir /path/to/your/project
  ```

- Using with Syncthing Web API:

  ```bash
  particle -web -host http://127.0.0.1:8384 -user youruser
  ```

> Particle does not overwrite existing `.stignore` files. It will append new ignore patterns to the existing ones.
>



### Flags:

- `-dir`: Target directory (for local scanning)
- `-web`: Get all directories from Syncthing Web API
- `-host`: Syncthing host (default: http://127.0.0.1:8384)
- `-user`: Syncthing user
- `-pwdFile`: Path to file containing Syncthing password
- `-syncthing`: Path to Syncthing executable file (used for resolving relative paths)



### Web Interface

When using the `-web` flag, Particle will connect to your Syncthing instance and fetch all shared directories. It will then generate appropriate `.stignore` files for each directory based on its content.

#### Configuration

Particle uses environment variables and command-line flags for configuration. The Syncthing password can be provided via:

1. A password file (specified with `-pwdFile`)
2. The `SYNCTHING_PASSWORD` environment variable
3. Interactive prompt (if not provided by other means)



## How It Works

Particle scans specified directories and applies a series of ignore pattern checks based on the project type. It generates `.stignore` files that are compatible with Syncthing, allowing for more granular control over which files and directories are synchronized.


### Supported Project Types

1. **Rust Projects**: Ignores the `target` directory when a `Cargo.toml` file is present.
2. **Node.js Projects**: Ignores `node_modules` directories.
3. **Dart Projects**: Ignores specific Dart and Flutter-related build and cache directories.
4. **Python Projects**: Ignores Conda environments and related files.

## Contributing

Contributions to Particle are welcome! Please feel free to submit pull requests, report bugs, or suggest new features through the GitHub issue tracker.

## Acknowledgments

This project was inspired by the need for more complex ignore patterns in Syncthing, as discussed in [Syncthing Issue #6195](https://github.com/syncthing/syncthing/issues/6195).

