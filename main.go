package main

import (
	"embed"
	"log"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
	"github.com/wailsapp/wails/v3/pkg/updater"
	appservice "skillview/internal/app"
	"skillview/internal/cache"
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
}

func main() {
	// Helpers must run before opening user settings, cache, or Skill sources.
	updater.HandleHelperMode()
	dataDir := os.Getenv("SKILLVIEW_DATA_DIR")
	if dataDir == "" {
		base, err := os.UserConfigDir()
		if err != nil {
			log.Fatal(err)
		}
		dataDir = filepath.Join(base, "SkillView")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	candidates := library.DefaultSources(home)
	fixtureRoot := os.Getenv("SKILLVIEW_SOURCE")
	if fixtureRoot != "" {
		candidates = []library.Source{}
	}
	lib, err := library.New(dataDir, candidates)
	if err != nil {
		log.Fatal(err)
	}
	if fixtureRoot != "" {
		if _, err := lib.AddSource(fixtureRoot, os.Getenv("SKILLVIEW_FIXTURE_EDIT") == "1"); err != nil {
			log.Fatal(err)
		}
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
	store, cacheErr := cache.Open(dataDir)
	if cacheErr == nil {
		defer store.Close()
		lib.SetCache(store)
		translator.SetCache(store)
	} else {
		log.Print("Local cache unavailable; using original files.")
	}
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
		Mac:        application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
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
	updateLanguage(lib.Settings().Language)
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
