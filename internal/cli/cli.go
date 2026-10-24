package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"skillview/internal/launch"
	"skillview/internal/library"
	"skillview/internal/updates"
)

const help = `SkillView

  app                         Open the app
  list [agent] [--current]     List cached Skills
  scan                        Scan known folders
  show <id|path> [file]        Read a Skill file
  plugins [agent]             List installed plugins
  current                     Show the current Agent
  recommend <query> [--top N]  Match Skill names and descriptions
  archived                    List archived Skills
  archive <id|path>            Preview archiving one copy
  restore <archive-id>         Restore one archived copy with --apply
  sync [agent ...]            Preview links; default Agents + Claude
  cleanup [agent]             Preview broken links and empty folders
  backup                      Back up catalogued Skills
  compare <agent> <agent>      Compare installed Skills
  topology                    Inspect links
  graph [agent]               Print a Mermaid Agent/Skill graph
  doctor                      Check folders, links and catalog issues
  security [path]             Inspect local text for risk patterns
  folders                     List approved folders
  folders add <path>          Add a folder with --apply [--writable]
  folders remove <id>         Remove a folder approval with --apply
  cli install                 Add ~/.local/bin/skillview
  version                     Show the version

Options: --json, --platform <agent>, --apply, --token <preview-token>
Archive, sync and cleanup require --apply --token from their preview.
Read-only folders must first be approved with folders add --writable.
No Skill download or installation commands.
`

type Options struct {
	Home       string
	DataDir    string
	Executable string
}

type arguments struct {
	command  string
	values   []string
	json     bool
	apply    bool
	current  bool
	writable bool
	platform string
	token    string
	top      int
}

func parse(args []string) (arguments, error) {
	result := arguments{command: "help", top: 5}
	if len(args) == 0 {
		return result, nil
	}
	result.command = strings.ToLower(args[0])
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--json":
			result.json = true
		case "--apply":
			result.apply = true
		case "--current":
			result.current = true
		case "--writable":
			result.writable = true
		case "--platform", "--token", "--top":
			flag := args[i]
			i++
			if i == len(args) {
				return result, fmt.Errorf("Missing value for %s", flag)
			}
			switch flag {
			case "--platform":
				result.platform = args[i]
			case "--token":
				result.token = args[i]
			case "--top":
				value, err := strconv.Atoi(args[i])
				if err != nil || value < 1 || value > 100 {
					return result, fmt.Errorf("--top must be between 1 and 100")
				}
				result.top = value
			}
		default:
			if strings.HasPrefix(args[i], "-") {
				return result, fmt.Errorf("Unknown option: %s", args[i])
			}
			result.values = append(result.values, args[i])
		}
	}
	switch result.command {
	case "guide":
		result.command = "recommend"
	case "sync-all":
		result.command = "sync"
	case "delete":
		result.command = "archive"
	case "web":
		result.command = "app"
	}
	return result, nil
}

