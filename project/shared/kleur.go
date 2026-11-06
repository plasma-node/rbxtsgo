// Package shared is the port of src/Shared from roblox-ts 3.0.0: constants, project types, diagnostics
// definitions, and small utilities used by both the project layer and the transformer.
package shared

import (
	"os"
	"strings"

	"golang.org/x/term"
)

// Kleur decides whether colors are applied, like the kleur package. Colors end up inside diagnostic message text,
// because upstream calls kleur.yellow while building the messages. So the same diagnostic has different bytes
// depending on the environment.
type Kleur struct {
	Enabled bool
}

// DetectKleur follows kleur 4.1.5, which reads the environment once at load.
func DetectKleur() Kleur {
	forceColor, hasForceColor := os.LookupEnv("FORCE_COLOR")
	_, hasNoColor := os.LookupEnv("NO_COLOR")
	enabled := os.Getenv("NODE_DISABLE_COLORS") == "" && !hasNoColor && os.Getenv("TERM") != "dumb" &&
		((hasForceColor && forceColor != "0") || term.IsTerminal(int(os.Stdout.Fd())))
	return Kleur{Enabled: enabled}
}

func (k Kleur) wrap(open, close, text string) string {
	if !k.Enabled {
		return text
	}
	// kleur re-opens the color after any close sequence already in the text.
	return open + strings.ReplaceAll(text, close, close+open) + close
}

// Yellow is kleur.yellow.
func (k Kleur) Yellow(text string) string { return k.wrap("\x1b[33m", "\x1b[39m", text) }

// Grey is kleur.grey.
func (k Kleur) Grey(text string) string { return k.wrap("\x1b[90m", "\x1b[39m", text) }
