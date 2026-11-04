import { type Adversary, Severity } from "@adversarylabs/sdk";
import { behaviorFiles, changedFiles, hasFeaturesUpdate } from "../predicates.js";

export const RULE_ID = "ailloy.features-contract";

/**
 * AGENTS.md sets a standing rule: any change to user-facing behavior must
 * update features.md in the SAME change. This rule flags PRs that touch
 * behavior code (internal/commands, pkg) without a corresponding features.md
 * edit. Deterministic — emits a finding directly, no model needed.
 */
export function register(app: Adversary): void {
  app.rule(RULE_ID, (ctx) => {
    const files = changedFiles(ctx.change);
    if (files === null || files.length === 0) return; // whole-target audit: nothing to assert

    const behavior = behaviorFiles(files);
    if (behavior.length === 0) return; // no behavior code touched
    if (hasFeaturesUpdate(files)) return; // contract satisfied

    ctx.finding({
      ruleId: RULE_ID,
      groupKey: "features-contract",
      category: "contract",
      severity: Severity.Medium,
      confidence: "high",
      title: "Behavior changed without a features.md update",
      summary:
        `This change touches behavior code (${behavior.length} file(s)) but does not update ` +
        "features.md. AGENTS.md makes \"keep features.md current\" a standing rule and part of " +
        "the definition of done: every user-facing behavior change must update the contract in " +
        "the same change.",
      whyItMatters:
        "features.md is the regression-prevention contract and the seed for test coverage. " +
        "Letting it drift from the code silently erodes the guarantees the rest of the repo relies on.",
      evidence: behavior.slice(0, 10).map((file) => ({
        file,
        message: "Behavior code changed here; no matching features.md update in this change.",
      })),
      recommendation:
        "Update features.md to describe the new/changed behavior (or confirm the change is " +
        "purely internal and note why no contract update is needed).",
      tags: ["features-md", "definition-of-done"],
    });
  });
}
