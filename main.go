package main

import (
	"context"
	"crypto/sha256"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"skillview/internal/cli"
	"skillview/internal/launch"
	"strings"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/updater"
	appservice "skillview/internal/app"
	"skillview/internal/library"
	"skillview/internal/locale"
	"skillview/internal/translation"
	"skillview/internal/updates"
)

//go:embed all:frontend/dist
var assets embed.FS

func init() {
	application.RegisterEvent[string]("skillview:command")
	application.RegisterEvent[updates.Status]("skillview:update")
	application.RegisterEvent[library.ScanProgress]("skillview:scan")
	application.RegisterEvent[translation.Progress]("skillview:translation")
	application.RegisterEvent[bool]("skillview:activated")
}

func main() {
	// Helpers must run before opening user settings, cache, or Skill sources.
	updater.HandleHelperMode()
	args := os.Args[1:]
	if len(args) > 0 && !strings.HasPrefix(args[0], "-psn_") || filepath.Base(os.Args[0]) == "skillview" {
		os.Exit(cli.Run(context.Background(), args, os.Stdout, os.Stderr, cli.Options{}))
	}
	dataDir, err := launch.DataDirectory()
	if err != nil {
		log.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	fixtureRoot := os.Getenv("SKILLVIEW_SOURCE")
	session, err := library.Open(dataDir, home, fixtureRoot, os.Getenv("SKILLVIEW_FIXTURE_EDIT") == "1")
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()
	lib := session.Library
	if session.CacheError != nil {
		log.Print("Local cache unavailable; using original files.")
	}
	var app *application.App
	var window *application.WebviewWindow
	var updateManager *updates.Manager
	var keys translation.KeyStore = translation.MacKeychain{}
	var pluginHomes func() library.PluginHomes
	var knownSources func() []library.Source = func() []library.Source { return library.DefaultSources(home) }
	if fixtureRoot != "" {
		keys = translation.NewMemoryKeys()
		knownSources = nil
		pluginHomes = func() library.PluginHomes {
			if root := os.Getenv("SKILLVIEW_PLUGIN_FIXTURE"); root != "" {
				return library.PluginHomes{Claude: filepath.Join(root, "claude"), Codex: filepath.Join(root, "codex")}
			}
			return library.PluginHomes{}
		}
	}
	translator, err := translation.New(dataDir, keys)
	if err != nil {
		log.Fatal(err)
	}
	translator.SetCache(session.Cache)
	updateLanguage := func(language string) {
		menu := application.NewMenu()
		appMenu := menu.AddSubmenu("SkillView")
		appMenu.Add(locale.T(language, "Settings…")).SetAccelerator("CmdOrCtrl+,").OnClick(func(*application.Context) { app.Event.Emit("skillview:command", "settings") })
		appMenu.Add(locale.T(language, "Check for Updates…")).OnClick(func(*application.Context) { app.Event.Emit("skillview:command", "updates") })
		appMenu.AddSeparator()
		for _, item := range []struct {
			role  application.Role
			label string
		}{{application.Hide, "Hide SkillView"}, {application.HideOthers, "Hide Others"}, {application.ShowAll, "Show All"}} {
			appMenu.AddRole(item.role)
			appMenu.FindByRole(item.role).SetLabel(locale.T(language, item.label))
		}
		appMenu.AddSeparator()
		appMenu.AddRole(application.Quit)
		appMenu.FindByRole(application.Quit).SetLabel(locale.T(language, "Quit SkillView"))
		fileMenu := menu.AddSubmenu(locale.T(language, "File"))
		for _, item := range []struct{ label, key, command string }{{"New Skill…", "CmdOrCtrl+n", "new"}, {"Save", "CmdOrCtrl+s", "save"}, {"Find", "CmdOrCtrl+f", "find"}, {"Scan Agent folders", "CmdOrCtrl+r", "rescan"}} {
			command := item.command
			fileMenu.Add(locale.T(language, item.label)).SetAccelerator(item.key).OnClick(func(*application.Context) { app.Event.Emit("skillview:command", command) })
		}
		editMenu := menu.AddSubmenu(locale.T(language, "Edit"))
		for _, item := range []struct {
			role  application.Role
			label string
		}{{application.Undo, "Undo"}, {application.Redo, "Redo"}, {application.Cut, "Cut"}, {application.Copy, "Copy"}, {application.Paste, "Paste"}, {application.SelectAll, "Select All"}} {
			editMenu.AddRole(item.role)
			editMenu.FindByRole(item.role).SetLabel(locale.T(language, item.label))
		}
		app.Menu.Set(menu)
		window.SetTitle(locale.T(language, "Skills"))
	}
	service := appservice.New(lib, appservice.Native{Application: func() *application.App { return app }, Window: func() *application.WebviewWindow { return window }, KnownSources: knownSources, PluginHomes: pluginHomes, UpdateLanguage: updateLanguage, Updates: func() *updates.Manager { return updateManager }}, translator)
	defer service.Close()
	app = application.New(application.Options{
		Name: "SkillView", Description: "Read, browse, and safely edit local Agent Skills.",
		Services:   []application.Service{application.NewService(service)},
		Assets:     application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		ShouldQuit: service.ShouldQuit,
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: fmt.Sprintf("%s-%x", launch.BundleID, sha256.Sum256([]byte(dataDir))),
			OnSecondInstanceLaunch: func(application.SecondInstanceData) {
				if window != nil {
					window.Show()
					window.Focus()
					app.Event.Emit("skillview:activated", true)
				}
			},
		},
		Mac: application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
	})
	updateManager, err = updates.New(app.Updater, fixtureRoot != "", func(status updates.Status) { app.Event.Emit("skillview:update", status) }, dataDir)
	if err != nil {
		log.Fatal(err)
	}
	app.Event.On(updater.EventDownloadProgress, func(event *application.CustomEvent) {
		if progress, ok := event.Data.(updater.Progress); ok {
			updateManager.Progress(progress)
		}
	})
	app.Event.On(updater.EventVerifying, func(*application.CustomEvent) { updateManager.Stage("verifying") })
	app.Event.On(updater.EventInstalling, func(*application.CustomEvent) { updateManager.Stage("extracting") })
	window = app.Window.NewWithOptions(application.WebviewWindowOptions{
		Name: "library", Title: "Skills", Width: 900, Height: 560, MinWidth: 760, MinHeight: 480,
		Mac:              application.MacWindow{TitleBar: application.MacTitleBarHiddenInset, InvisibleTitleBarHeight: 38, Backdrop: application.MacBackdropNormal},
		BackgroundColour: application.NewRGB(243, 244, 247), URL: "/",
	})
	window.RegisterHook(events.Common.WindowClosing, func(event *application.WindowEvent) {
		if !service.ShouldQuit() {
			event.Cancel()
		}
	})
	app.Event.OnApplicationEvent(events.Mac.ApplicationDidBecomeActive, func(*application.ApplicationEvent) {
		app.Event.Emit("skillview:activated", true)
	})
	updateLanguage(lib.Settings().Language)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
