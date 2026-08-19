// Command kui plugin manages the plugin lifecycle: list, install, remove,
// and inspect installed plugins.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/biggs-100/kui/internal/plugin"
)

const pluginUsage = `kui plugin SUBCOMMAND

plugin subcommands:
  list [--format table|json]   list installed plugins
  install <path> [--project]   install a plugin from a local path
  remove <name> [--yes]        remove an installed plugin
  info <name>                  show detailed plugin information
`

// runPlugin dispatches the plugin management subcommands.
func runPlugin(root string, args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, pluginUsage)
		return 2
	}

	switch args[0] {
	case "list":
		return pluginList(args[1:])
	case "install":
		return pluginInstall(root, args[1:])
	case "remove":
		return pluginRemove(args[1:])
	case "info":
		return pluginInfo(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "kui: unknown plugin subcommand %q\n", args[0])
		fmt.Fprint(os.Stderr, pluginUsage)
		return 2
	}
}

// pluginListDir returns the default plugin directory (global).
func pluginListDir() string {
	return filepath.Join(configRoot(), "plugins")
}

// pluginProjectDir returns the project-local plugin directory.
func pluginProjectDir(root string) string {
	return filepath.Join(root, ".kui", "plugins")
}

// pluginList lists installed plugins. Without flags it prints a human-readable
// table; --format json outputs a JSON array.
func pluginList(args []string) int {
	format := "table"
	for i := 0; i < len(args); i++ {
		if args[i] == "--format" && i+1 < len(args) {
			format = args[i+1]
			i++
		}
	}

	dir := pluginListDir()
	discovery := plugin.NewPluginDiscoveryFromDir(dir)
	manifests, err := discovery.Discover()
	if err != nil {
		fmt.Fprintf(os.Stderr, "kui: discover plugins: %v\n", err)
		return 1
	}

	if len(manifests) == 0 {
		if format == "json" {
			fmt.Fprintln(os.Stdout, "[]")
		} else {
			fmt.Fprintln(os.Stdout, "No plugins installed.")
		}
		return 0
	}

	switch format {
	case "json":
		type pluginJSON struct {
			Name        string `json:"name"`
			Version     string `json:"version"`
			Type        string `json:"type"`
			EntryPoint  string `json:"entry_point"`
			Description string `json:"description,omitempty"`
		}
		items := make([]pluginJSON, 0, len(manifests))
		for _, m := range manifests {
			items = append(items, pluginJSON{
				Name:        m.Name,
				Version:     m.Version,
				Type:        string(m.Type),
				EntryPoint:  m.EntryPoint,
				Description: m.Description,
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(items); err != nil {
			fmt.Fprintf(os.Stderr, "kui: encode json: %v\n", err)
			return 1
		}
	default:
		fmt.Fprintf(os.Stdout, "%-30s %-10s %-12s %s\n", "NAME", "VERSION", "TYPE", "STATUS")
		fmt.Fprintf(os.Stdout, "%-30s %-10s %-12s %s\n", strings.Repeat("-", 30), strings.Repeat("-", 10), strings.Repeat("-", 12), strings.Repeat("-", 6))
		for _, m := range manifests {
			fmt.Fprintf(os.Stdout, "%-30s %-10s %-12s %s\n", m.Name, m.Version, string(m.Type), "loaded")
		}
	}
	return 0
}

// pluginInstall installs a plugin from a local path into the global plugins
// directory. With --project it installs to the project-local directory.
func pluginInstall(root string, args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, pluginUsage)
		return 2
	}

	source := args[0]
	projectMode := false
	for _, a := range args[1:] {
		if a == "--project" {
			projectMode = true
		}
	}

	installDir := pluginListDir()
	if projectMode {
		installDir = pluginProjectDir(root)
	}

	// Ensure install directory exists.
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "kui: create plugin directory: %v\n", err)
		return 1
	}

	// Set up discovery + registry for the installer.
	discovery := plugin.NewPluginDiscoveryFromDir(installDir)
	registry := plugin.NewPluginRegistry(discovery)
	installer := plugin.NewPluginInstaller(registry, installDir)

	p, err := installer.Install(source)
	if err != nil {
		fmt.Fprintf(os.Stderr, "kui: install plugin: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(os.Stdout, "Installed plugin %s (v%s)\n", p.Manifest.Name, p.Manifest.Version)
	return 0
}

// pluginRemove removes an installed plugin by name. The --yes flag skips
// confirmation.
func pluginRemove(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, pluginUsage)
		return 2
	}

	name := args[0]
	autoYes := false
	for _, a := range args[1:] {
		if a == "--yes" || a == "-y" {
			autoYes = true
		}
	}

	if !autoYes {
		fmt.Fprintf(os.Stderr, "Remove plugin %q? [y/N] ", name)
		var answer string
		_, _ = fmt.Scanln(&answer)
		answer = strings.ToLower(strings.TrimSpace(answer))
		if answer == "" || (answer != "y" && answer != "yes") {
			fmt.Fprintln(os.Stderr, "Cancelled.")
			return 0
		}
	}

	dir := pluginListDir()
	discovery := plugin.NewPluginDiscoveryFromDir(dir)
	registry := plugin.NewPluginRegistry(discovery)
	if err := registry.Load(); err != nil {
		fmt.Fprintf(os.Stderr, "kui: load plugins: %v\n", err)
		return 1
	}

	installer := plugin.NewPluginInstaller(registry, dir)
	if err := installer.Uninstall(name); err != nil {
		fmt.Fprintf(os.Stderr, "kui: remove plugin: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(os.Stdout, "Removed plugin %s\n", name)
	return 0
}

// pluginInfo shows detailed information about a specific installed plugin.
func pluginInfo(args []string) int {
	if len(args) == 0 {
		fmt.Fprint(os.Stderr, pluginUsage)
		return 2
	}

	name := args[0]

	dir := pluginListDir()
	discovery := plugin.NewPluginDiscoveryFromDir(dir)
	manifests, err := discovery.Discover()
	if err != nil {
		fmt.Fprintf(os.Stderr, "kui: discover plugins: %v\n", err)
		return 1
	}

	for _, m := range manifests {
		if m.Name == name {
			fmt.Fprintf(os.Stdout, "Name:        %s\n", m.Name)
			fmt.Fprintf(os.Stdout, "Version:     %s\n", m.Version)
			fmt.Fprintf(os.Stdout, "Type:        %s\n", m.Type)
			fmt.Fprintf(os.Stdout, "Entry Point: %s\n", m.EntryPoint)
			if m.Description != "" {
				fmt.Fprintf(os.Stdout, "Description: %s\n", m.Description)
			}
			if len(m.Capabilities) > 0 {
				fmt.Fprintf(os.Stdout, "Capabilities: %s\n", strings.Join(m.Capabilities, ", "))
			}
			if len(m.Permissions) > 0 {
				fmt.Fprintf(os.Stdout, "Permissions:  %s\n", strings.Join(m.Permissions, ", "))
			}
			if m.ProtocolVersion != "" {
				fmt.Fprintf(os.Stdout, "Protocol:    %s\n", m.ProtocolVersion)
			}
			return 0
		}
	}

	fmt.Fprintf(os.Stderr, "kui: plugin %q not found\n", name)
	return 1
}
