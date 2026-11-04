import { pathToFileURL } from "node:url";
import { Adversary } from "@adversarylabs/sdk";
import { register as registerFeaturesContract } from "./rules/features-contract.js";
import { register as registerDocsDrift } from "./rules/docs-drift.js";
import { register as registerMoldCorrectness } from "./rules/mold-correctness.js";
import { register as registerGoQuality } from "./rules/go-quality.js";

const adversary = new Adversary({
  name: "ailloy-guardian",
  version: "0.1.0",
  review: { minimumConfidence: "medium" },
});

registerFeaturesContract(adversary);
registerDocsDrift(adversary);
registerMoldCorrectness(adversary);
registerGoQuality(adversary);

export default adversary;

if (process.argv[1] !== undefined && import.meta.url === pathToFileURL(process.argv[1]).href) {
  await adversary.runFromEnvironment();
}
