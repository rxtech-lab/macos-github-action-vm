---
slug: code/go-packages
title: Go Package Reference
description: Go package index for the RVMM command-line application
---

# Go package reference

This index is derived from `go doc` for module `github.com/rxtech-lab/rvmm` using Go 1.22. Run `go doc -all <package>` from `cli/` for full declarations and field documentation.

## `github.com/rxtech-lab/rvmm`

The executable package selects the Bubble Tea TUI by default. It also exposes the `run` and `monitor` command modes through the process entry point.

## `assets`

Embedded build assets:

```text
var ConfigExample []byte
var EkidenPlist []byte
```

## `internal/config`

Loads, defaults, and validates the YAML configuration.

```text
type Config struct { ... }
    func Load(configPath string) (*Config, error)
type DaemonConfig struct { ... }
type GitHubConfig struct { ... }
type OptionsConfig struct { ... }
type PostHogConfig struct { ... }
type RegistryConfig struct { ... }
type VMConfig struct { ... }
func (c *Config) Validate() error
```

## `internal/daemon`

Manages launchd jobs for the runner and log monitor.

```text
func Install(log *zap.Logger, cfg *config.Config, configPath string, out io.Writer) error
func InstallMonitor(log *zap.Logger, cfg *config.Config, configPath string, out io.Writer) error
func IsRunning(cfg *config.Config) (bool, error)
func Status(log *zap.Logger, cfg *config.Config, out io.Writer) error
func StatusMonitor(log *zap.Logger, cfg *config.Config, out io.Writer) error
func Uninstall(log *zap.Logger, cfg *config.Config, out io.Writer) error
func UninstallMonitor(log *zap.Logger, cfg *config.Config, out io.Writer) error
type PlistData struct { ... }
```

## `internal/monitor`

Polls a log file and sends new lines to PostHog.

```text
type LogTailer struct { ... }
    func NewLogTailer(filePath string, logType string, posthog *posthog.Client, log *zap.Logger) *LogTailer
    func (t *LogTailer) Start(ctx context.Context) error
```

## `internal/posthog`

Builds and sends single or batched PostHog capture requests.

```text
type CaptureRequest struct { ... }
type Client struct { ... }
    func NewClient(cfg *config.PostHogConfig, log *zap.Logger) *Client
    func (c *Client) CaptureLogEvent(logType string, logLine string) error
    func (c *Client) CaptureLogEventBatch(logType string, logLines []string) error
type Event struct { ... }
```

## `internal/runner`

Coordinates registration tokens, Tart instances, SSH, and ephemeral runner workers.

```text
func Run(ctx context.Context, log *zap.Logger, cfg *config.Config) error
type GitHubClient struct { ... }
    func NewGitHubClient(cfg *config.Config, log *zap.Logger) *GitHubClient
    func (g *GitHubClient) GetRegistrationToken() (string, error)
type RegistrationTokenResponse struct { ... }
type SSHClient struct { ... }
    func NewSSHClient(cfg *config.Config, log *zap.Logger) *SSHClient
    func (s *SSHClient) ConfigureRunner(ctx context.Context, ip, token, runnerName string) error
    func (s *SSHClient) Execute(ctx context.Context, ip, command string, showOutput bool) error
    func (s *SSHClient) ExecuteWithOutput(ctx context.Context, ip, command string) (string, error)
    func (s *SSHClient) RunRunner(ctx context.Context, ip string) error
    func (s *SSHClient) WaitForSSH(ctx context.Context, ip string) error
type VMManager struct { ... }
    func NewVMManager(cfg *config.Config, log *zap.Logger) *VMManager
    func (v *VMManager) Cleanup(ctx context.Context, instanceName string)
    func (v *VMManager) Clone(ctx context.Context, instanceName string) error
    func (v *VMManager) Delete(ctx context.Context, instanceName string) error
    func (v *VMManager) GetCachePath() string
    func (v *VMManager) GetRegistryPath() string
    func (v *VMManager) ImageExists(ctx context.Context) (bool, error)
    func (v *VMManager) Login(ctx context.Context) error
    func (v *VMManager) PullImage(ctx context.Context) error
    func (v *VMManager) Start(ctx context.Context, instanceName string) (*exec.Cmd, error)
    func (v *VMManager) Stop(ctx context.Context, instanceName string) error
    func (v *VMManager) WaitForIP(ctx context.Context, instanceName string) (string, error)
```

## `internal/setup`

Installs and validates host dependencies and creates the sample configuration.

```text
var RequiredPackages = []string { ... }
var RequiredTools = []string { ... }
func CheckDependencies() error
func Run(log *zap.Logger) error
func RunWithIO(log *zap.Logger, stdout, stderr io.Writer, stdin io.Reader) error
```

## `internal/tui`

Implements the Bubble Tea interface and delegates work to the setup, runner, daemon, and image-management packages.

```text
func Run()
```

