# Changelog

## [0.6.21](https://github.com/nimble-giant/ailloy/compare/v0.6.20...v0.6.21) (2026-05-03)


### Bug Fixes

* temper skips template validation for process:false outputs ([d7deea1](https://github.com/nimble-giant/ailloy/commit/d7deea1144c5189c873a4b4e9de96de337908186)), closes [#165](https://github.com/nimble-giant/ailloy/issues/165)

## [0.6.20](https://github.com/nimble-giant/ailloy/compare/v0.6.19...v0.6.20) (2026-05-03)


### Features

* add ailloy self-upgrade command ([b9471d7](https://github.com/nimble-giant/ailloy/commit/b9471d7b7fe46d4db6053f4b891819f2dda7449e))
* add foundry temper command for index validation ([974e33d](https://github.com/nimble-giant/ailloy/commit/974e33d652b08ca924b806998c54f7ccb6dcbba7))
* add reinstall alias to evolve ([aa564b9](https://github.com/nimble-giant/ailloy/commit/aa564b9a1f011f6bc748eaef4d7874b11e996597))
* nested foundries — transitive mold aggregation across foundry indexes ([d1bdd54](https://github.com/nimble-giant/ailloy/commit/d1bdd548103e9341159d26265ff508930685540b))
* rename refine to evolve and add Pokemon-style TUI animation ([c1018a1](https://github.com/nimble-giant/ailloy/commit/c1018a16394e5200d959b7553e10183205cc578a))


### Bug Fixes

* identify installed molds by (source, subpath), not source alone ([5f6b13c](https://github.com/nimble-giant/ailloy/commit/5f6b13ce3cfc10c96042743b0ba3128addbf2f05))

## [0.6.19](https://github.com/nimble-giant/ailloy/compare/v0.6.18...v0.6.19) (2026-05-02)


### Features

* add --as-plugin flag to cast for claude code plugin output ([3964197](https://github.com/nimble-giant/ailloy/commit/3964197780c02c2a44e139e20089c2f2aeaac64d))
* add ailloy uninstall command ([7ec6f09](https://github.com/nimble-giant/ailloy/commit/7ec6f094b2b62cddfc287fe9feaf38246c760c80))
* add Files manifest to LockEntry ([210d82d](https://github.com/nimble-giant/ailloy/commit/210d82d7ded2ccfca359ad606965a0032e54c5d7))
* add FoundryForSource lookup helper ([fb29e1b](https://github.com/nimble-giant/ailloy/commit/fb29e1b9be12f0c65cccfe949072aa12e6281a55))
* add Homebrew installer ([a5af99d](https://github.com/nimble-giant/ailloy/commit/a5af99d0c4080f01f0de31b7e400c821bc37e79d))
* add installed manifest types and YAML I/O ([dcf85a7](https://github.com/nimble-giant/ailloy/commit/dcf85a73c273986d9f4fc60d1c89cbd3e938aa09))
* add interactive ailloy foundries TUI ([0b2e4c4](https://github.com/nimble-giant/ailloy/commit/0b2e4c4b662f17a6ae5e36c767a6f44559f3e0e9))
* add UninstallMold with manifest-based safe deletion ([356575a](https://github.com/nimble-giant/ailloy/commit/356575a29a2199c1b6547b45041833858a5e485b))
* add UpsertEntry/FindBySource/FindByName to manifest ([e3073eb](https://github.com/nimble-giant/ailloy/commit/e3073ebfa6f9515a71612ce426c8290b2371e0a6))
* brand the foundries TUI with Ailloy palette + vim h/l for tabs ([d4f9b2b](https://github.com/nimble-giant/ailloy/commit/d4f9b2b3f00e07122234363abddc9362585e6af4))
* default to https when adding a foundry without a protocol ([d9168b7](https://github.com/nimble-giant/ailloy/commit/d9168b7a4b9fd37f6fe5e339bcfab3cf084b972a))
* drive recast from installed manifest, update lock if present ([b5b6606](https://github.com/nimble-giant/ailloy/commit/b5b66066a49da6884c4c9a40bd54da740468ba06))
* install every mold in a foundry via CLI and TUI ([cbcc17b](https://github.com/nimble-giant/ailloy/commit/cbcc17b3221c358047512f07932e61f59474b939))
* record ingot installs in installed manifest ([4c924ed](https://github.com/nimble-giant/ailloy/commit/4c924ed1881ae62433e0d4018d925c0e60d06451))
* record installed files and hashes during cast ([184a0a5](https://github.com/nimble-giant/ailloy/commit/184a0a58609a8dfd54b76063b8798b570c4e407e))
* rewrite quench as opt-in pin/verify driven by installed manifest ([f616520](https://github.com/nimble-giant/ailloy/commit/f616520cef23e336cc1d4877f19322946305cb75))
* show installed badge on Discover rows ([1e2605d](https://github.com/nimble-giant/ailloy/commit/1e2605d75c7ef6170d221ed1c2959c0eaf096770))
* thread --claude-plugin through CastMold and foundry install ([2f86aa9](https://github.com/nimble-giant/ailloy/commit/2f86aa9ea18e6e688d40b7fd16b989f1faec6b71))
* write installed manifest after successful cast ([2950aed](https://github.com/nimble-giant/ailloy/commit/2950aed074c665947996aa8ea20385e06aa036f1))


### Bug Fixes

* adapt cast --as-plugin to buildIngotResolver(flux, root) signature ([7744364](https://github.com/nimble-giant/ailloy/commit/7744364b33a136fe0024d797e485a93be66c1cff))
* appease golangci-lint errcheck and staticcheck ([9d22274](https://github.com/nimble-giant/ailloy/commit/9d22274e661c1b450bd8df180640e29547f63beb))
* auto-fetch foundry indexes when Discover catalog is empty ([6546a39](https://github.com/nimble-giant/ailloy/commit/6546a39ec86e413b8aa6dbd9104b6c790b4fbe3a))
* broadcast non-key messages to all tabs ([41010c7](https://github.com/nimble-giant/ailloy/commit/41010c7485ff17cdd629b8770cb7eb41d359da21))
* clean up empty directories left by skipped renders ([1081164](https://github.com/nimble-giant/ailloy/commit/1081164c918ba8252fccce0dfdf0ec702050dcc4)), closes [#145](https://github.com/nimble-giant/ailloy/issues/145)
* fetch tags when refreshing the foundry bare clone ([dc92250](https://github.com/nimble-giant/ailloy/commit/dc922502d96387d8335d76fd1067ce15c5a90da2))
* globalLockPath returns empty when home is unresolvable ([53dbd07](https://github.com/nimble-giant/ailloy/commit/53dbd073815f85cbaf83653e7295ea60f0e21b07))
* include the verified default in foundry searches ([fa29d19](https://github.com/nimble-giant/ailloy/commit/fa29d1916638a77402417e36982033a620f167c6))
* log corrupt-manifest reset and clarify Metadata error contract ([7b13c7b](https://github.com/nimble-giant/ailloy/commit/7b13c7baf620b8c80dc7c58d3752260f796285b1))
* report scoped pin count and drop placeholder --rescan flag ([5d8baee](https://github.com/nimble-giant/ailloy/commit/5d8baeef569f71c999fa3242e4f31e9de4261937))
* require existing lock for scoped quench and stop double-reporting drift ([a4cf8b0](https://github.com/nimble-giant/ailloy/commit/a4cf8b007519a277a27976aabf4784e40cfecad5))
* silence cast output during TUI installs ([ba08742](https://github.com/nimble-giant/ailloy/commit/ba087424ca515ffc2253be187f26a8bacd3b67d3))
* skip recast manifest update when fetch fails ([c0dc828](https://github.com/nimble-giant/ailloy/commit/c0dc828f5aa58d9282d5a950a177f1852326994c))

## [0.6.18](https://github.com/nimble-giant/ailloy/compare/v0.6.17...v0.6.18) (2026-05-01)


### Features

* add foundry new command to scaffold foundry indexes ([4d275f5](https://github.com/nimble-giant/ailloy/commit/4d275f50b6ff686527f5d7dbcd5ded0862653530))
* add verified badge for molds from official nimble-giant foundry ([6c97f78](https://github.com/nimble-giant/ailloy/commit/6c97f78177ecb82dbef1939eb89f2e12b0e44cbc))
* implement foundry index system for SCM-agnostic mold discovery ([#77](https://github.com/nimble-giant/ailloy/issues/77)) ([6bc906c](https://github.com/nimble-giant/ailloy/commit/6bc906c6fb7fb9daa8a22e8dd373043ccfae7969))


### Bug Fixes

* include mold source root in ingot search path ([bc7f9d4](https://github.com/nimble-giant/ailloy/commit/bc7f9d488ae629f5c0d7b0faed8ec5607a428717)), closes [#140](https://github.com/nimble-giant/ailloy/issues/140)
* update CI to Go 1.26 and golangci-lint v2 for vulnerability fix ([85add44](https://github.com/nimble-giant/ailloy/commit/85add4401dfcaeeb698c067e1bdc4f1a04093095))
* use golangci-lint-action v7 for golangci-lint v2 support ([3557abe](https://github.com/nimble-giant/ailloy/commit/3557abea0ce71968be47edcdf4e2f26006219e21))

## [0.6.17](https://github.com/nimble-giant/ailloy/compare/v0.6.16...v0.6.17) (2026-05-01)


### Features

* multi-destination output mapping with per-dest context ([8fb2e98](https://github.com/nimble-giant/ailloy/commit/8fb2e98e0c534bfb64f595d4a7bdaed9fd45b1d4)), closes [#135](https://github.com/nimble-giant/ailloy/issues/135)

## [0.6.16](https://github.com/nimble-giant/ailloy/compare/v0.6.15...v0.6.16) (2026-05-01)


### Bug Fixes

* skip .md files outside output manifest during template parsing ([1f16073](https://github.com/nimble-giant/ailloy/commit/1f160733ea8b1fa4705783de7c1736e8387a5990)), closes [#126](https://github.com/nimble-giant/ailloy/issues/126)
* skip writing files that render to empty content ([81ad690](https://github.com/nimble-giant/ailloy/commit/81ad690c86308127d218776bc71593ba9a776d68)), closes [#130](https://github.com/nimble-giant/ailloy/issues/130)
* yaml-parse --set values to support array types ([26ba0bb](https://github.com/nimble-giant/ailloy/commit/26ba0bb2c5dd04d08c947beeb5e70c3d40d63a78)), closes [#129](https://github.com/nimble-giant/ailloy/issues/129)

## [0.6.15](https://github.com/nimble-giant/ailloy/compare/v0.6.14...v0.6.15) (2026-04-30)


### Bug Fixes

* **assay:** resolve staticcheck QF1012 issues surfaced by golangci-lint upgrade ([1ba792c](https://github.com/nimble-giant/ailloy/commit/1ba792c28d37c3bf66add90528881526d81e1a86))
* enhance foundry resolution and mold templates ([fa57ade](https://github.com/nimble-giant/ailloy/commit/fa57adeee3076ea56c709aff983c04ffe9beb76c))

## [0.6.14](https://github.com/nimble-giant/ailloy/compare/v0.6.13...v0.6.14) (2026-04-18)


### Features

* add --lint flag to temper command for assay linting ([6a44463](https://github.com/nimble-giant/ailloy/commit/6a444630236ffe4a710548e0be75854b3d8bf699))
* add 'has' template function ([a47fc61](https://github.com/nimble-giant/ailloy/commit/a47fc612136dcc638763b8c9809faa388daa0d4c))
* implements option to skip lock file resolution ([cda41e4](https://github.com/nimble-giant/ailloy/commit/cda41e41a7e947b2b5047f5a12d3c704f18b7285))


### Bug Fixes

* address lint issues in temper --lint implementation ([ee18fb1](https://github.com/nimble-giant/ailloy/commit/ee18fb1c0fa0731ae0e60a9273e64c413cff450d))

## [0.6.13](https://github.com/nimble-giant/ailloy/compare/v0.6.12...v0.6.13) (2026-04-01)


### Features

* add context-usage lint rule with token estimation and rollups ([0110c58](https://github.com/nimble-giant/ailloy/commit/0110c58bc560de6c9493afded6a74607ec6ca75f))

## [0.6.12](https://github.com/nimble-giant/ailloy/compare/v0.6.11...v0.6.12) (2026-03-24)


### Features

* add 'has' function to template engine ([0a25383](https://github.com/nimble-giant/ailloy/commit/0a2538384c2082342a812b243d3e7e0abc846b36))

## [0.6.11](https://github.com/nimble-giant/ailloy/compare/v0.6.10...v0.6.11) (2026-03-24)


### Features

* implement file ignoring mechanism with .ailloyignore support ([c75098d](https://github.com/nimble-giant/ailloy/commit/c75098dd720f05ce82634f877aaf71661278c99d))

## [0.6.10](https://github.com/nimble-giant/ailloy/compare/v0.6.9...v0.6.10) (2026-03-19)


### Features

* add 5 new skill lint rules and improve cross-platform support ([de5f5a8](https://github.com/nimble-giant/ailloy/commit/de5f5a83a7ee5a228d7a33a40e50324e60cdf7ef))
* add description-length lint rule ([b65d605](https://github.com/nimble-giant/ailloy/commit/b65d605788d896da913ae14820f888d0a652bcc2))

## [0.6.9](https://github.com/nimble-giant/ailloy/compare/v0.6.8...v0.6.9) (2026-03-18)


### Features

* add Claude plugin directory support to assay lint ([ec38ed5](https://github.com/nimble-giant/ailloy/commit/ec38ed5118c2d7462594d42f4824f6fb88ee91cc))
* **assay:** agents-md tip, ailloy config allow-fields, lint --fix ([ec38ed5](https://github.com/nimble-giant/ailloy/commit/ec38ed5118c2d7462594d42f4824f6fb88ee91cc))
* **assay:** group console output by rule with educational rationale headers ([ec38ed5](https://github.com/nimble-giant/ailloy/commit/ec38ed5118c2d7462594d42f4824f6fb88ee91cc))
* **assay:** structured multi-line console output with first-class tips ([ec38ed5](https://github.com/nimble-giant/ailloy/commit/ec38ed5118c2d7462594d42f4824f6fb88ee91cc))
* extend plugin lint to cover skills, rules, and hooks ([ec38ed5](https://github.com/nimble-giant/ailloy/commit/ec38ed5118c2d7462594d42f4824f6fb88ee91cc))
* make plugin file collection recursive for marketplace support ([ec38ed5](https://github.com/nimble-giant/ailloy/commit/ec38ed5118c2d7462594d42f4824f6fb88ee91cc))


### Bug Fixes

* **assay:** collapse unknown-frontmatter warnings and add extra-allowed-fields ([ec38ed5](https://github.com/nimble-giant/ailloy/commit/ec38ed5118c2d7462594d42f4824f6fb88ee91cc))
* **assay:** correct CI failures in formatter test and multiline detection ([ec38ed5](https://github.com/nimble-giant/ailloy/commit/ec38ed5118c2d7462594d42f4824f6fb88ee91cc))
* **assay:** detect multiline frontmatter fields that Claude Code rejects ([ec38ed5](https://github.com/nimble-giant/ailloy/commit/ec38ed5118c2d7462594d42f4824f6fb88ee91cc))
* **assay:** structure rule false positive on files with frontmatter ([ec38ed5](https://github.com/nimble-giant/ailloy/commit/ec38ed5118c2d7462594d42f4824f6fb88ee91cc))
* **assay:** suggest [@filepath](https://github.com/filepath) syntax in duplicate-topics diagnostic ([ec38ed5](https://github.com/nimble-giant/ailloy/commit/ec38ed5118c2d7462594d42f4824f6fb88ee91cc))
* **assay:** suppress gosec G122 on existing nosec-annotated ReadFile calls ([ec38ed5](https://github.com/nimble-giant/ailloy/commit/ec38ed5118c2d7462594d42f4824f6fb88ee91cc))

## [0.6.8](https://github.com/nimble-giant/ailloy/compare/v0.6.7...v0.6.8) (2026-03-17)


### Features

* add assay command to lint AI instruction files ([d6b588b](https://github.com/nimble-giant/ailloy/commit/d6b588b8bfb1c360a3921852186ad64adf282ca3))


### Bug Fixes

* add WorkingBanner to forge command before processing ([eaea78d](https://github.com/nimble-giant/ailloy/commit/eaea78dd844f5728bea204533c994af9d668ae26)), closes [#88](https://github.com/nimble-giant/ailloy/issues/88)
* remove release-as pin to stop v0.6.7 re-release loop ([aa27597](https://github.com/nimble-giant/ailloy/commit/aa27597da27d812de152b76a58c322cc4cbb947c))
* replace ! warning icon with ⚠️ in recast command ([e9e63fd](https://github.com/nimble-giant/ailloy/commit/e9e63fdb3fa2de5cd55d7c663b7ccd8d72c57956)), closes [#86](https://github.com/nimble-giant/ailloy/issues/86)
* resolve CI failures from Go 1.25 upgrade ([0e25785](https://github.com/nimble-giant/ailloy/commit/0e25785fced65deeb8fd4339ea4848bfd41b6b06))
* title-case file-write confirmation in forge command ([b4b75d9](https://github.com/nimble-giant/ailloy/commit/b4b75d93f6d7510e1306c5874a917e56a44e811f)), closes [#89](https://github.com/nimble-giant/ailloy/issues/89)
* upgrade Go to 1.25.8 to resolve stdlib security vulnerabilities ([c42a5cf](https://github.com/nimble-giant/ailloy/commit/c42a5cf625bb2ee1f9c17f1e57e9cd0c905eaa52))
* use ErrorStyle for Execute() error output in root.go ([074fbf1](https://github.com/nimble-giant/ailloy/commit/074fbf1239af8b72b51baefd6bc91e99fd9fed23)), closes [#90](https://github.com/nimble-giant/ailloy/issues/90)
* wrap mold new next steps output in InfoBoxStyle border ([f84de57](https://github.com/nimble-giant/ailloy/commit/f84de57311dd82efd34f26a3ddfeda59d25a75ca)), closes [#87](https://github.com/nimble-giant/ailloy/issues/87)

## [0.6.7](https://github.com/nimble-giant/ailloy/compare/v0.6.6...v0.6.7) (2026-02-24)


### Features

* add --global flag to cast for user-level installs ([c162c3a](https://github.com/nimble-giant/ailloy/commit/c162c3a53d87d8a4c144ab8158e54f4b03c908c2))
* add AGENTS.md support to ailloy molds ([6f7a635](https://github.com/nimble-giant/ailloy/commit/6f7a63542fa230fbfcb541e56d0c0a2b4d763aae))
* add brainstorm command for structured idea analysis ([5dd7aba](https://github.com/nimble-giant/ailloy/commit/5dd7aba1efed41e0a6cb4402ab8956825e2a5e02))
* add Claude Code workflow template with --with-workflows flag ([5197c98](https://github.com/nimble-giant/ailloy/commit/5197c98cb19814f962a16d9b3e71bf5504debd98))
* add CODEOWNERS for nimble-giant/engineers ([0d4f3ec](https://github.com/nimble-giant/ailloy/commit/0d4f3ecbee4d1312e63daf0fda5881200aeeca91))
* add forge command for dry-run template rendering ([#42](https://github.com/nimble-giant/ailloy/issues/42)) ([3ed442b](https://github.com/nimble-giant/ailloy/commit/3ed442bd7da4ee215236ce499132df2e87003123))
* add foundry, mold get, and ingot get/add commands ([04b05a7](https://github.com/nimble-giant/ailloy/commit/04b05a716b60d01866486e99a9f9a93bf58d079c)), closes [#47](https://github.com/nimble-giant/ailloy/issues/47)
* add mold new command to scaffold mold directories ([e543e2d](https://github.com/nimble-giant/ailloy/commit/e543e2dc3f07ac4d88b28b8b5571cdc734d3cb9d))
* add recast and quench commands ([ae03379](https://github.com/nimble-giant/ailloy/commit/ae0337959e594ff11c839d5374a8c0b9b645dc96)), closes [#48](https://github.com/nimble-giant/ailloy/issues/48)
* add skill for creating new ailloy templates ([f797c87](https://github.com/nimble-giant/ailloy/commit/f797c87580b6a6d36cf8f3157545f6b8b811450a))
* add smelt command for mold packaging ([#45](https://github.com/nimble-giant/ailloy/issues/45)) ([78279de](https://github.com/nimble-giant/ailloy/commit/78279dea2a09acd00249c7694fd264ac84a63797))
* add temper command for mold/ingot validation ([bc2ac37](https://github.com/nimble-giant/ailloy/commit/bc2ac370fe6f0d0f8ac7495bfcc8ad5e2c06804a)), closes [#43](https://github.com/nimble-giant/ailloy/issues/43)
* adds ci, release, and security badges ([bcfc25a](https://github.com/nimble-giant/ailloy/commit/bcfc25a62574eb1457dc26c9f6c6a6e93d9d6fb0))
* allows ignoring templates ([a3cd65c](https://github.com/nimble-giant/ailloy/commit/a3cd65ca368a155bf701fe7f20ad365c6261a853))
* brainstorm skill ([2100774](https://github.com/nimble-giant/ailloy/commit/2100774e188f525826ef511f2d9642c16a1e614c))
* claude code review agent workflow for github ([e93cc91](https://github.com/nimble-giant/ailloy/commit/e93cc91acb4f29aadd507f81e88dc86ab582293e))
* convention-based mold structure with output path mappings ([bc1a460](https://github.com/nimble-giant/ailloy/commit/bc1a4608c9f501102aa9caecffe77d8e484e8eda))
* define mold.yaml and ingot.yaml manifest formats ([#40](https://github.com/nimble-giant/ailloy/issues/40)) ([12d48c4](https://github.com/nimble-giant/ailloy/commit/12d48c4ef920f6891c1fd6e95620877d1eeee48f))
* helm-style flux.yaml and optional flux.schema.yaml ([c217c02](https://github.com/nimble-giant/ailloy/commit/c217c028f61c611a69b8d3b252a873ac5d9738fa))
* implement brainstorm as embedded ailloy template ([d05d21f](https://github.com/nimble-giant/ailloy/commit/d05d21fad63bc565765181eb188f702fb77d6138))
* implement dynamic build information display in cli ([9f79c2f](https://github.com/nimble-giant/ailloy/commit/9f79c2f12ed9b2c288a0b6d3101b894ae362baf8))
* implement stuffbin-based binary output for smelt ([f834463](https://github.com/nimble-giant/ailloy/commit/f834463153b3af19ff211471e06f153c5bf99816))
* ingot template partials and flux schema validation ([#44](https://github.com/nimble-giant/ailloy/issues/44)) ([68bacff](https://github.com/nimble-giant/ailloy/commit/68bacff33fc8153d8ab7cd71835ee7ccf9083e85))
* introduce semantic model layer for generations ([a9b8769](https://github.com/nimble-giant/ailloy/commit/a9b8769b8aad47ae69315dfe0071678471875997))
* make ailloy tool-agnostic across AI coding tools ([e69b0e1](https://github.com/nimble-giant/ailloy/commit/e69b0e1a13e0228f3321959828c7abfc7850e356))
* rebuild interactive ux ([ba66237](https://github.com/nimble-giant/ailloy/commit/ba66237af120a53dacfee4c28108407f5d21b0bc))
* refactor anneal as dynamic mold-agnostic configuration wizard ([193251a](https://github.com/nimble-giant/ailloy/commit/193251a474edcb06da3e5201d9b9aa7ff7f6897e))
* scm-native foundry resolution and local cache for remote molds ([e2bc197](https://github.com/nimble-giant/ailloy/commit/e2bc1975eff4ff5ef9263cc26a45676563459101))
* ship it 🚀 — ailloy v0.0.0 ([0550117](https://github.com/nimble-giant/ailloy/commit/055011784e26333bbc1e2e15baf6770929698c75))
* surface command aliases in help text using lipgloss table ([8014da5](https://github.com/nimble-giant/ailloy/commit/8014da592e9537d973abf61cd1195507debaf0f5))
* upgrade template engine with conditionals and model rendering ([e5adcf0](https://github.com/nimble-giant/ailloy/commit/e5adcf06ea1d0102a3581510795eef209734d0c3)), closes [#26](https://github.com/nimble-giant/ailloy/issues/26)


### Bug Fixes

* add gosec G104 nosec directives to mergo.Merge calls ([7bab98c](https://github.com/nimble-giant/ailloy/commit/7bab98cb2400bbaaf22d67c9935e995e05339482))
* add nosec annotations for GoSec G304 in manifest loaders ([032ccba](https://github.com/nimble-giant/ailloy/commit/032ccba241222d8231eb986eac76c87ae6e47086))
* address errcheck lint issue in smelt binary packaging ([a428687](https://github.com/nimble-giant/ailloy/commit/a428687731c8d18dfab3f6f8941d4e7598f0c829))
* address errcheck lint issues in smelt package ([f22be39](https://github.com/nimble-giant/ailloy/commit/f22be39561490cc4362e217390114a49b6fad9bf))
* auto-discover root files in map output form ([0765de5](https://github.com/nimble-giant/ailloy/commit/0765de5e337e59321d4f805b9be436fa08d867b2))
* cast --global should install to ~/ not ~/.ailloy/ ([c162c3a](https://github.com/nimble-giant/ailloy/commit/c162c3a53d87d8a4c144ab8158e54f4b03c908c2))
* clarify brainstorm command produces reports, not implementation ([b6fa8df](https://github.com/nimble-giant/ailloy/commit/b6fa8df6315ad4d54321903e007b9e1d188707aa))
* clarify mold directories are flexible, not fixed ([27e548d](https://github.com/nimble-giant/ailloy/commit/27e548d9be8213fc8175d2c0d21c01a210a3d40d))
* convert docs/README.md to valid GitHub Flavored Markdown ([ff2166a](https://github.com/nimble-giant/ailloy/commit/ff2166a230703b501a5ab2e9e7d201200b1ac64d))
* gh deprecated projects api ([b1611a8](https://github.com/nimble-giant/ailloy/commit/b1611a8d5e8926485bc070aad2d466f464a9f70f))
* improve error handling and assertions in plugin validator tests ([71f8c26](https://github.com/nimble-giant/ailloy/commit/71f8c26af611ee18abb50088a640842556ab566d))
* sanitize file paths in manifest loaders (G304) ([0512278](https://github.com/nimble-giant/ailloy/commit/0512278b0734dc66ad53f05793644bb03468e204))
* start-issue no longer gets confused with sub-issues ([9cb5416](https://github.com/nimble-giant/ailloy/commit/9cb5416c73594cc85194b8de61d47f6371344426))
* use dotted paths in smelt and add root mold.yaml ([a5d9b38](https://github.com/nimble-giant/ailloy/commit/a5d9b382885007848b5983b13a127deb513d9f9d))
* use explicit error discard for mergo.Merge calls ([ab8f93c](https://github.com/nimble-giant/ailloy/commit/ab8f93c1205181add9cb48e9b6ec8c382c9532b4))

## [0.1.13](https://github.com/nimble-giant/ailloy/compare/v0.1.12...v0.1.13) (2026-02-23)


### Features

* make ailloy tool-agnostic across AI coding tools ([e69b0e1](https://github.com/nimble-giant/ailloy/commit/e69b0e1a13e0228f3321959828c7abfc7850e356))

## [0.1.12](https://github.com/nimble-giant/ailloy/compare/v0.1.11...v0.1.12) (2026-02-23)


### Features

* add foundry, mold get, and ingot get/add commands ([04b05a7](https://github.com/nimble-giant/ailloy/commit/04b05a716b60d01866486e99a9f9a93bf58d079c)), closes [#47](https://github.com/nimble-giant/ailloy/issues/47)
* add recast and quench commands ([ae03379](https://github.com/nimble-giant/ailloy/commit/ae0337959e594ff11c839d5374a8c0b9b645dc96)), closes [#48](https://github.com/nimble-giant/ailloy/issues/48)

## [0.1.11](https://github.com/nimble-giant/ailloy/compare/v0.1.10...v0.1.11) (2026-02-22)


### Features

* scm-native foundry resolution and local cache for remote molds ([e2bc197](https://github.com/nimble-giant/ailloy/commit/e2bc1975eff4ff5ef9263cc26a45676563459101))

## [0.1.10](https://github.com/nimble-giant/ailloy/compare/v0.1.9...v0.1.10) (2026-02-21)


### Features

* add --global flag to cast for user-level installs ([c162c3a](https://github.com/nimble-giant/ailloy/commit/c162c3a53d87d8a4c144ab8158e54f4b03c908c2))
* refactor anneal as dynamic mold-agnostic configuration wizard ([193251a](https://github.com/nimble-giant/ailloy/commit/193251a474edcb06da3e5201d9b9aa7ff7f6897e))


### Bug Fixes

* cast --global should install to ~/ not ~/.ailloy/ ([c162c3a](https://github.com/nimble-giant/ailloy/commit/c162c3a53d87d8a4c144ab8158e54f4b03c908c2))

## [0.1.9](https://github.com/nimble-giant/ailloy/compare/v0.1.8...v0.1.9) (2026-02-21)


### Features

* add temper command for mold/ingot validation ([bc2ac37](https://github.com/nimble-giant/ailloy/commit/bc2ac370fe6f0d0f8ac7495bfcc8ad5e2c06804a)), closes [#43](https://github.com/nimble-giant/ailloy/issues/43)

## [0.1.8](https://github.com/nimble-giant/ailloy/compare/v0.1.7...v0.1.8) (2026-02-21)


### Features

* add forge command for dry-run template rendering ([#42](https://github.com/nimble-giant/ailloy/issues/42)) ([3ed442b](https://github.com/nimble-giant/ailloy/commit/3ed442bd7da4ee215236ce499132df2e87003123))
* add smelt command for mold packaging ([#45](https://github.com/nimble-giant/ailloy/issues/45)) ([78279de](https://github.com/nimble-giant/ailloy/commit/78279dea2a09acd00249c7694fd264ac84a63797))
* convention-based mold structure with output path mappings ([bc1a460](https://github.com/nimble-giant/ailloy/commit/bc1a4608c9f501102aa9caecffe77d8e484e8eda))
* define mold.yaml and ingot.yaml manifest formats ([#40](https://github.com/nimble-giant/ailloy/issues/40)) ([12d48c4](https://github.com/nimble-giant/ailloy/commit/12d48c4ef920f6891c1fd6e95620877d1eeee48f))
* helm-style flux.yaml and optional flux.schema.yaml ([c217c02](https://github.com/nimble-giant/ailloy/commit/c217c028f61c611a69b8d3b252a873ac5d9738fa))
* implement stuffbin-based binary output for smelt ([f834463](https://github.com/nimble-giant/ailloy/commit/f834463153b3af19ff211471e06f153c5bf99816))
* ingot template partials and flux schema validation ([#44](https://github.com/nimble-giant/ailloy/issues/44)) ([68bacff](https://github.com/nimble-giant/ailloy/commit/68bacff33fc8153d8ab7cd71835ee7ccf9083e85))
* surface command aliases in help text using lipgloss table ([8014da5](https://github.com/nimble-giant/ailloy/commit/8014da592e9537d973abf61cd1195507debaf0f5))


### Bug Fixes

* add gosec G104 nosec directives to mergo.Merge calls ([7bab98c](https://github.com/nimble-giant/ailloy/commit/7bab98cb2400bbaaf22d67c9935e995e05339482))
* add nosec annotations for GoSec G304 in manifest loaders ([032ccba](https://github.com/nimble-giant/ailloy/commit/032ccba241222d8231eb986eac76c87ae6e47086))
* address errcheck lint issue in smelt binary packaging ([a428687](https://github.com/nimble-giant/ailloy/commit/a428687731c8d18dfab3f6f8941d4e7598f0c829))
* address errcheck lint issues in smelt package ([f22be39](https://github.com/nimble-giant/ailloy/commit/f22be39561490cc4362e217390114a49b6fad9bf))
* sanitize file paths in manifest loaders (G304) ([0512278](https://github.com/nimble-giant/ailloy/commit/0512278b0734dc66ad53f05793644bb03468e204))
* use dotted paths in smelt and add root mold.yaml ([a5d9b38](https://github.com/nimble-giant/ailloy/commit/a5d9b382885007848b5983b13a127deb513d9f9d))
* use explicit error discard for mergo.Merge calls ([ab8f93c](https://github.com/nimble-giant/ailloy/commit/ab8f93c1205181add9cb48e9b6ec8c382c9532b4))

## [0.1.7](https://github.com/nimble-giant/ailloy/compare/v0.1.6...v0.1.7) (2026-02-18)


### Features

* allows ignoring templates ([a3cd65c](https://github.com/nimble-giant/ailloy/commit/a3cd65ca368a155bf701fe7f20ad365c6261a853))
* rebuild interactive ux ([ba66237](https://github.com/nimble-giant/ailloy/commit/ba66237af120a53dacfee4c28108407f5d21b0bc))
* upgrade template engine with conditionals and model rendering ([e5adcf0](https://github.com/nimble-giant/ailloy/commit/e5adcf06ea1d0102a3581510795eef209734d0c3)), closes [#26](https://github.com/nimble-giant/ailloy/issues/26)

## [0.1.6](https://github.com/nimble-giant/ailloy/compare/v0.1.5...v0.1.6) (2026-02-17)


### Bug Fixes

* gh deprecated projects api ([b1611a8](https://github.com/nimble-giant/ailloy/commit/b1611a8d5e8926485bc070aad2d466f464a9f70f))

## [0.1.5](https://github.com/nimble-giant/ailloy/compare/v0.1.4...v0.1.5) (2026-02-17)


### Features

* introduce semantic model layer for generations ([a9b8769](https://github.com/nimble-giant/ailloy/commit/a9b8769b8aad47ae69315dfe0071678471875997))

## [0.1.4](https://github.com/nimble-giant/ailloy/compare/v0.1.3...v0.1.4) (2026-02-16)


### Bug Fixes

* start-issue no longer gets confused with sub-issues ([9cb5416](https://github.com/nimble-giant/ailloy/commit/9cb5416c73594cc85194b8de61d47f6371344426))

## [0.1.3](https://github.com/nimble-giant/ailloy/compare/v0.1.2...v0.1.3) (2026-02-16)


### Features

* add Claude Code workflow template with --with-workflows flag ([5197c98](https://github.com/nimble-giant/ailloy/commit/5197c98cb19814f962a16d9b3e71bf5504debd98))
* claude code review agent workflow for github ([e93cc91](https://github.com/nimble-giant/ailloy/commit/e93cc91acb4f29aadd507f81e88dc86ab582293e))

## [0.1.2](https://github.com/nimble-giant/ailloy/compare/v0.1.1...v0.1.2) (2026-02-15)


### Features

* add brainstorm command for structured idea analysis ([5dd7aba](https://github.com/nimble-giant/ailloy/commit/5dd7aba1efed41e0a6cb4402ab8956825e2a5e02))
* add skill for creating new ailloy templates ([f797c87](https://github.com/nimble-giant/ailloy/commit/f797c87580b6a6d36cf8f3157545f6b8b811450a))
* brainstorm skill ([2100774](https://github.com/nimble-giant/ailloy/commit/2100774e188f525826ef511f2d9642c16a1e614c))
* implement brainstorm as embedded ailloy template ([d05d21f](https://github.com/nimble-giant/ailloy/commit/d05d21fad63bc565765181eb188f702fb77d6138))
* implement dynamic build information display in cli ([9f79c2f](https://github.com/nimble-giant/ailloy/commit/9f79c2f12ed9b2c288a0b6d3101b894ae362baf8))


### Bug Fixes

* clarify brainstorm command produces reports, not implementation ([b6fa8df](https://github.com/nimble-giant/ailloy/commit/b6fa8df6315ad4d54321903e007b9e1d188707aa))
* improve error handling and assertions in plugin validator tests ([71f8c26](https://github.com/nimble-giant/ailloy/commit/71f8c26af611ee18abb50088a640842556ab566d))

## [0.1.1](https://github.com/nimble-giant/ailloy/compare/v0.1.0...v0.1.1) (2026-02-14)


### Features

* adds ci, release, and security badges ([bcfc25a](https://github.com/nimble-giant/ailloy/commit/bcfc25a62574eb1457dc26c9f6c6a6e93d9d6fb0))

## 0.1.0 (2026-02-13)


### Features

* ship it 🚀 — ailloy v0.0.0 ([0550117](https://github.com/nimble-giant/ailloy/commit/055011784e26333bbc1e2e15baf6770929698c75))

## Changelog
