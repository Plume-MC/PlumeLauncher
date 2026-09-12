<p align="center">
  <img src="build/appicon.png" alt="Plume Launcher icon" width="96">
</p>

<h1 align="center">Plume Launcher</h1>

<p align="center">A desktop Minecraft launcher for managing multiple instances in one place.</p>

<p align="center">
  <a href="https://github.com/Plume-MC/PlumeLauncher/stargazers">
    <img src="https://img.shields.io/github/stars/Plume-MC/PlumeLauncher?style=flat&amp;logo=github&amp;label=Stars" alt="GitHub stars">
  </a>
  <a href="https://github.com/Plume-MC/PlumeLauncher/issues">
    <img src="https://img.shields.io/github/issues/Plume-MC/PlumeLauncher?style=flat&amp;logo=github&amp;label=Issues" alt="Open GitHub issues">
  </a>
  <a href="https://github.com/Plume-MC/PlumeLauncher/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/Plume-MC/PlumeLauncher?style=flat&amp;label=License" alt="MIT license">
  </a>
  <a href="https://github.com/Plume-MC/PlumeLauncher/releases">
    <img src="https://img.shields.io/github/v/tag/Plume-MC/PlumeLauncher?style=flat&amp;label=Latest%20tag" alt="Latest Git tag">
  </a>
  <a href="https://github.com/Plume-MC/PlumeLauncher/actions/workflows/ci.yml">
    <img src="https://github.com/Plume-MC/PlumeLauncher/actions/workflows/ci.yml/badge.svg" alt="CI status">
  </a>
</p>

<p align="center">
  <a href="#supported-platforms">
    <img src="https://img.shields.io/badge/Linux-GTK4%20%2B%20WebKitGTK%206-4c9a2a?style=flat&amp;logo=linux&amp;logoColor=white" alt="Linux with GTK4 and WebKitGTK 6">
  </a>
  <a href="#supported-platforms">
    <img src="https://img.shields.io/badge/Windows%2010%2B-WebView2-0078d6?style=flat&amp;logo=windows&amp;logoColor=white" alt="Windows 10 and later with WebView2">
  </a>
</p>

<p align="center">
  <a href="#development">
    <img src="https://img.shields.io/badge/Go-1.26%2B-00ADD8?style=flat&amp;logo=go&amp;logoColor=white" alt="Go 1.26 or later">
  </a>
  <a href="#development">
    <img src="https://img.shields.io/badge/Wails-v3.0.0--beta.20-4b5563?style=flat" alt="Wails v3.0.0 beta.20">
  </a>
  <a href="#development">
    <img src="https://img.shields.io/badge/React-19-61DAFB?style=flat&amp;logo=react&amp;logoColor=20232a" alt="React 19">
  </a>
  <a href="#development">
    <img src="https://img.shields.io/badge/TypeScript-7-3178C6?style=flat&amp;logo=typescript&amp;logoColor=white" alt="TypeScript 7">
  </a>
  <a href="#development">
    <img src="https://img.shields.io/badge/Vite-8-646CFF?style=flat&amp;logo=vite&amp;logoColor=white" alt="Vite 8">
  </a>
  <a href="#development">
    <img src="https://img.shields.io/badge/Bun-package%20manager-f9f1e1?style=flat&amp;logo=bun&amp;logoColor=14151a" alt="Bun package manager">
  </a>
  <a href="#development">
    <img src="https://img.shields.io/badge/Tailwind%20CSS-v4-06B6D4?style=flat&amp;logo=tailwindcss&amp;logoColor=white" alt="Tailwind CSS v4">
  </a>
</p>

<p align="center">
  <a href="#features">Features</a> |
  <a href="#supported-platforms">Platforms</a> |
  <a href="#installation">Installation</a> |
  <a href="#development">Development</a> |
  <a href="#troubleshooting">Troubleshooting</a> |
  <a href="#contributing">Contributing</a>
</p>

Plume Launcher keeps each Minecraft setup isolated in its own instance, with its
own version, loader, settings, and saves. Create an instance, install the files,
then launch it from the library. The app is built with Wails v3, with a Go
backend and a React frontend.

## Features

### Instance library

- Create instances for Vanilla, Fabric, or Quilt releases.
- List, search, filter by loader, and sort instances.
- Edit per-instance settings such as RAM limits and Java path.
- Open the instance folder or delete an instance from the library.

### Download and repair

- Install instance files with visible file progress.
- Retry a failed install without recreating the instance.
- Verify installed files and repair broken ones.
- Cancel an active download and return the instance to its prior state.

### Launch and monitor

- Launch an instance and track preparing, running, stopping, crashed, and failed states.
- Stop a running game from the launcher.
- Launch arguments that contain tokens are redacted before logging.

### Accounts

- Play offline with a local profile name.
- Sign in with an Ely.by account, refresh the session, and sign out.
- Ely.by tokens are stored in the OS keyring, not in plain config files.
- Select which account launches an instance.

### Java runtimes

