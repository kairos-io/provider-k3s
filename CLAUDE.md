# CLAUDE.md — provider-k3s

## What This Repo Is

This is the K3s cluster provider plugin for the Kairos immutable Linux OS. It implements the `clusterplugin.ClusterPlugin` interface from the Kairos SDK and produces `yip.YipConfig` stage definitions that configure K3s at boot time. It handles three node roles (init, controlplane, worker), proxy environments, airgap image imports, and cluster reset via uninstall scripts.

## Architecture

```
main.go                          — wires ClusterPlugin + HandleClusterReset, calls log.InitLogger
pkg/provider/provider.go         — all bootstrap logic: ClusterProvider, parseOptions, parseFiles, parseStages
pkg/provider/reset.go            — HandleClusterReset handler
pkg/constants/constants.go       — stage name strings and ProviderOptions key constants
pkg/log/log.go                   — logrus setup with lumberjack rotation and version field injection
pkg/types/types.go               — minimal shared types (MountPoint)
pkg/version/version.go           — single var Version string, set via -ldflags at build time
api/k3s_config.go                — K3sServerConfig struct + StringListKeys var
api/agent_config.go              — K3sAgentConfig struct
```

The binary is named `agent-provider-k3s`. It lives at `/system/providers/agent-provider-k3s` in the image.

## Provider Implementation Pattern

`ClusterProvider` is a pure function: `func ClusterProvider(cluster clusterplugin.Cluster) yip.YipConfig`. It builds a `yip.YipConfig` with a single `"boot.before"` stage list and returns it. No side effects, no I/O, no globals mutated.

`HandleClusterReset` is an event handler: `func HandleClusterReset(event *pluggable.Event) pluggable.EventResponse`. It runs the appropriate uninstall shell script via `exec.Command` and surfaces any error in `response.Error`.

Both are registered in `main.go` via `clusterplugin.ClusterPlugin` — the provider function is assigned to `Plugin.Provider`, the reset handler to a `pluggable.FactoryPlugin`.

## K3s-Specific Patterns

**Role dispatch is a switch statement, not a map or interface dispatch.** `parseOptions` uses a `switch cluster.Role` with cases for `RoleInit`, `RoleControlPlane`, `RoleWorker`. Init sets `ClusterInit: true`, ControlPlane sets `Server:` to the control plane URL, Worker sets `Server:` and unmarshals into `K3sAgentConfig` instead of `K3sServerConfig`.

**Config is written to `/etc/rancher/k3s/config.d/` as numbered YAML files, then merged with `jq`.** The merge command is a raw inline shell string passed as a `yip.Stage` command. Files are:
- `90_userdata.yaml` — user-supplied options filtered through the appropriate config struct
- `99_userdata.yaml` — provider-derived options (token, server URL, tls-san, cluster-init)

The final merge writes to `/etc/rancher/k3s/config.yaml`. File permissions are `0400`.

**`ProviderOptions` is a `map[string]string`.** The key `"cluster_root_path"` is accessed via `cluster.ProviderOptions[constants.ClusterRootPath]` and used to prefix script paths. The special case `cluster-init: "no"` requires manual JSON byte-level patching because Go marshals `omitempty bool` as absent when false.

**StringListKeys** is a package-level `var` in `api` that lists K3s config keys which should be treated as `[]string` even when provided as a single comma-separated string. `decodeOptions` iterates the user YAML map and calls `decodeOption` per key. If a value is a comma-separated string, it is split into `[]string`. If the key is in `StringListKeys`, a single-element string is promoted to `[]string`.

**System service names** are local constants in `provider.go`: `serverSystemName = "k3s"` and `agentSystemName = "k3s-agent"`. Both OpenRC and systemd are handled as separate `yip.Stage` entries, each guarded by an `If:` shell conditional (`[ -x /sbin/openrc-run ]` and `[ -x /bin/systemctl ]`).

**Swap disable** is always prepended as the first stage via `getSwapDisableStage()`. It runs `sed` on `/etc/fstab` and `swapoff -a`.

