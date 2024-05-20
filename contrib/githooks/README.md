# Git hooks

We have pre-commit and pre-push.

Pre-commit using <https://pre-commit.com/>

## Installation

### Pre-commit hooks

Using pip:

`pip install pre-commit`

Or Using homebrew:

`brew install pre-commit`

### Pre-push hooks

Install golangci-lint:

`go install github.com/golangci/golangci-lint/cmd/golangci-lint`

Install markdownlint-cli:

`npm install -g markdownlint-cli`

## Installation of githook scripts

From root directory:

`pre-commit install -c ./contrib/githooks/pre-commit-config.yaml`

`./scripts/setup-git-hooks.sh`
