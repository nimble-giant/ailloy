package foundries

// Tab identifies one of the four panes in the foundries TUI.
type Tab int

const (
	TabDiscover Tab = iota
	TabInstalled
	TabFoundries
	TabHealth
	tabCount
)

var tabNames = [...]string{
	TabDiscover:  "Discover",
	TabInstalled: "Installed",
	TabFoundries: "Foundries",
	TabHealth:    "Health",
}

func (t Tab) String() string { return tabNames[t] }
