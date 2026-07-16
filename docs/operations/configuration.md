---
slug: operations/configuration
title: RVMM Configuration and Operations
description: Configure, run, monitor, and release RVMM
---

# RVMM configuration and operations

RVMM reads YAML from the path passed with `-config`. Without an explicit path it searches for `rvmm.yaml` in the current directory, `$HOME/.rvmm`, and `/etc/rvmm`. The interactive setup flow writes an example `rvmm.yaml` in the current directory if one does not already exist.

## Host requirements

RVMM is intended for macOS on Apple Silicon with hardware virtualization enabled. Runtime and image-building operations use:

- `tart` for VM images and instances
- `sshpass` for guest automation
- `wget` for downloads
- `packer` from `hashicorp/tap` for image templates

The TUI Setup action can install Homebrew and these packages.

## Configuration sections

### `github`

| Field | Meaning |
| --- | --- |
| `api_token` | GitHub token used to request runner registration tokens. |
| `registration_endpoint` | Organization or repository runner registration-token endpoint. |
| `runner_url` | Organization or repository URL passed to the guest runner configuration. |
| `runner_name` | Base runner name; the worker slot is appended at runtime. |
| `runner_labels` | Labels registered on each ephemeral runner. |
| `runner_group` | Optional organization or enterprise runner group. |

`api_token`, `registration_endpoint`, and `runner_url` are required.

### `vm`

`username` and `password` are the guest SSH credentials and are required. `display` controls the Tart display configuration applied before boot and defaults to `3840x2160`.

### `registry`

`image_name` is required and can name a local Tart image or an OCI image. Set `url`, `username`, and `password` when RVMM must authenticate and pull from a registry. If `image_name` already starts with the configured registry URL, RVMM avoids adding the prefix twice.

### `options`

| Field | Default | Meaning |
| --- | --- | --- |
| `truncate_size` | empty | Optional target size used when resizing a pulled image. |
| `log_file` | `runner.log` | TUI log path. |
| `max_concurrent_runners` | `1` | Number of VM worker slots; must be at least one. |
| `shutdown_flag_file` | `.shutdown` | Stops new work and waits for active workers when this file exists. |
| `working_directory` | `/Users/admin/vm` | Runner logs and launchd working directory. |

### `daemon`

`label` identifies the launchd job, `plist_path` selects a system LaunchDaemon or user LaunchAgent location, and `user` is written into the generated plist. Installing under `/Library/LaunchDaemons` generally requires elevated filesystem permissions.

### `posthog`

Set `enabled: true`, then provide `api_key`, `host`, and a unique `machine_label` to forward runner logs. These values are validated only when monitoring is enabled.

## Common commands

```bash
# Build and test the CLI
cd cli
make build
make test

# Run interactively
./bin/rvmm

# Run ephemeral workers in the foreground
./bin/rvmm run -config /path/to/rvmm.yaml

# Forward new runner logs to PostHog
./bin/rvmm monitor -config /path/to/rvmm.yaml

# Build the example guest image
cd ../guest
packer init runner.pkr.hcl
packer build runner.pkr.hcl
```

## Operational checks

- Confirm the image is visible with `tart list` before starting workers.
- Confirm the guest credentials match the Packer image and that SSH is reachable.
- Confirm the GitHub token can call the configured registration endpoint.
- Check the configured launchd domain with `launchctl print system/<label>` for a system daemon or `launchctl print gui/$(id -u)/<label>` for a user agent.
- Inspect `stdout` and `stderr` under `options.working_directory` when a runner or monitor job fails.
- Remove the shutdown flag before restarting the runner loop.

Treat GitHub, registry, guest, PostHog, Apple signing, and notarization credentials as secrets. Do not commit a populated `rvmm.yaml`.

