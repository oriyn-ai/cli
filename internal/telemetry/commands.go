package telemetry

// This file is the typed allowlist of every top-level CLI command.
// Adding a new command to oriyn means adding both:
//
//   1. A `TrackCliCommand{Name}(subcommand string)` method here.
//   2. A matching case in the dispatch switch in cmd/root.go.
//
// Mirrors Vercel's RootTelemetryClient pattern: the trackCli{Command}
// method names form a code-review-gated allowlist of what can be
// captured. Since the underlying `trackCommand` helper is unexported,
// external callers cannot bypass the allowlist by passing a raw string.

// TrackCliCommandInit records an `oriyn init` invocation.
func (c *Client) TrackCliCommandInit(subcommand string) {
	c.trackCommand("init", subcommand)
}

// TrackCliCommandLogin records an `oriyn login` invocation.
func (c *Client) TrackCliCommandLogin(subcommand string) {
	c.trackCommand("login", subcommand)
}

// TrackCliCommandLogout records an `oriyn logout` invocation.
func (c *Client) TrackCliCommandLogout(subcommand string) {
	c.trackCommand("logout", subcommand)
}

// TrackCliCommandWhoami records an `oriyn whoami` invocation.
func (c *Client) TrackCliCommandWhoami(subcommand string) {
	c.trackCommand("whoami", subcommand)
}

// TrackCliCommandDoctor records an `oriyn doctor` invocation.
func (c *Client) TrackCliCommandDoctor(subcommand string) {
	c.trackCommand("doctor", subcommand)
}

// TrackCliCommandSkill records an `oriyn skill ...` invocation.
func (c *Client) TrackCliCommandSkill(subcommand string) {
	c.trackCommand("skill", subcommand)
}

// TrackCliCommandProducts records an `oriyn products ...` invocation.
func (c *Client) TrackCliCommandProducts(subcommand string) {
	c.trackCommand("products", subcommand)
}

// TrackCliCommandPersonas records an `oriyn personas ...` invocation.
func (c *Client) TrackCliCommandPersonas(subcommand string) {
	c.trackCommand("personas", subcommand)
}

// TrackCliCommandHypotheses records an `oriyn hypotheses ...` invocation.
func (c *Client) TrackCliCommandHypotheses(subcommand string) {
	c.trackCommand("hypotheses", subcommand)
}

// TrackCliCommandKnowledge records an `oriyn knowledge ...` invocation.
func (c *Client) TrackCliCommandKnowledge(subcommand string) {
	c.trackCommand("knowledge", subcommand)
}

// TrackCliCommandTimeline records an `oriyn timeline` invocation.
func (c *Client) TrackCliCommandTimeline(subcommand string) {
	c.trackCommand("timeline", subcommand)
}

// TrackCliCommandReplay records an `oriyn replay` invocation.
func (c *Client) TrackCliCommandReplay(subcommand string) {
	c.trackCommand("replay", subcommand)
}

// TrackCliCommandSynthesize records an `oriyn synthesize` invocation.
func (c *Client) TrackCliCommandSynthesize(subcommand string) {
	c.trackCommand("synthesize", subcommand)
}

// TrackCliCommandEnrich records an `oriyn enrich` invocation.
func (c *Client) TrackCliCommandEnrich(subcommand string) {
	c.trackCommand("enrich", subcommand)
}

// TrackCliCommandExperiment records an `oriyn experiment ...` invocation.
func (c *Client) TrackCliCommandExperiment(subcommand string) {
	c.trackCommand("experiment", subcommand)
}

// TrackCliCommandTelemetry records an `oriyn telemetry ...` invocation.
func (c *Client) TrackCliCommandTelemetry(subcommand string) {
	c.trackCommand("telemetry", subcommand)
}

// TrackCliCommandRoot records the root binary being invoked with no
// recognized subcommand (e.g. `oriyn` alone, or an unknown subcommand
// that fell through to the help-on-error path).
func (c *Client) TrackCliCommandRoot(subcommand string) {
	c.trackCommand("root", subcommand)
}
