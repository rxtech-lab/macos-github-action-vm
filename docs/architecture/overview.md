---
slug: architecture/overview
title: RVMM Architecture Overview
description: Components and runtime flow of the macOS GitHub Actions runner VM manager
---

# RVMM architecture overview

RVMM is a Go command-line application for running ephemeral GitHub Actions self-hosted runners inside Tart virtual machines on Apple Silicon Macs. It combines host setup, image management, runner registration, bounded VM concurrency, launchd integration, and optional PostHog log forwarding.

## Repository layout

| Area | Responsibility |
| --- | --- |
| `cli/main.go` | Selects the interactive TUI, headless runner, or headless monitor entry point. |
| `cli/internal/config` | Loads `rvmm.yaml`, applies defaults, and validates required settings. |
| `cli/internal/runner` | Obtains GitHub registration tokens and manages Tart VM, SSH, and ephemeral runner lifecycles. |
| `cli/internal/tui` | Implements the Bubble Tea menus for setup, configuration, image operations, runners, daemons, and logs. |
| `cli/internal/daemon` | Installs, removes, and inspects runner and monitor launchd jobs. |
| `cli/internal/monitor` | Tails runner stdout and stderr files. |
| `cli/internal/posthog` | Sends new log lines to PostHog capture endpoints. |
| `cli/internal/setup` | Installs or validates host dependencies and creates a sample configuration. |
| `cli/assets` | Embeds the launchd plist template and example configuration in the binary. |
| `guest/runner.pkr.hcl` | Provides an example Packer template for a Tart runner image. |
| `.github/workflows` | Creates releases and builds, signs, notarizes, and publishes the macOS installer. |

## Runtime modes

Running `rvmm` without arguments opens the interactive TUI. The TUI calls the same config, setup, runner, daemon, and image-management packages used by headless operation.

`rvmm run -config rvmm.yaml` starts the runner loop. `rvmm monitor -config rvmm.yaml` starts two log tailers for the configured working directory and requires `posthog.enabled: true`.

## Runner lifecycle

1. Load and validate configuration.
2. Verify that `tart`, `sshpass`, `wget`, and `packer` are available.
3. Log in to the OCI registry when credentials are configured.
4. Reuse a local Tart image or pull the configured image from the registry.
5. Create `options.max_concurrent_runners` worker slots.
6. For each slot, request a short-lived registration token from GitHub.
7. Clone the base image to an instance named `<runner_name>_<slot>` and boot it without graphics.
8. Wait for the guest IP and SSH, then configure an ephemeral GitHub Actions runner in the guest.
9. Run one Actions job, stop the VM, and delete the instance during cleanup.
10. Return the slot to the pool and repeat until the context is cancelled or the shutdown flag exists.

Each worker owns its own `VMManager`, while the GitHub client is shared. The base image is initialized once before workers begin.

## VM image construction

The example Packer template uses the Tart plugin and starts from a Cirrus Labs macOS image. It downloads the latest arm64 GitHub Actions runner into the guest, installs Apple certificate authorities, selects Xcode, accepts its license, completes first-launch setup, and downloads available platforms and simulator runtimes.

The CLI accepts external `.pkr.hcl` templates, runs `packer init`, and then runs `packer build` from the selected template's directory. Supporting provisioner files therefore stay beside the template.

## Background services and observability

The runner can be installed as a system LaunchDaemon or user LaunchAgent depending on `daemon.plist_path`. Its stdout and stderr are written under `options.working_directory`.

The optional monitor runs as a separate LaunchAgent. It polls both files, starts at their current end, detects truncation, and forwards subsequent non-empty lines as `mac_ci_log_line` PostHog events identified by `posthog.machine_label`.

## Release flow

The manual semantic-release workflow creates a GitHub release from `main`. The release workflow runs on the self-hosted macOS ARM64 runner, executes Go tests, builds and signs `rvmm`, creates and notarizes a macOS installer package for release events, and uploads the package to the GitHub release.

