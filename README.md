# scratch

`scratch` creates and tracks local temporary development environments.

## Install

```sh
go install github.com/chargeflux/scratch@latest
```

## Usage

`scratch` uses [pebble](https://github.com/cockroachdb/pebble) to track environments in `$HOME/.config/scratch/data` on Linux/macOS and `%AppData%` on Windows.

Environments are created in `$HOME/.local/share/scratch` on Linux/macOS and `%LocalAppData%` on Windows.

`scratch` respects `XDG_CONFIG_HOME` and `XDG_DATA_HOME`.

Newly created environments automatically open in VS Code but this behavior can be overridden.

### Commands

Create a new environment

```sh
scratch new <name>
```

List environments

```sh
scratch list
```

See `scratch -h` for more information about available commands and flags

## Environments

**Python**: `uv` is used to initialize a new python project and virtual environment

## Contributing

Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.
