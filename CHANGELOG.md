## 1.0.0 (2025-11-20)


### ⚠ BREAKING CHANGES

* Removed add-site.sh, use add-cert.sh for certificates only

### Features

* add Aura CLI for global command access ([dd7289d](https://github.com/ivannovak/localaura/commit/dd7289d52c5b9e9c4b9cd404ad2bf324682eb2af))
* add semantic-release for automated versioning ([19d7909](https://github.com/ivannovak/localaura/commit/19d7909e2c559a29c0b32fe81e3aedbb4f922c3c))
* initial Aura proxy setup with Docker labels and mkcert integration ([1c74815](https://github.com/ivannovak/localaura/commit/1c748158652b3055902c498848fd20c0945a22c0))
* replace /etc/hosts with CoreDNS wildcard DNS resolution ([3463ce4](https://github.com/ivannovak/localaura/commit/3463ce4d44391e826ce126776167cb399b9a4aaf))


### Bug Fixes

* add package-lock.json and migrate to semantic-release-replace-plugin ([5b3c9f9](https://github.com/ivannovak/localaura/commit/5b3c9f96c0a8814d417368ca219b60d2625da631))
* remove deprecated exportloopref linter and restructure semantic-release workflow ([a591416](https://github.com/ivannovak/localaura/commit/a5914169bb29429804903706fe43067455372ffa))
* remove results validation from semantic-release replace config ([247dd70](https://github.com/ivannovak/localaura/commit/247dd70fc852995b9bffd24df53d9478443f7c7b))
* resolve linting errors and CI workflow issues ([fbd94c6](https://github.com/ivannovak/localaura/commit/fbd94c65959945edde19183d56e5410f7a4356ee))
* update semantic-release checkout to use correct commit SHA ([aa65b60](https://github.com/ivannovak/localaura/commit/aa65b60f379ef8b2a86d3d2ab4b65f8bc6bf22a1))


### Code Refactoring

* simplify to Docker labels-only approach ([d34d101](https://github.com/ivannovak/localaura/commit/d34d1013a7fa4fada596ec8fb88723363d93cc5e))

# Changelog

All notable changes to this project will be documented in this file.

This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html) and uses [Conventional Commits](https://www.conventionalcommits.org/).

Releases are automated using [semantic-release](https://semantic-release.gitbook.io/).
