package main

import (
	"embed"
	"errors"
	"fmt"

	"log"

	"plumelauncher/internal/bootstrap"
	"plumelauncher/internal/instances"
	"plumelauncher/internal/java"
	"plumelauncher/internal/logging"
	"plumelauncher/internal/services"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	// Register a custom event whose associated data type is string.
	// This is not required, but the binding generator will pick up registered events
	// and provide a strongly typed JS/TS API for them.
	application.RegisterEvent[services.DownloadProgressEvent](services.EventDownloadProgress)
	application.RegisterEvent[services.InstanceStateEvent](services.EventInstanceState)
	application.RegisterEvent[services.LaunchStateEvent](services.EventLaunchState)
	application.RegisterEvent[services.LogLineEvent](services.EventLogLine)
}

// main function serves as the application's entry point. It initializes the application, creates a window,
// and starts a goroutine that emits a time-based event every second. It subsequently runs the application and
// logs any error that might occur.
func main() {
	config, err := bootstrap.Initialize("")
	if err != nil {
		showStartupError(err)
		return
	}
	logger, err := logging.New(config.DataRoot)
	if err != nil {
		showStartupError(err)
		return
	}
	defer logger.Close()
	logger.Info("launcher_started", "dataRoot", config.DataRoot)
	dataRootLock, err := bootstrap.AcquireDataRootLock(config.DataRoot)
	if err != nil {
		if errors.Is(err, bootstrap.ErrDataRootLocked) {
			showStartupError(fmt.Errorf("another Plume Launcher is already using this data folder:\n\n%s", config.DataRoot))
		} else {
			showStartupError(err)
		}
		return
	}
	defer dataRootLock.Release()
	defaults, err := instances.LoadConfig(config.DataRoot)
	if err != nil {
		showStartupError(err)
		return
	}
	if !defaults.JavaDefaultInitialized {
		defaults.JavaDefaultInitialized = true
		if systemJava, err := java.SystemDefault(); err == nil && systemJava != nil {
			defaults.DefaultJavaPath = systemJava.Path
		}
		if err := instances.SaveConfig(config.DataRoot, defaults); err != nil {
			showStartupError(err)
			return
		}
	}
	registry := instances.NewRegistry()
	accountService := &services.AccountService{DataRoot: config.DataRoot}
	instanceManager := instances.NewManager(config.DataRoot, defaults)
	launchService := &services.LaunchService{DataRoot: config.DataRoot, Registry: registry, Instances: instanceManager, Logger: logger}
	instanceService := &services.InstanceService{
		DataRoot: config.DataRoot,
		Manager:  instanceManager,
	}

	// Create a new Wails application by providing the necessary options.
	// Variables 'Name' and 'Description' are for application metadata.
	// 'Assets' configures the asset server with the 'FS' variable pointing to the frontend files.
	// 'Bind' is a list of Go struct instances. The frontend has access to the methods of these instances.
	// 'Mac' options tailor the application when running an macOS.
	app := application.New(application.Options{
		Name:        "Plume Launcher",
		Description: "Plume Launcher",
		Services: []application.Service{
			application.NewService(accountService),
			application.NewService(instanceService),
			application.NewService(&services.SystemService{DataRoot: config.DataRoot, Defaults: defaults}),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})
	launchService.App = app
	app.RegisterService(application.NewService(&services.HomeService{
		DataRoot:  config.DataRoot,
		Defaults:  defaults,
		Instances: instanceManager,
		Registry:  registry,
		Launch:    launchService,
		Accounts:  accountService,
		App:       app,
		Logger:    logger,
	}))

	// Create a new window with the necessary options.
	// 'Title' is the title of the window.
	// 'Mac' options tailor the window when running on macOS.
	// 'BackgroundColour' is the background colour of the window.
	// 'URL' is the URL that will be loaded into the webview.
	mainWindow := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title: "Plume Launcher",
		// Window sized to the golden ratio (1000 / 618 ≈ 1.618).
		Width:  1600,
		Height: 900,
		// Minimum window size keeps the launcher layout responsive: the
		// `lg:` (3-column) instance grid, AccountDialog (860), SettingsSheet
		// (720 + 140 nav) and InstanceDetailSheet (640) all fit with breathing
		// room, and the TopBar + toolbar + at least one card row remain visible.
		MinWidth:  1024,
		MinHeight: 640,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(18, 16, 17),
		URL:              "/",
	})
	allowClose := false
	mainWindow.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if allowClose || !launchService.HasRunning() {
			return
		}
		event.Cancel()
		dialog := app.Dialog.Question().SetTitle("Minecraft is still running").SetMessage("Stop Minecraft and close Plume Launcher?")
		stop := dialog.AddButton("Stop and close")
		cancel := dialog.AddButton("Keep open")
		dialog.SetDefaultButton(stop)
		dialog.SetCancelButton(cancel)
		stop.OnClick(func() {
			allowClose = true
			launchService.KillAll()
			mainWindow.Close()
		})
		dialog.Show()
	})

	// Run the application. This blocks until the application has been exited.
	err = app.Run()

	// Kill all running game processes before exit.
	launchService.KillAll()

	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
	}
}

func showStartupError(err error) {
	app := application.New(application.Options{Name: "Plume Launcher"})
	app.Dialog.Error().SetTitle("Plume Launcher could not start").SetMessage(err.Error()).Show()
}