func Run(ctx context.Context, args []string, out, errors io.Writer, options Options) int {
	a, err := parse(args)
	if err != nil {
		fmt.Fprintln(errors, err)
		return 2
	}
	if a.current && a.platform != "" {
		fmt.Fprintln(errors, "Choose --current or --platform.")
		return 2
	}
	if a.apply && !contains([]string{"archive", "restore", "sync", "cleanup", "folders"}, a.command) {
		fmt.Fprintln(errors, "--apply is only for file or folder changes.")
		return 2
	}
	if a.writable && (a.command != "folders" || len(a.values) != 2 || a.values[0] != "add") {
		fmt.Fprintln(errors, "--writable requires folders add.")
		return 2
	}
	if a.token != "" && !contains([]string{"archive", "sync", "cleanup"}, a.command) {
		fmt.Fprintln(errors, "--token requires archive, sync or cleanup.")
		return 2
	}
	if a.platform != "" && !contains([]string{"list", "show", "archive", "plugins", "recommend", "graph", "sync", "cleanup"}, a.command) {
		fmt.Fprintln(errors, "This command does not accept --platform.")
		return 2
	}
	if a.current && !contains([]string{"list", "show", "archive", "recommend", "graph"}, a.command) {
		fmt.Fprintln(errors, "This command does not accept --current.")
		return 2
	}
	if a.command == "help" || a.command == "--help" || a.command == "-h" {
		fmt.Fprint(out, help)
		return 0
	}
	if a.command == "version" || a.command == "--version" {
		fmt.Fprintln(out, updates.Version())
		return 0
	}
	if options.Home == "" {
		options.Home, err = os.UserHomeDir()
	}
	if err == nil && options.Executable == "" {
		options.Executable, err = os.Executable()
	}
	if err != nil {
		fmt.Fprintln(errors, err)
		return 1
	}
	if a.command == "app" {
		if len(a.values) != 0 {
			fmt.Fprintln(errors, "Usage: skillview app")
			return 2
		}
		if err = launch.OpenApp(options.Executable); err != nil {
			fmt.Fprintln(errors, err)
			return 1
		}
		return 0
	}
	if a.command == "cli" {
		if len(a.values) != 1 || a.values[0] != "install" {
			fmt.Fprintln(errors, "Usage: skillview cli install")
			return 2
		}
		path, err := launch.InstallCommand(options.Executable, options.Home)
		if err != nil {
			fmt.Fprintln(errors, err)
			return 1
		}
		fmt.Fprintln(out, path)
		fmt.Fprintln(errors, "Add ~/.local/bin to PATH if needed.")
		return 0
	}
	known := map[string]bool{}
	for _, command := range strings.Fields("list scan show plugins current recommend archived archive restore sync cleanup backup compare topology graph doctor security folders") {
		known[command] = true
	}
	if !known[a.command] {
		fmt.Fprintf(errors, "Unknown command: %s\n", a.command)
		return 2
	}
	if options.DataDir == "" {
		options.DataDir, err = launch.DataDirectory()
	}
	if err != nil {
		fmt.Fprintln(errors, err)
		return 1
	}
	session, err := library.Open(options.DataDir, options.Home, os.Getenv("SKILLVIEW_SOURCE"), os.Getenv("SKILLVIEW_FIXTURE_EDIT") == "1")
	if err != nil {
		fmt.Fprintln(errors, err)
		return 1
	}
	defer session.Close()
	r := runner{ctx: ctx, lib: session.Library, options: options, args: a, out: out, errors: errors}
	if err = r.execute(); err != nil {
		fmt.Fprintln(errors, err)
		return 1
	}
	return 0
}

type runner struct {
	ctx     context.Context
	lib     *library.Library
	options Options
	args    arguments
	out     io.Writer
	errors  io.Writer
}