- Scan the system for installed Java runtimes.
- Add a custom Java executable after validation.
- Set a default Java path used for new instances.
- Validate a Java path against the major version an instance needs.

### Data and settings

- One data root holds accounts, instances, game files, settings, and logs.
- Change the data root with `PLUME_DATA_ROOT` or from settings; the choice is
  stored in `bootstrap.json` and applied on restart.
- Open the data root, game files, or log folder from settings.
- Set default RAM bounds, GPU preference, and an optional launcher wrapper command.

## Supported platforms

| Platform | Requirements | Distribution |
| --- | --- | --- |
| Linux | GTK4 and WebKitGTK 6 | AppImage, DEB, RPM, and Arch packages |
| Windows 10 and later | Microsoft WebView2 Runtime | Portable executable and Wails packaging tasks |

The Linux `appimage` target bundles more dependencies for portability. The
`appimage-lite` target expects the host system to provide GTK4, WebKitGTK 6,
and their runtime dependencies.

## Installation

Download the package for your platform from the
[Plume Launcher releases page](https://github.com/Plume-MC/PlumeLauncher/releases).

### Linux

For an AppImage:

```bash
chmod +x ./plume-launcher-1.0.0-linux-x86_64-system.AppImage
./plume-launcher-1.0.0-linux-x86_64-system.AppImage
```

Install a system package with the matching tool:

```bash
sudo apt install ./plume-launcher_1.0.0-1_amd64.deb
sudo dnf install ./plume-launcher-1.0.0-1.x86_64.rpm
sudo pacman -U ./plume-launcher-1.0.0-1-x86_64.pkg.tar.zst
```

### Windows

Run the downloaded `.exe`. On Windows 10 or managed machines, install the
Evergreen WebView2 Runtime if the app window does not start.

### First launch

Add an account, create an instance, install its files, then press Play. If the
game needs a specific Java major version, set it in the instance settings
before launching.

Default data locations:

- Linux: `~/.local/share/PlumeLauncher` or `$XDG_DATA_HOME/PlumeLauncher`
- Windows: `%LOCALAPPDATA%/PlumeLauncher`

### Portable mode

To keep data beside the application instead, create an empty file named
`portable.txt` next to the executable and restart the launcher. The settings
screen shows a portable-mode notice while it is active. Move the app folder to
relocate everything together. Keep the folder somewhere writable; a system
program folder is not a good portable location.

Explicit overrides still win: a custom path argument beats `PLUME_DATA_ROOT`,
which beats portable mode, which beats a saved data root. Signed-in Ely.by
sessions stay on the machine because tokens live in the OS keyring, so expect
to sign in again after moving portable data to another computer.

## Development

### Requirements

- Go 1.26 or later
- Bun
- Wails v3 CLI
- Platform dependencies required by Wails and the target operating system

The repository currently uses Wails `v3.0.0-beta.20`.

Clone the repository and install dependencies:

```bash
git clone https://github.com/Plume-MC/PlumeLauncher.git
cd PlumeLauncher/myapp
go mod download
cd frontend
bun install
cd ..
```

Run the app with hot reload:

```bash
wails3 dev
```

Build a production executable. The output is written to `bin/`:

```bash
wails3 build
```

Run the checks used by the project:

```bash
make check
make check-all
go test ./...
```

Package the current platform:

```bash
wails3 package
```

Linux package targets are also available through the Makefile:

```bash
make deb rpm arch appimage
make appimage-lite
```

After changing an exported Go service, regenerate the TypeScript bindings:

```bash
wails3 generate bindings -clean -ts
```

Generated bindings live in `frontend/bindings/`. Do not edit those files by hand.

## Project layout

- `main.go` contains the Wails application shell and window setup.
- `internal/` contains instances, download, launch, Java, auth, metadata, and storage logic.
- `internal/services/` exposes the service facade used by Wails.
- `frontend/src/` contains screens, components, layouts, and service adapters.
- `frontend/bindings/` contains generated TypeScript bindings.
- `build/` contains Wails build, icon, and packaging assets.

## Troubleshooting

| Symptom | Check |
| --- | --- |
| Game needs another Java version | Set the instance Java path to a runtime with the required major version. |
| Another launcher instance locks the data folder | Close the other window, or point one copy at another data root. |
| Download fails or stalls | Retry the instance, then verify and repair it from the library. |
| Launch ends as crashed or failed | Open the log folder from settings and read the latest game log. |
| Windows app does not open | Install or repair Microsoft WebView2 Runtime. |
| Linux app does not start | Install GTK4 and WebKitGTK 6 for the distribution. |
| Ely.by session expired | Sign in again; the stored token was removed or rejected. |

## Contributing

Open an issue for a bug or feature request. Pull requests that change behavior
should include tests for the affected path and pass `make check`.

## Disclaimer

Plume Launcher is an independent project. It is not affiliated with, endorsed
by, or connected to Mojang Studios or Microsoft.

## License

Plume Launcher is available under the [MIT License](LICENSE).
