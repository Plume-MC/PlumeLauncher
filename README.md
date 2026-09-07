# Plume Launcher

Plume Launcher is a compact Minecraft launcher and instance manager built on Wails v3 (Go + React).

## Development

```sh
wails3 dev
```

## Build & Package

| OS      | Command                                           | Output                                         |
| ------- | ------------------------------------------------- | ---------------------------------------------- |
| Windows | `wails3 task windows:package ARCH=amd64`          | `bin/plume-launcher-installer-amd64.exe` (NSIS) |
| Linux   | `wails3 task linux:package ARCH=amd64`            | AppImage + deb/rpm/Arch in `bin/`              |
| macOS   | `wails3 task darwin:package`                      | `bin/plume-launcher.app`                       |

Linux builds from non-Linux hosts (or without a C compiler) use the Docker
cross-compile image: `wails3 task setup:docker` once, then `linux:build:docker`.

## Release Status

Production remains **NO-GO** until the following deferred steps land:

- **STEP-001** — top-level product metadata in `wails.json` and bundle identifiers.
- **STEP-009** — release-quality error pages and user-facing copy.
- **STEP-011** — Linux tray + system integration (notifications, autostart).

`STEP-010` (Windows/Linux packaging) is in progress:

- Linux DEB/RPM/AUR metadata now points at the real `plume-launcher` binary
  (commit `9fa7268`).
- Windows NSIS / MSIX templates and `darwin/Info.plist` still reference the
  legacy `My Product` / `Rozelith Corporation` / `com.drenzzz.myapp` strings.
  Regenerate them with `wails3 update build-assets -config build/config.yml
  -dir build -name plume-launcher -binaryname plume-launcher -productname "Plume Launcher" -productidentifier com.plume.launcher -productversion 1.0.0 -productcompany Plume -productcopyright "(c) 2026, Plume" -productdescription "A compact Minecraft launcher and instance manager"`, then
  review and commit the rewrite. `makensis` (NSIS) and `appimagetool` (AppImage)
  must be available on `PATH` for the actual `package` invocation.

Frontend lint is blocked by the TypeScript 7.0 / `typescript-eslint`
incompatibility; bump `@typescript-eslint` packages once upstream support lands.

# Welcome to Your New Wails3 Project!

Congratulations on generating your Wails3 application! This README will guide you through the next steps to get your project up and running.

## Getting Started

1. Navigate to your project directory in the terminal.

2. To run your application in development mode, use the following command:

   ```
   wails3 dev
   ```

   This will start your application and enable hot-reloading for both frontend and backend changes.

3. To build your application for production, use:

   ```
   wails3 build
   ```

   This will create a production-ready executable in the `build` directory.

## Exploring Wails3 Features

Now that you have your project set up, it's time to explore the features that Wails3 offers:

1. **Check out the examples**: The best way to learn is by example. Visit the `examples` directory in the `v3/examples` directory to see various sample applications.

2. **Run an example**: To run any of the examples, navigate to the example's directory and use:

   ```
   go run .
   ```

   Note: Some examples may be under development during the alpha phase.

3. **Explore the documentation**: Visit the [Wails3 documentation](https://v3.wails.io/) for in-depth guides and API references.

4. **Join the community**: Have questions or want to share your progress? Join the [Wails Discord](https://discord.gg/JDdSxwjhGf) or visit the [Wails discussions on GitHub](https://github.com/wailsapp/wails/discussions).

## Project Structure

Take a moment to familiarize yourself with your project structure:

- `frontend/`: Contains your frontend code (HTML, CSS, JavaScript/TypeScript)
- `main.go`: The entry point of your Go backend
- `app.go`: Define your application structure and methods here
- `wails.json`: Configuration file for your Wails project

## Next Steps

1. Modify the frontend in the `frontend/` directory to create your desired UI.
2. Add backend functionality in `main.go`.
3. Use `wails3 dev` to see your changes in real-time.
4. When ready, build your application with `wails3 build`.

Happy coding with Wails3! If you encounter any issues or have questions, don't hesitate to consult the documentation or reach out to the Wails community.
