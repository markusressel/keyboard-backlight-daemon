# AGENTS.md

Go daemon for Linux that turns the keyboard backlight off when the user is idle and restores it on input activity.

## Commands

```sh
go build ./...                 # compile everything (CI runs this)
go test -v ./...               # run tests (CI runs this); `make test` also clears the test cache
go vet ./...                   # static checks
gofmt -l .                     # list unformatted files; must print nothing
make build                     # build to bin/keyboard-backlight-daemon
```

Do not run `make deploy`, `make daemon-restart` or `make daemon-stop`: they use `sudo` and `systemctl` on the host.
Do not run the daemon binary against real hardware unless asked: it writes to `/sys/class/leds/*/brightness` and reads `/dev/input/*`.

## Layout

- `main.go`: entry point, calls `cmd.Execute()`.
- `cmd/`: cobra CLI. `root.go` holds the default run command (detect light, build service, run). `version.go` is the `version` subcommand.
- `internal/config/`: viper-based config loading into `config.CurrentConfig` (global).
- `internal/light/`: `Light` interface (brightness as a `[0..1]` fraction) and sysfs implementation; `detect.go` finds the backlight under `/sys/class/leds`.
- `internal/service/`: `KbdService`. Watches input devices, runs an idle timer, sets brightness. Goroutines are coordinated with `oklog/run`.
- `internal/util/`: small file helpers (read/write ints, find files by regex).
- `keyboard-backlight-daemon.yaml`: example config. `keyboard-backlight-daemon.service`: systemd unit. `contrib/PKGBUILD`: Arch packaging.

## Conventions

- Module path: `github.com/markusressel/keyboard-backlight-daemon`. Go version is set in `go.mod`; do not bump it without being asked.
- Application code lives under `internal/`. Keep `cmd/` thin.
- Brightness values crossing the `Light` interface are fractions in `[0..1]`, never raw sysfs integers.
- Config keys are camelCase in YAML (`backlightPath`, `inputEventDevices`, `idleTimeout`) and map to fields of `config.Configuration`. A new option needs: struct field, default in `setDefaultValues()` if any, an entry in `keyboard-backlight-daemon.yaml`, and a README mention.
- Output uses plain `fmt` plus `pterm` for the header. Keep it that way; no new logging framework.
- Follow `gofmt` style. Match the existing file's import grouping and comment density.
- Add dependencies only when needed, and run `go mod tidy` afterwards.

## Gotchas

- The code reads and writes sysfs and `/dev/input` devices, which usually need root or group membership. Unit tests must not depend on real devices; use `t.TempDir()` files or fake the `light.Light` interface.
- Errors are returned, not panicked on: `NewLight`, `DetectKeyboardBacklight`, `FindFilesMatching`, `listenToEvents`, `config.ReadConfigFile`. Only `cmd` decides to exit (`fatal`). A missing config file is not an error; defaults apply.
- Input events are 24-byte Linux `input_event` structs, parsed little-endian in `internal/service/service.go`. This assumes a 64-bit platform (CI builds linux amd64 and arm64).
- `internal/service/service.go` uses package-level channels; `activeListeners` is guarded by a mutex. Be careful when touching concurrency there.
- Tests use `t.TempDir()` fakes and a `fakeLight` in `internal/service`. Untested on purpose: `KbdService.Run`, the polling loop in `watchInputDevices`, and `rootCmd.Run` (they need real devices or call `os.Exit`). `cmd.Execute` can only run once per process.
- `config` tests set `homedir.DisableCache = true`; go-homedir otherwise caches `$HOME`.
- Do not commit `bin/` or `.idea/`.

## CI

`.github/workflows/go.yml` runs `go test -v ./...` and `go build -v ./...`, then builds linux amd64 and arm64 binaries and attaches them to tagged releases. `.github/workflows/codespell.yml` runs codespell; add legitimate false positives to `.codespellignore`.
