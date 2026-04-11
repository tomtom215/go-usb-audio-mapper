<!-- SPDX-License-Identifier: MIT -->
<!-- Copyright 2025 Tom F. (https://github.com/tomtom215) -->

# Project Governance

## Roles

### Contributor

Anyone who submits a pull request, files an issue, or participates in discussions. No special permissions required.

### Committer

Trusted contributors granted write access to the repository. Committers can merge PRs and manage issues. Selected through nomination based on sustained quality contributions.

### Maintainer

Maintainers set the project's technical direction, approve architectural changes, manage releases, and administer repository settings.

**Initial Maintainer:** Tom F. ([@tomtom215](https://github.com/tomtom215))

## Decision-Making

This project uses **lazy consensus**. Proposals (issues, PRs, discussions) are considered accepted if no objections are raised within **7 calendar days**. Minor updates (typos, documentation fixes, dependency bumps) may skip this waiting period.

## Escalation

When consensus cannot be reached:

1. The maintainer facilitates documented discussion
2. A decision deadline is set (minimum 7 days)
3. If needed, a majority vote among committers is held
4. The maintainer breaks ties

## Code of Conduct

All participants are expected to behave respectfully and professionally. Violations may be reported to the maintainer via GitHub.

## Releases

Releases follow [Semantic Versioning 2.0.0](https://semver.org/). The `CHANGELOG.md` is updated in [Keep a Changelog](https://keepachangelog.com/) format for every release. See [RELEASING.md](RELEASING.md) for the release process.
