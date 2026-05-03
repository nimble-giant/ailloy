package ceremony

import (
	"github.com/nimble-giant/ailloy/pkg/styles"
)

// Pre-built themes for the seven metallurgy commands. Centralizing the
// palette/glyph/strike choices here keeps each command's wiring trivial
// and ensures visual consistency across the family.

// Assay — a chemical purity test. The metaphor is the magnifying glass
// scanning the sample. One sweep beat (Strikes=1), magnifier glyph.
var Assay = Theme{
	Name:             "assay",
	Verb:             "Assaying",
	StampWord:        "ASSAY COMPLETE",
	Glyph:            "⚗️",
	Primary:          styles.Primary1,
	Accent:           styles.Accent1,
	Strikes:          1,
	StrikeChar:       "🔍",
	StrikeIntervalMs: 220,
}

// Temper — heat treatment to test strength. Three hammer strikes pace
// the entrance like a smith testing a blade.
var Temper = Theme{
	Name:             "temper",
	Verb:             "Tempering",
	StampWord:        "TEMPERED",
	Glyph:            "🔥",
	Primary:          styles.Primary1,
	Accent:           styles.Accent1,
	Strikes:          3,
	StrikeChar:       "✦",
	StrikeIntervalMs: 120,
}

// Cast — pouring molten metal. Three pour beats; the molten droplet is
// the per-beat glyph.
var Cast = Theme{
	Name:             "cast",
	Verb:             "Casting",
	StampWord:        "CAST",
	Glyph:            "🪙",
	Primary:          styles.Primary1,
	Accent:           styles.Accent1,
	Strikes:          3,
	StrikeChar:       "•",
	StrikeIntervalMs: 130,
}

// Recast — re-melting and re-pouring. Two strikes, dimmer accent.
var Recast = Theme{
	Name:             "recast",
	Verb:             "Recasting",
	StampWord:        "RECAST",
	Glyph:            "♻️",
	Primary:          styles.Primary1,
	Accent:           styles.Accent2,
	Strikes:          2,
	StrikeChar:       "•",
	StrikeIntervalMs: 140,
}

// Quench — plunging hot metal into water. One snap beat, snowflake.
// This is the "freeze" command — the brief beat reads as a pause.
var Quench = Theme{
	Name:             "quench",
	Verb:             "Quenching",
	StampWord:        "QUENCHED",
	Glyph:            "❄️",
	Primary:          styles.Info,
	Accent:           styles.White,
	Strikes:          1,
	StrikeChar:       "≈",
	StrikeIntervalMs: 200,
}

// Smelt — extracting refined ingots. Three furnace pulses.
var Smelt = Theme{
	Name:             "smelt",
	Verb:             "Smelting",
	StampWord:        "INGOT READY",
	Glyph:            "📦",
	Primary:          styles.Accent1,
	Accent:           styles.Warning,
	Strikes:          3,
	StrikeChar:       "▒",
	StrikeIntervalMs: 130,
}

// Forge — hammer striking anvil. The brief preview beat.
var Forge = Theme{
	Name:             "forge",
	Verb:             "Forging",
	StampWord:        "PREVIEW READY",
	Glyph:            "🔨",
	Primary:          styles.Primary1,
	Accent:           styles.Accent1,
	Strikes:          3,
	StrikeChar:       "✦",
	StrikeIntervalMs: 110,
}
