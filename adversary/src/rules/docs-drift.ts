import { type Adversary, Severity } from "@adversarylabs/sdk";
import { changedFiles, commandSurfaceFiles, hasDocsUpdate } from "../predicates.js";

export const RULE_ID = "ailloy.docs-drift";

/**
 * When the CLI's command/flag surface (internal/commands, cmd) changes but no
 * docs (docs/, README.md, AGENTS.md) change alongside it, the command tables
 * and reference docs are at risk of drifting. Deterministic finding.
 */
export function register(app: Adversary): void {
  app.rule(RULE_ID, (ctx) => {
    const files = changedFiles(ctx.change);
    if (files === null || files.length === 0) return;

    const surface = commandSurfaceFiles(files);
    if (surface.length === 0) return;
    if (hasDocsUpdate(files)) return;

    ctx.finding({
      ruleId: RULE_ID,
      groupKey: "docs-drift",
      category: "documentation",
      severity: Severity.Low,
      confidence: "medium",
      title: "Command surface changed without a docs update",
      summary:
        `This change touches the CLI command surface (${surface.length} file(s) under ` +
        "internal/commands or cmd) but updates no docs/, README.md, or AGENTS.md. New or changed " +
        "commands and flags may not be reflected in the reference docs and command tables.",
      whyItMatters:
        "ailloy's docs and the AGENTS.md command tables are how users discover behavior. " +
        "Undocumented command/flag changes surface as support burden and stale guidance.",
      evidence: surface.slice(0, 10).map((file) => ({
        file,
        message: "Command-surface file changed; no docs/README/AGENTS update in this change.",
      })),
      recommendation:
        "Reflect the command/flag change in docs/ (and the AGENTS.md command table if the set of " +
        "commands changed), or confirm the change is not user-observable.",
      tags: ["docs", "cli-surface"],
    });
  });
}
