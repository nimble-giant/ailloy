/**
 * Pure path-classification helpers shared by the diff-aware rules.
 *
 * These are deliberately dependency-free and side-effect-free so each rule's
 * decision logic can be unit-tested directly (see test/predicates.test.ts)
 * without spinning up the Adversary runtime.
 */

import type { ChangeContext } from "@adversarylabs/sdk";

/** Repository-relative paths the runner reported as changed, or null when the
 * review has no change context (whole-target audit with no diff). */
export function changedFiles(change: ChangeContext | null | undefined): string[] | null {
  if (!change) return null;
  return [...(change.changedFiles ?? [])];
}

export function isGoFile(path: string): boolean {
  return path.endsWith(".go");
}

export function isGoTestFile(path: string): boolean {
  return path.endsWith("_test.go");
}

/**
 * "Behavior code" is non-test Go under the surfaces that back ailloy's
 * user-facing behavior: the command layer and the reusable packages.
 * AGENTS.md's standing rule ties changes here to a features.md update.
 */
export function isBehaviorCode(path: string): boolean {
  if (!isGoFile(path) || isGoTestFile(path)) return false;
  return path.startsWith("internal/commands/") || path.startsWith("pkg/");
}

export function behaviorFiles(files: readonly string[]): string[] {
  return files.filter(isBehaviorCode);
}

export function isFeaturesDoc(path: string): boolean {
  return path === "features.md";
}

export function hasFeaturesUpdate(files: readonly string[]): boolean {
  return files.some(isFeaturesDoc);
}

/**
 * "Command surface" is non-test Go that defines commands/flags — the CLI's
 * observable interface. Changes here are expected to be reflected in docs.
 */
export function isCommandSurface(path: string): boolean {
  if (!isGoFile(path) || isGoTestFile(path)) return false;
  return path.startsWith("internal/commands/") || path.startsWith("cmd/");
}

export function commandSurfaceFiles(files: readonly string[]): string[] {
  return files.filter(isCommandSurface);
}

export function isDocsUpdate(path: string): boolean {
  return path.startsWith("docs/") || path === "README.md" || path === "AGENTS.md";
}

export function hasDocsUpdate(files: readonly string[]): boolean {
  return files.some(isDocsUpdate);
}
