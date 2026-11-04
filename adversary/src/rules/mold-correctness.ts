import { readFile } from "node:fs/promises";
import { dirname, join, posix } from "node:path";
import { type Adversary, Severity } from "@adversarylabs/sdk";
import { parse } from "yaml";

export const RULE_ID = "ailloy.mold-correctness";

const SKIP_SEGMENTS = ["node_modules/", "adversary/", "dist/"];

function shouldSkip(path: string): boolean {
  return SKIP_SEGMENTS.some((seg) => path.startsWith(seg) || path.includes(`/${seg}`));
}

/**
 * Match a manifest both at the target root and nested. `**\/name` does not match
 * a root-level file (`**` requires a directory segment), and molds routinely put
 * their manifest at the root, so we glob both and dedupe.
 */
async function findManifests(
  rglob: (pattern: string) => Promise<string[]>,
  name: string,
): Promise<string[]> {
  const [root, nested] = await Promise.all([rglob(name), rglob(`**/${name}`)]);
  return [...new Set([...root, ...nested])].filter((p) => !shouldSkip(p));
}

async function loadYaml(repoPath: string, rel: string): Promise<Record<string, unknown> | null> {
  try {
    const raw = await readFile(join(repoPath, rel), "utf8");
    const doc = parse(raw);
    return doc && typeof doc === "object" ? (doc as Record<string, unknown>) : {};
  } catch {
    return null;
  }
}

function nonEmptyString(value: unknown): boolean {
  return typeof value === "string" && value.trim().length > 0;
}

/**
 * Correctness scan of ailloy's bundled mold/ingot/ore manifests (an
 * AI-instructions package manager should catch these about itself):
 *   - mold.yaml / ore.yaml must declare name + version
 *   - ore.yaml must declare kind: ore
 *   - ingot.yaml `files:` entries must exist on disk
 * Deterministic findings — runs against whatever manifests exist in the target.
 */
export function register(app: Adversary): void {
  app.rule(RULE_ID, async (ctx) => {
    const [molds, ores, ingots] = await Promise.all([
      findManifests(ctx.rglob, "mold.yaml"),
      findManifests(ctx.rglob, "ore.yaml"),
      findManifests(ctx.rglob, "ingot.yaml"),
    ]);

    for (const rel of [...molds, ...ores]) {
      const doc = await loadYaml(ctx.repoPath, rel);
      if (doc === null) continue;
      const missing: string[] = [];
      if (!nonEmptyString(doc.name)) missing.push("name");
      if (!nonEmptyString(doc.version)) missing.push("version");
      if (rel.endsWith("ore.yaml") && doc.kind !== "ore") missing.push('kind: ore');
      if (missing.length > 0) {
        ctx.finding({
          ruleId: RULE_ID,
          // groupKey must be unique per issue: the SDK derives a finding's dedupe
          // id from `ruleId:groupKey ?? category`, so findings that share a
          // category collapse into one without a distinct groupKey.
          groupKey: `mold-fields:${rel}`,
          category: "mold",
          severity: Severity.Medium,
          confidence: "high",
          title: `Manifest missing required field(s): ${missing.join(", ")}`,
          summary: `${rel} is missing ${missing.join(", ")}. ailloy requires these to resolve and render the package.`,
          evidence: [{ file: rel, message: `Missing: ${missing.join(", ")}` }],
          recommendation: `Add the missing field(s) to ${rel}.`,
          tags: ["mold", "manifest"],
        });
      }
    }

    for (const rel of ingots) {
      const doc = await loadYaml(ctx.repoPath, rel);
      if (doc === null) continue;
      const files = Array.isArray(doc.files) ? (doc.files as unknown[]) : [];
      const baseDir = dirname(rel);
      for (const entry of files) {
        if (!nonEmptyString(entry)) continue;
        const target = posix.normalize(posix.join(baseDir, entry as string));
        try {
          await readFile(join(ctx.repoPath, target));
        } catch {
          ctx.finding({
            ruleId: RULE_ID,
            groupKey: `ingot-missing:${rel}:${target}`,
            category: "mold",
            severity: Severity.Medium,
            confidence: "high",
            title: "Ingot references a file that does not exist",
            summary: `${rel} lists "${entry}" in files:, but ${target} was not found. Casting this ingot would fail.`,
            evidence: [{ file: rel, message: `Missing referenced file: ${target}` }],
            recommendation: `Create ${target} or remove the entry from ${rel}.`,
            tags: ["mold", "ingot"],
          });
        }
      }
    }
  });
}
