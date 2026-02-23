# Changelog

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
