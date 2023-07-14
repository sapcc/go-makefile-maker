# go-makefile-maker

[![CI](https://github.com/sapcc/go-makefile-maker/actions/workflows/ci.yaml/badge.svg)](https://github.com/sapcc/go-makefile-maker/actions/workflows/ci.yaml)

Generates a Makefile and optionally also GitHub workflows and a Dockerfile for your Go application:

* Makefile follows established Unix conventions for installing and packaging,
  and includes targets for vendoring, running tests and checking code quality.
* GitHub Workflows use [GitHub Actions](https://github.com/features/actions) to lint,
  build, and test your code. Additionally, you can enable workflows for checking your
  codebase for security issues (e.g. [CodeQL] code scanning), spelling errors, and missing
  license headers.

## Installation

The easiest way to get `go-makefile-maker` is: `go install github.com/sapcc/go-makefile-maker@latest`.

We also support the usual Makefile invocations: `make`, `make check`, and `make install`. The latter understands the conventional environment variables for choosing install locations: `DESTDIR` and `PREFIX`.

You usually want something like `make && sudo make install PREFIX=/usr/local`.

## Usage

Put a `Makefile.maker.yaml` file in your Git repository's root directory, then run the following to generate Makefile and GitHub workflows:

```sh
$ go-makefile-maker
```

`go-makefile-maker` also generates a `help` target for usage info:

```sh
$ make help
```

In addition to the `Makefile`, you should also commit the `Makefile.maker.yaml` file so that your users don't need to have `go-makefile-maker` installed.

## Configuration

`go-makefile-maker` requires a config file (`Makefile.maker.yaml`) in the [YAML format][yaml].

Take a look at `go-makefile-maker`'s [own config file](./Makefile.maker.yaml) for an example of what a config could like.

The config file has the following sections:

* [metadata](#metadata)
* [binaries](#binaries)
* [testPackages](#testpackages)
* [coverageTest](#coveragetest)
* [dockerfile](#dockerfile)
* [variables](#variables)
* [vendoring](#vendoring)
* [golangciLint](#golangcilint)
* [spellCheck](#spellcheck)
* [renovate](#renovate)
* [verbatim](#verbatim)
* [githubWorkflow](#githubworkflow)
  * [githubWorkflow\.global](#githubworkflowglobal)
  * [githubWorkflow\.ci](#githubworkflowci)
  * [githubWorkflow\.pushContainerToGhcr](#githubworkflowpushcontainertoghcr)
  * [githubWorkflow\.securityChecks](#githubworkflowsecuritychecks)
  * [githubWorkflow\.license](#githubworkflowlicense)
  * [githubWorkflow\.spellCheck](#githubworkflowspellcheck)

### `metadata`

```yaml
metadata:
  url: https://github.com/foo/bar
```

`metadata` contains information about the project which cannot be guessed consistently:

- `url` is the repository's remote URL.

### `binaries`

```yaml
binaries:
  - name: example
    fromPackage: ./cmd/example
    installTo: bin/
  - name: test-helper
    fromPackage: ./cmd/test-helper
```

For each binary specified here, a target will be generated that builds it with `go build` and puts it in `build/$NAME`.
The `fromPackage` is a Go module path relative to the directory containing the Makefile.

If `installTo` is set for at least one binary, the `install` target is added to the Makefile, and all binaries with `installTo` are installed by it.
In this case, `example` would be installed as `/usr/bin/example` by default, and `test-helper` would not be installed.

### `testPackages`

```yaml
testPackages:
  only: '/internal'
  except: '/test/util|/test/mock'
```

By default, all packages inside the repository are subject to testing, but this section can be used to restrict this.

The values in `only` and `except` are regexes for `grep -E`.
Since only entire packages (not single source files) can be selected for testing, the regexes have to match package names, not on file names.

### `coverageTest`

```yaml
coverageTest:
  only: '/internal'
  except: '/test/util|/test/mock'
```

When `make check` runs `go test`, it produces a test coverage report.
By default, all packages inside the repository are subject to coverage testing, but this section can be used to restrict this.

The values in `only` and `except` are regexes for `grep -E`.
Since only entire packages (not single source files) can be selected for coverage testing, the regexes have to match package names, not on file names.

### `dockerfile`

```yaml
dockerfile:
  enabled: true
  entrypoint: [ "/bin/bash", "--", "--arg" ]
  extraDirectives:
    - 'LABEL mylabel=myvalu'
  extraIgnores:
    - tmp
    - files
  extraPackages:
    - curl
    - openssl
  user: root
```

When `enabled`, go-makefile-maker will generate a `Dockerfile` and a `.dockerignore` file.
The Dockerfile uses the [Golang base image](https://hub.docker.com/_/golang) to run `make install`, then copies all installed files into a fresh [Alpine base image](https://hub.docker.com/_/alpine).
The image is provisioned with a dedicated user account (name `appuser`, UID 4200, home directory `/home/appuser`) and user group (name `appgroup`, GID 4200) with stable names and IDs.
This user account is intended for use with all payloads that do not require a root user.

* `entrypoint` allows overwriting the final entrypoint.
* `extraDirectives` appends additional directives near the end of the Dockerfile.
* `extraIgnores` appends entries in `.dockerignore` to the default ones.
* `extraPackages` installs extra Alpine packages in the final Docker layer. `ca-certificates` is always installed.
* `runAsRoot` skips the privilege drop in the Dockerfile, i.e. the `USER appuser:appgroup` command is not added.

### `variables`

```yaml
variables:
  GO_BUILDFLAGS: '-mod vendor'
  GO_LDFLAGS: ''
  GO_TESTENV: ''
```

Allows to override the default values of Makefile variables used by the autogenerated recipes.
This mechanism cannot be used to define new variables to use in your own rules; use `verbatim` for that.
By default, all accepted variables are empty.
The only exception is that `GO_BUILDFLAGS` defaults to `-mod vendor` when vendoring is enabled (see below).

A typical usage of `GO_LDFLAGS` is to give compile-time values to the Go compiler with the `-X` linker flag:

```yaml
variables:
  GO_LDFLAGS: '-X github.com/foo/bar.Version = $(shell git describe --abbrev=7)'
```

However, for this specific usecase, we suggest that your application use `github.com/sapcc/go-api-declarations/bininfo`
instead. When the respective module is present as a direct dependency in the `go.mod` file, go-makefile-maker will
auto-generate suitable linker flags to fill the global variables in the `bininfo` package.

`GO_TESTENV` can contain environment variables to pass to `go test`:

```yaml
variables:
  GO_TESTENV: 'POSTGRES_HOST=localhost POSTGRES_DATABASE=unittestdb'
```

### `vendoring`

```yaml
vendoring:
  enabled: false
```

Set `vendoring.enabled` to `true` if you vendor all dependencies in your repository. With vendoring enabled:

1. The default for `GO_BUILDFLAGS` is set to `-mod vendor`, so that build targets default to using vendored dependencies.
   This means that building binaries does not require a network connection.
2. The `make tidy-deps` target is replaced by a `make vendor` target that runs `go mod tidy && go mod verify` just like `make tidy-deps`, but also runs `go
   mod vendor`.
   This target can be used to get the vendor directory up-to-date before commits.

### `golangciLint`

```yaml
golangciLint:
  createConfig: false
  errcheckExcludes:
    - io/ioutil.ReadFile
    - io.Copy(*bytes.Buffer)
    - io.Copy(os.Stdout)
    - (*net/http.Client).Do
  skipDirs:
    - easypg/migrate/*
```

The `make check` and `make static-check` targets use [`golangci-lint`](https://golangci-lint.run) to lint your code.

If `createConfig` is set to `true` then `go-makefile-maker` will create a
config file (`.golangci.yaml`) for `golangci-lint` and keep it up-to-date (in case of new
changes). This config file enables extra linters in addition to the default ones and
configures various settings that can improve code quality.

Additionally, if `createConfig` is `true`, you can specify a list of files skipped entirely by golangci-lint in `skipDirs`
and a list of functions to be excluded from `errcheck` linter in `errcheckExcludes` field.
Refer to [`errcheck`'s README](https://github.com/kisielk/errcheck#excluding-functions) for info on the format
for function signatures that `errcheck` accepts.

Take a look at `go-makefile-maker`'s own [`golangci-lint` config file](./.golangci.yaml) for an up-to-date example of what the generated config would look like.

### `spellCheck`

```yaml
spellCheck:
  ignoreWords:
    - example
    - exampleTwo
```

`golangci-lint` (if `golangciLint.createConfig` is `true`) and the spell check GitHub workflow (`githubWorkflow.spellCheck`) use [`misspell`][misspell] to check for spelling errors.

If `spellCheck.ignoreWords` is defined then both `golangci-lint` and spell check workflow will give this word list to `misspell` so that they can be ignored during its checks.

### `renovate`

```yaml
renovate:
  enabled: true
  assignees:
    - devnull
    - urandom
  goVersion: 1.18
  packageRules:
    - matchPackageNames: []
      matchPackagePrefixes: []
      matchUpdateTypes: []
      matchDepTypes: []
      matchFiles: []
      allowedVersions: ""
      autoMerge: false
      enabled: false
```

Generate [RenovateBot](https://renovatebot.com/) config to automatically create pull requests weekly on Fridays with dependency updates.

To assign people to the PRs created by renovate, add their GitHub handle to the `assignees` list.

Optionally overwrite go version with `goVersion`, by default the Go version from `go.mod` file will be used.

Additionally, you can also define [`packageRules`](https://docs.renovatebot.com/configuration-options/#packagerules). Note that only the fields mentioned above are accepted when defining a `packageRule`. The following package rules are defined by default:

```yaml
packageRules:
  - matchPackagePatterns: [".*"]
    groupName: "all" # group all PRs together
  - matchPackageNames: ["golang"]
    allowedVersions: $goVersion.x
  - matchDepTypes: ["action"]
    enabled: false # because github-actions will be updated by go-makefile-maker itself, see githubWorkflow config section below.
  - matchDepTypes: ["dockerfile"]
    enabled: false # because dockerfile will be updated by go-makefile-maker itself, see docker config section above.
  # This package rule will be added if go.mod file has a `k8s.io/*` dependency.
  - matchPackagePrefixes: ["k8s.io/"]
    allowedVersions: 0.25.x
  # This package rule will be added along with the required matchPackagePrefixes if go.mod file has the respective dependencies.
  - matchPackagePrefixes:
      - github.com/sapcc/go-api-declarations
      - github.com/sapcc/gophercloud-sapcc
      - github.com/sapcc/go-bits
    autoMerge: true
```

### `verbatim`

```yaml
verbatim: |
  run-example: build/example
    ./build/example example-config.txt
```

This field can be used to add your own definitions and rules to the Makefile.
The text in this field is copied into the Makefile mostly verbatim, with one exception:
Since YAML does not like tabs for indentation, we allow rule recipes to be indented with spaces.
This indentation will be replaced with tabs before writing it into the actual Makefile.

### `githubWorkflow`

The `githubWorkflow` section holds configuration options that define the behavior of various GitHub workflows.

**Hint**: You can prevent the workflows from running by including `[ci skip]` in your commit message
([more info](https://github.blog/changelog/2021-02-08-github-actions-skip-pull-request-and-push-workflows-with-skip-ci/)).

#### `githubWorkflow.global`

This section defines global settings that apply to all workflows. If the same setting is
supported by a specific workflow and is defined then that will take override its global
value.

```yaml
global:
  defaultBranch: dev
  goVersion: 1.18
```

`defaultBranch` specifies the Git branch on which `push` actions will trigger the
workflows. This does not affect pull requests, they will automatically trigger all
workflows regardless of which branch they are working against. `go-makefile-maker` will
automatically run `git symbolic-ref refs/remotes/origin/HEAD | sed
's@^refs/remotes/origin/@@'` and use its value by default.

`goVersion` specifies the Go version that is used for jobs that require Go.
`go-makefile-maker` will automatically retrieve the Go version from `go.mod` file and use
that by default.

#### `githubWorkflow.ci`

This workflow:

* checks your code using `golangci-lint`
* ensures that your code compiles successfully
* runs tests and generates test coverage report
* uploads the test coverage report to [Coveralls]

```yaml
ci:
  enabled: true
  runOn:
    - macos-latest
    - ubuntu-latest
    - windows-latest
  coveralls: true
  postgres:
    enabled: true
    version: 12
  kubernetesEnvtest:
    enabled: true
    version: 1.25.x!
  ignorePaths: []
```

`runOn` specifies a list of machine(s) to run the `build` and `test` jobs on ([more
info][ref-runs-on]). You can use this to ensure that your build compilation and tests are
successful on multiple operating systems. Default value for this is `ubuntu-latest`.

If `coveralls` is `true` then your test coverage report will be uploaded to [Coveralls]. Make sure that you have enabled Coveralls for your GitHub repo beforehand.

If `postgres.enabled` is `true` then a PostgreSQL service container will be added for the
`test` job. You can connect to this PostgreSQL service at `localhost:54321` with
`postgres` as username and password ([More info][postgres-service-container]).
`postgres.version` specifies the Docker Hub image tag for the [`postgres`
image][docker-hub-postgres] that is used for this container. By default `12` is used as
image tag.

If `kubernetesEnvtest.enabled` is `true` then
[Envtest](https://book.kubebuilder.io/reference/envtest.html) binaries will be downloaded
using
[`setup-envtest`](https://pkg.go.dev/sigs.k8s.io/controller-runtime/tools/setup-envtest)
for the `test` job. The version for binaries can be specified using the
`kubernetesEnvtest.version` field. By default `1.25.x!` is used as the version.

`ignorePaths` specifies a list of filename patterns. Workflows will not trigger if a path
name matches a pattern in this list. [More info][ref-onpushpull] and [filter pattern cheat
sheet][ref-pattern-cheat-sheet]. This option is not defined by default.

### `githubWorkflow.pushContainerToGhcr`

If `enabled` is set to true, the generated `Dockerfile` is build and pushed to repository path under `ghcr.io`.

```yaml
pushContainerToGhcr:
  enabled: true
```

#### `githubWorkflow.securityChecks`

If `securityChecks` is enabled then it will generate the following workflows:

* [CodeQL] workflow will run [CodeQL], GitHub's industry-leading semantic code analysis
  engine, on your source code to find security vulnerabilities. You can see the security
  report generated by CodeQL under your repo's Security tab.

  In addition to running the workflow when new code is pushed, this workflow will also run
  on a weekly basis (every Monday at 07:00 AM) so that existing code can be checked for
  new vulnerabilities.

* [dependency-review] workflow will scan your pull requests for dependency changes and
  will raise an error if any new dependencies have existing vulnerabilities.
  It uses the [GitHub Advisory Database](https://github.com/advisories) as a source.

* [govulncheck] workflow will scan your dependencies for vulnerarbilites and
  will raise an error if any dependency has an existing vulnerability and the code path is in use.
  It uses the [Go Vulnerability Database](https://pkg.go.dev/vuln/) as a source.

```yaml
securityChecks:
  enabled: true
```

#### `githubWorkflow.license`

This workflow ensures that all your source code files have a license header. It
uses [`addlicense`][addlicense] for this.

```yaml
license:
  enabled: true
  patterns:
    - "**/*.go"
  ignorePatterns:
    - "vendor/**"
```

`patterns` specifies a list of file patterns to check. For convenience, the `globstar`
option is enabled for the workflow's shell session therefore you can use `**` in your file
patterns. Additionally, `addlicense` will scan directory patterns recursively. See
`addlicense`'s [README][addlicense] for more info. Default value for this is `**/*.go`,
i.e. check all Go files.

`ignorePatterns` specifies a list of file patterns to check. You can use any pattern
[supported by doublestar][doublestar-pattern]. See `addlicense`'s [README][addlicense] for
more info. Default value for this is `vendor/**`, i.e. exclude everything under `vendor`
directory.

**Hint**: you can also use `addlicense` to add license headers to all Go files excluding
`vendor` directory by running `make license-headers`.

#### `githubWorkflow.spellCheck`

This workflow uses [`misspell`][misspell] to check your repo for spelling errors. Unlike
`golangci-lint` that only runs `misspell` on `.go` files, this workflow will run
`misspell` on your entire repo.

```yaml
spellCheck:
  enabled: true
```

[codeql]: https://codeql.github.com/
[coveralls]: https://coveralls.io
[docker-hub-postgres]: https://hub.docker.com/_/postgres/
[doublestar-pattern]: https://github.com/bmatcuk/doublestar#patterns
[govulncheck]: https://github.com/golang/vuln
[misspell]: https://github.com/client9/misspell
[postgres-service-container]: https://docs.github.com/en/actions/guides/creating-postgresql-service-containers#testing-the-postgresql-service-container
[ref-onpushpull]: https://docs.github.com/en/actions/reference/workflow-syntax-for-github-actions#onpushpull_requestpaths
[ref-pattern-cheat-sheet]: https://docs.github.com/en/actions/reference/workflow-syntax-for-github-actions#filter-pattern-cheat-sheet
[ref-runs-on]: https://docs.github.com/en/actions/reference/workflow-syntax-for-github-actions#jobsjob_idruns-on
[yaml]: https://yaml.org/
[addlicense]: https://github.com/google/addlicense
[dependency-review]: https://github.com/actions/dependency-review-action
