//go:build tools

// Package deps pins the libraries the application is built on so that
// go.mod tracks them before the code that imports them lands.
package deps

import (
	_ "charm.land/bubbles/v2/textinput"
	_ "charm.land/bubbletea/v2"
	_ "charm.land/huh/v2"
	_ "charm.land/lipgloss/v2"
	_ "github.com/BurntSushi/toml"
	_ "github.com/charmbracelet/x/exp/teatest/v2"
	_ "github.com/mdp/qrterminal/v3"
	_ "github.com/spf13/cobra"
	_ "golang.org/x/mod/semver"
)
