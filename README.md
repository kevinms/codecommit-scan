# codecommit-scan

This utility provides a few quality-of-life commands for inspecting CodeCommit
repositories and pull requests.

A few things it can do:

* List all open PRs waiting for your approval.

```sh
codecommit-scan [-region <region>]
```

* List all open PRs authored by you.

```sh
codecommit-scan [-region <region>] -mine
```

## Install

If you have `go` tools installed, you can fetch and build the utility with:

```sh
go install github.com/kevinms/codecommit-scan@latest
```

## Configure

The AWS shared configuration is automatically loaded, e.g. `~/.aws/config` on
Linux.
