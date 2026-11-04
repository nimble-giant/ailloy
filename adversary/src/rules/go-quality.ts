import { basename } from "node:path";
import { type Adversary, Severity } from "@adversarylabs/sdk";
import { isGoFile, isGoTestFile } from "../predicates.js";

export const RULE_ID = "ailloy.go-quality";

interface Pattern {
  readonly re: RegExp;
  readonly title: string;
  readonly note: string;
  /** Skip when the file's basename is in this set (e.g. os.Exit is fine in main.go). */
  readonly allowIn?: ReadonlySet<string>;
}

const PATTERNS: readonly Pattern[] = [
  {
    re: /\bos\.Exit\s*\(/,
    title: "os.Exit below the application boundary",
    note: "os.Exit bypasses deferred cleanup and makes the code hard to test; prefer returning an error to the command layer.",
    allowIn: new Set(["main.go"]),
  },
  {
    re: /\bexec\.(Command|CommandContext)\s*\(/,
    title: "external command execution",
    note: "Shelling out is an injection surface; confirm arguments are not attacker-influenced and no shell interpolation is used.",
  },
];

/**
 * Heuristic reviewer for risky Go patterns on changed files. Emits observations
 * that the CLI model broker can synthesize/prioritize when a model provider is
 * configured; the `aggregate` below guarantees deterministic findings even with
 * no model (graceful degradation). Conservative by design to limit false positives.
 */
export function register(app: Adversary): void {
  app.defineRule({
    id: RULE_ID,
    category: "go-quality",
    defaultSeverity: Severity.Low,
    defaultConfidence: "medium",
    groupBy: ["subject"],
    aggregate(observations) {
      const notes = [...new Set(observations.map((o) => String(o.metadata?.note ?? o.title)))];
      return {
        title: `Review ${observations.length} risky Go pattern(s)`,
        summary: notes.join(" "),
        recommendation: "Confirm each flagged line is intentional and safe, or refactor as noted.",
      };
    },
  });

  app.rule(RULE_ID, async (ctx) => {
    const sources = await ctx.loadInScopeSources({
      include: (path) => isGoFile(path) && !isGoTestFile(path),
    });

    for (const source of sources) {
      const name = basename(source.path);
      const lines = source.content.split("\n");
      lines.forEach((line, index) => {
        for (const pattern of PATTERNS) {
          if (pattern.allowIn?.has(name)) continue;
          if (!pattern.re.test(line)) continue;
          ctx.observe({
            ruleId: RULE_ID,
            subject: source.path,
            category: "go-quality",
            severity: Severity.Low,
            confidence: "medium",
            title: pattern.title,
            location: { file: source.path, line: index + 1 },
            evidence: line.trim().slice(0, 200),
            metadata: { note: pattern.note },
          });
        }
      });
    }
  });
}