**Airgap image import** is conditional on `cluster.ImportLocalImages`. When true, two commands are appended: `chmod +x` the import script, then execute it. The script copies `.tar` files into `/var/lib/rancher/k3s/agent/images/`.

## Config Struct Design

`K3sServerConfig` and `K3sAgentConfig` are large flat structs in the `api` package. Every field has both `json` and `yaml` tags with `omitempty`. Field names use Go CamelCase, struct tags use the exact K3s CLI flag names with hyphens. Types match K3s semantics: flags that accept multiple values are `[]string`, single-value flags are `string`, booleans are `bool`, numeric flags are `int` or `time.Duration`.

There is no embedding, no interface, no generics. The two structs are separate and independently maintained. Config filtering is done by unmarshalling into the appropriate struct then re-marshalling — fields not present in the struct are silently dropped. This is intentional; it is how server-only and agent-only options are separated.

## Error Handling Rules

**In `ClusterProvider` and its callees, `logrus.Fatalf` is used for unrecoverable parse/marshal errors.** There is no error return from `ClusterProvider`. If YAML unmarshalling of `cluster.Options` fails, the process exits. This is deliberate — a misconfigured cluster should not boot into an unknown state.

**In `HandleClusterReset`, errors are returned in `response.Error` as a formatted string.** No `logrus` in the reset path. No panics. The pattern is: declare `var response pluggable.EventResponse`, populate `response.Error` on failure, return `response` at every exit point.

**Errors from `json.Marshal` / `kyaml.YAMLToJSON` after the initial unmarshal are ignored with `_`.** This reflects the author's intent: by the time you get to the JSON conversion, the data has already been validated by struct unmarshalling.

**The `getDefaultNoProxy` function uses a direct type assertion `data["cluster-cidr"].(string)` without the ok check.** Do not add ok-checked assertions here — the existing code accepts a panic if the field type is wrong, which is treated as a programmer error.

## Code Style Rules

**Functions are short and single-purpose.** `parseOptions`, `parseFiles`, `parseStages`, `proxyEnv`, `getDefaultNoProxy`, `getNodeCIDR`, `getClusterRootPath`, `decodeOptions`, `decodeOption`, `getSwapDisableStage` are all separate functions. No function does more than one logical job.

**No named return values.** Functions return values are positional. Multiple return values are used for `(options, proxyOptions, userOptions)` in `parseOptions` — three `[]byte` values returned in a defined order with no names.

**Variable declarations follow Go idiom**: `var stages []yip.Stage` for slices that will be appended to; `:=` for everything else. No `new()`. No pointer receivers on config structs.

**Import grouping**: stdlib first, then third-party, then internal. The `_ "embed"` blank import appears in the stdlib group. `yip` and `kyaml` are aliased: `yip "github.com/mudler/yip/pkg/schema"` and `kyaml "sigs.k8s.io/yaml"`. `yaml` (gopkg.in/yaml.v3) is unaliased.

**Constants are grouped by purpose in separate `const (...)` blocks.** Stage name strings live in `pkg/constants/constants.go` alongside `ProviderOptions` key constants. Local-only constants (paths, system names, proxy constants) are defined as unexported `const` blocks at the top of `provider.go`.

**`fmt.Sprintf` is used for all string interpolation** — no string concatenation with `+` except in `noProxy` accumulation in `proxyEnv` and `getDefaultNoProxy`, which follows the existing `noProxy = noProxy + "," + ...` pattern.

**No error wrapping with `fmt.Errorf("%w", ...)`.** Errors passed to `response.Error` use `fmt.Sprintf("...: %s", err.Error())`. This is consistent across the codebase.

**The jq merge command is written as a single inline string literal** in a `Commands: []string{...}` field. Do not break it into a multiline raw string or a variable. Keep it inline.

## Testing Conventions

Tests live in `pkg/provider/provider_test.go`, same package (`package provider`). They test unexported functions directly.