func (r *runner) emit(value any) error {
	encoder := json.NewEncoder(r.out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func (r *runner) exact(count int) error {
	if len(r.args.values) != count {
		return fmt.Errorf("Invalid arguments for %s. See skillview help.", r.args.command)
	}
	return nil
}

func (r *runner) catalog() (library.Catalog, error) { return r.lib.CachedCatalog() }

func (r *runner) scan() (library.Catalog, error) {
	if os.Getenv("SKILLVIEW_SOURCE") == "" {
		if err := r.lib.RefreshSources(library.DefaultSources(r.options.Home)); err != nil {
			return library.Catalog{}, err
		}
	}
	last := -1
	return r.lib.Scan(r.ctx, func(p library.ScanProgress) {
		if p.SourcesDone != last {
			fmt.Fprintf(r.errors, "Scanned %d/%d folders; %d Skills\n", p.SourcesDone, p.SourcesTotal, p.Skills)
			last = p.SourcesDone
		}
	})
}

func (r *runner) selectedSkills(catalog library.Catalog, name string) ([]library.Skill, error) {
	if name == "" || name == "all" {
		return catalog.Skills, nil
	}
	platform, err := library.FindPlatform(r.options.Home, name)
	if err != nil {
		return nil, err
	}
	result := []library.Skill{}
	for _, skill := range catalog.Skills {
		if platform.Includes(skill) {
			result = append(result, skill)
		}
	}
	return result, nil
}

func (r *runner) resolve(catalog library.Catalog, value string) (library.Skill, error) {
	path, _ := filepath.Abs(value)
	if filepath.Base(path) == "SKILL.md" {
		path = filepath.Dir(path)
	}
	if real, err := filepath.EvalSymlinks(path); err == nil {
		path = real
	}
	matches := []library.Skill{}
	for _, skill := range catalog.Skills {
		if skill.ID == value || skill.Path == path {
			return skill, nil
		}
		if skill.Name == value {
			matches = append(matches, skill)
		}
	}
	if len(matches) == 1 {
		return matches[0], nil
	}
	if len(matches) > 1 {
		return library.Skill{}, fmt.Errorf("Multiple copies match. Use a Skill ID or original path.")
	}
	return library.Skill{}, fmt.Errorf("Skill not found. Run skillview scan first.")
}

func (r *runner) execute() error {
	a := r.args
	if a.platform != "" && (a.command == "sync" || a.command == "cleanup") {
		a.values = append(a.values, a.platform)
	}
	if contains([]string{"current", "archived", "backup", "doctor", "topology"}, a.command) {
		if err := r.exact(0); err != nil {
			return err
		}
	}
	if a.command == "folders" {
		return r.folders()
	}
	if a.command == "current" {
		runtime := CurrentRuntime()
		platform, err := library.FindPlatform(r.options.Home, runtime.Platform)
		if err != nil {
			return err
		}
		return r.emit(map[string]any{"runtime": runtime, "paths": platform.Paths})
	}
	if a.command == "plugins" {
		if len(a.values) > 1 {
			return fmt.Errorf("Usage: skillview plugins [agent]")
		}
		catalog := r.lib.DiscoverPlugins(library.DefaultPluginHomes(r.options.Home))
		name := a.platform
		if len(a.values) == 1 {
			name = a.values[0]
		}
		if name != "" {
			platform, err := library.FindPlatform(r.options.Home, name)
			if err != nil {
				return err
			}
			filtered := []library.Plugin{}
			for _, p := range catalog.Plugins {
				if p.Agent == platform.Name {
					filtered = append(filtered, p)
				}
			}
			catalog.Plugins = filtered
		}
		if a.json {
			return r.emit(catalog)
		}
		for _, p := range catalog.Plugins {
			fmt.Fprintf(r.out, "%s\t%s\t%s\n", p.Name, p.Version, p.Agent)
		}
		return nil
	}
	if a.command == "archived" {
		value, err := r.lib.ArchivedSkills()
		if err != nil {
			return err
		}
		return r.emit(value)
	}
	if a.command == "restore" {
		if err := r.exact(1); err != nil {
			return err
		}
		if !a.apply {
			return r.emit(map[string]any{"restore": a.values[0], "applied": false})
		}
		if err := r.lib.RestoreArchivedSkill(a.values[0]); err != nil {
			return err
		}
		_, err := r.scan()
		return err
	}
	if a.command == "security" {
		if len(a.values) > 1 {
			return fmt.Errorf("Usage: skillview security [path]")
		}
		path := ""
		if len(a.values) == 1 {
			path = a.values[0]
		} else {
			p, _ := library.FindPlatform(r.options.Home, "agents")
			path = p.WritePath
		}
		findings, err := library.SecurityScan(path)
		if err != nil {
			return err
		}
		return r.emit(findings)
	}
	if a.command == "scan" {
		if err := r.exact(0); err != nil {
			return err
		}
		catalog, err := r.scan()
		if err != nil {
			return err
		}
		if a.json {
			return r.emit(catalog)
		}
		fmt.Fprintf(r.out, "%d Skill copies\n", len(catalog.Skills))
		return nil
	}
	catalog, err := r.catalog()
	if err != nil {
		return err
	}
	if catalog.NeedsScan && a.command != "doctor" {
		return fmt.Errorf("No current catalog. Run skillview scan.")
	}
	filter := a.platform
	if a.current {
		filter = CurrentRuntime().Platform
	}
	if a.command == "list" || a.command == "graph" {
		if len(a.values) > 1 {
			return fmt.Errorf("Use one Agent filter.")
		}
		if len(a.values) == 1 {
			filter = a.values[0]
		}
	}
	skills, err := r.selectedSkills(catalog, filter)
	if err != nil {
		return err
	}
	lookup := catalog
	lookup.Skills = skills
	switch a.command {
	case "list":
		return r.list(skills)
	case "show":
		if len(a.values) < 1 || len(a.values) > 2 {
			return fmt.Errorf("Usage: skillview show <id|path> [file]")
		}
		skill, err := r.resolve(lookup, a.values[0])
		if err != nil {
			return err
		}
		path := "SKILL.md"
		if len(a.values) == 2 {
			path = a.values[1]
		}
		doc, err := r.lib.Read(skill.ID, path)
		if err != nil {
			return err
		}
		if a.json {
			return r.emit(doc)
		}
		_, err = fmt.Fprint(r.out, doc.Text)
		return err
	case "archive":
		if err := r.exact(1); err != nil {
			return err
		}
		skill, err := r.resolve(lookup, a.values[0])
		if err != nil {
			return err
		}
		if !a.apply {
			preview, err := r.lib.PlanArchive(skill.ID)
			if err != nil {
				return err
			}
			return r.emit(preview)
		}
		if a.token == "" {
			return fmt.Errorf("Use --token from the archive preview.")
		}
		receipt, err := r.lib.ArchiveSkill(skill.ID, a.token)
		if err != nil {
			return err
		}
		if err = r.emit(receipt); err != nil {
			return err
		}
		_, err = r.scan()
		return err
	case "sync", "cleanup":
		if a.apply {
			plan, err := r.lib.ApplyManagement(r.options.Home, a.command, a.values, a.token)
			if err != nil {
				return err
			}
			if err = r.emit(plan); err != nil {
				return err
			}
			_, err = r.scan()
			return err
		}
		plan, err := r.lib.PlanManagement(r.options.Home, a.command, a.values)
		if err != nil {
			return err
		}
		return r.emit(plan)
	case "backup":
		path, err := r.lib.Backup()
		if err != nil {
			return err
		}
		return r.emit(map[string]string{"path": path})
	case "recommend":
		if len(a.values) == 0 {
			return fmt.Errorf("Usage: skillview recommend <query>")
		}
		if filter == "" {
			skills, err = r.selectedSkills(catalog, CurrentRuntime().Platform)
			if err != nil {
				return err
			}
		}
		return r.emit(Recommend(skills, strings.Join(a.values, " "), a.top))
	case "compare":
		if err := r.exact(2); err != nil {
			return err
		}
		left, err := r.selectedSkills(catalog, a.values[0])
		if err != nil {
			return err
		}
		right, err := r.selectedSkills(catalog, a.values[1])
		if err != nil {
			return err
		}
		return r.emit(Compare(left, right))
	case "topology", "doctor":
		return r.inspect(catalog)
	case "graph":
		fmt.Fprintln(r.out, "graph LR")
		for _, skill := range skills {
			for _, tool := range skill.Tools {
				fmt.Fprintf(r.out, "  a%s[%s] --> s%s[%s]\n", graphID(tool), strconv.Quote(tool), skill.ID, strconv.Quote(skill.Name))
			}
		}
		return nil
	}
	return fmt.Errorf("Unknown command.")
}

func (r *runner) folders() error {
	a := r.args
	if len(a.values) == 0 {
		return r.emit(r.lib.Settings().Sources)
	}
	if len(a.values) != 2 {
		return fmt.Errorf("Usage: skillview folders add <path> [--writable] --apply | folders remove <id> --apply")
	}
	switch a.values[0] {
	case "add":
		if !a.apply {
			return r.emit(map[string]any{"path": a.values[1], "writable": a.writable, "applied": false})
		}
		source, err := r.lib.AddSource(a.values[1], a.writable)
		if err != nil {
			return err
		}
		return r.emit(source)
	case "remove":
		if !a.apply {
			return r.emit(map[string]any{"remove": a.values[1], "applied": false})
		}
		return r.lib.RemoveSource(a.values[1])
	}
	return fmt.Errorf("Unknown folders command.")
}

type Group struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Agents      []string        `json:"agents"`
	Copies      []library.Skill `json:"copies"`
}

func (r *runner) list(skills []library.Skill) error {
	byName := map[string]*Group{}
	for _, skill := range skills {
		key := strings.ToLower(strings.TrimSpace(skill.Name))
		group := byName[key]
		if group == nil {
			group = &Group{Name: skill.Name, Description: skill.Description, Agents: []string{}, Copies: []library.Skill{}}
			byName[key] = group
		}
		group.Copies = append(group.Copies, skill)
		for _, tool := range skill.Tools {
			if !contains(group.Agents, tool) {
				group.Agents = append(group.Agents, tool)
			}
		}
	}
	groups := []Group{}
	for _, group := range byName {
		sort.Strings(group.Agents)
		groups = append(groups, *group)
	}
	sort.Slice(groups, func(i, j int) bool { return strings.ToLower(groups[i].Name) < strings.ToLower(groups[j].Name) })
	if r.args.json {
		return r.emit(groups)
	}
	for _, group := range groups {
		fmt.Fprintf(r.out, "%s\t%s\t%s\n", group.Name, strings.Join(group.Agents, ", "), strings.Join(strings.Fields(group.Description), " "))
	}
	return nil
}

func contains(values []string, value string) bool {
	for _, item := range values {
		if item == value {
			return true
		}
	}
	return false
}

func graphID(value string) string {
	result := ""
	for _, c := range []byte(value) {
		result += fmt.Sprintf("%02x", c)
	}
	return result
}

func (r *runner) inspect(catalog library.Catalog) error {
	value, err := r.lib.Inspect(r.args.command == "doctor")
	if err != nil {
		return err
	}
	value.NeedsScan = catalog.NeedsScan
	return r.emit(value)
}