**Table-driven tests using `[]struct{...}` with a `name string` field.** The struct is defined inline inside the test function. No shared test fixtures, no test helpers, no `testing.T` helpers beyond `t.Run`, `t.Errorf`, and `t.Logf`.

**Test names use natural English phrasing with colons for sub-cases**: `"Init: Standard"`, `"Init: 2-Node"`, `"Control Plane: With Options"`. Not snake_case, not `TestFoo_bar`.

**Assertions use `bytes.Equal` for `[]byte` comparison** and `reflect.DeepEqual` for map comparison. No testify assertions. No gomega. No mock frameworks.

**The test for `parseOptions` specifies exact JSON byte output.** When adding test cases, the `expectedOptions` must match the exact JSON serialisation order produced by `json.Marshal` on `K3sServerConfig` (alphabetical by field tag name).

**Exploratory/diagnostic tests** (like `Test_unmarshall`) are kept in the file alongside functional tests. They use `t.Logf` to print results, not assertions. Do not delete them.

## Patterns to Avoid

- Do not add middleware, decorators, or functional options patterns. Config is built directly.
- Do not introduce interfaces for `ClusterProvider` or the config structs. They are concrete types.
- Do not add error wrapping or sentinel errors. Errors are either fatal (logrus.Fatalf) or string-encoded into EventResponse.
- Do not use `context.Context` anywhere in the provider logic. This codebase has no context passing.
- Do not split `provider.go` into multiple files unless a new top-level feature (equivalent in scope to reset.go) is added.
- Do not add a `New()` constructor for config structs. Struct literals are used directly.
- Do not use `os.Exit` directly. `logrus.Fatal` is used at the top level; the rest of the code does not call os.Exit.
- Do not add validation functions that return errors for config fields. Invalid config is caught by YAML unmarshal failures.
- Do not add config structs to the `provider` package. Config types belong in `api/`.
- Do not use `map[string]string` for stage commands — always `[]string`.
- Do not use `strings.Builder` for building proxy env strings. The existing pattern uses `append` to a `[]string` and `strings.Join`.
- Do not add logging inside `decodeOptions` or `decodeOption`. These are pure transform functions.
- Do not add a separate `pkg/config/` package. Config generation stays in `pkg/provider/provider.go`.

## Function Design & Testability

- **Every function does one thing and fits in ~20–30 lines.** If it grows beyond that, extract named helpers.
- **Write functions so they can be unit tested in isolation** — no hidden side effects, no global state access, no I/O buried inside business logic.
- **Most business logic must be unit testable** without spinning up a server, database, or Kubernetes cluster. Separate I/O at the boundary.
- **Use guard clauses / early returns** to reduce nesting. Flat code is easier to read and test than deeply nested.
- **Accept interfaces, return concrete types.** This makes callers mockable without reflection or code generation.
- **Keep interfaces small** — 1–3 methods. Large interfaces are hard to mock and signal poor separation of concerns.

## General Go Practices

- **Dependency injection over globals.** Pass dependencies via constructors or function parameters — not package-level singletons (except logging).
- **`context.Context` is always the first parameter** on any function that performs I/O. Never store it in a struct field.
- **Table-driven tests** for any function with multiple input/output cases: `[]struct{ name, input, expected }` with `t.Run`.
- **Test naming:** `TestFuncName_Scenario` — e.g. `TestCreateCluster_MissingName`.
- **Prefer `switch` over long `if/else if` chains.**
- **Short variable names in small scopes** (`i`, `v`, `err`) are idiomatic; use descriptive names in wider scopes.
- **No goroutines unless concurrency is genuinely required.** Sequential code is easier to test and reason about.
- **Avoid `init()` for anything except registering handlers or loggers.** Never use it for config loading or side-effectful setup.
- **Respect context cancellation** in any loop that calls external services.
- **Import grouping:** stdlib / external / internal — separated by blank lines, sorted by `goimports`.
- **Don't over-abstract.** Don't create an interface or wrapper until there are ≥2 concrete implementations or a clear testing need.
- **No naked `panic` in library code.** Panics are only acceptable in `main` or test setup for truly unrecoverable state.
