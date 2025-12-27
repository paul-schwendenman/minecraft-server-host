# Comprehensive Test Plan

This document outlines the complete testing strategy for the Minecraft Server Host project. Use this as a reference when creating detailed implementation plans for each component.

---

## Table of Contents

1. [Current State Summary](#current-state-summary)
2. [UI Apps & Libraries](#ui-apps--libraries)
3. [Go CLI (minecraftctl)](#go-cli-minecraftctl)
4. [Lambda Functions](#lambda-functions)
5. [Packer Scripts](#packer-scripts)
6. [CI/CD Integration](#cicd-integration)
7. [Priority Matrix](#priority-matrix)
8. [Success Metrics](#success-metrics)
9. [Implementation Progress](#implementation-progress)
10. [Next Steps](#next-steps)

---

## Current State Summary

| Component | Test Framework | Test Files | Tests | Coverage | CI Integration |
|-----------|---------------|------------|-------|----------|----------------|
| UI - Manager App | Vitest + jsdom | 1 | 2 | Minimal | ✅ `pnpm -r test` |
| UI - Worlds App | Vitest + Playwright | 1 | 2 | Minimal | ✅ `pnpm -r test` |
| UI - @minecraft/ui | Vitest + Playwright | 4 | 20 | Good | ✅ `pnpm -r test` |
| UI - @minecraft/data | Vitest | 1 | 21 | Good | ✅ `pnpm -r test` |
| Go CLI | Go testing | 13 | 282 | 37% | ✅ Coverage reporting |
| Lambda - control | pytest + moto | 3 | 19 | Good | ✅ pytest in CI |
| Lambda - details | pytest | 2 | 13 | Good | ✅ pytest in CI |
| Lambda - worlds | pytest + moto | 2 | 15 | Good | ✅ pytest in CI |
| Packer scripts | bats + shellcheck | 4 | 53 | Good | ✅ bats + shellcheck |

**Total Tests: 427** (UI: 45, Go: 282, Lambda: 47, Packer: 53)

---

## UI Apps & Libraries

### Existing Infrastructure

- **Test Runner**: Vitest 3.2.4
- **Component Testing**: @testing-library/svelte
- **Browser Testing**: @vitest/browser + Playwright
- **DOM Matchers**: @testing-library/jest-dom
- **Environment**: jsdom (manager) / Playwright browser (libs)

### Package: @minecraft/ui (`libs/ui/`)

**Components to Test:**

| Component | File | Priority | Test Type | Notes |
|-----------|------|----------|-----------|-------|
| AsyncButton | `AsyncButton.svelte` | High | Unit | Loading states, click handling, disabled states |
| ServerStatus | `ServerStatus.svelte` | High | Unit | Status display, button actions, state transitions |
| ServerDetails | `ServerDetails.svelte` | Medium | Unit | Data rendering, prop handling |
| ActivePlayerList | `ActivePlayerList.svelte` | Medium | Unit | List rendering, empty states |
| ActivePlayerMessage | `ActivePlayerMessage.svelte` | Low | Unit | Individual item rendering |
| ServerVersion | `ServerVersion.svelte` | Low | Unit | Version display |
| Spinner | `Spinner.svelte` | Low | Unit | Animation, visibility |

**Existing Tests (all skipped):**
- `ServerStatus.svelte.test.js` - 128 lines, needs store mocking fixes
- `ActivePlayerList.svelte.test.js` - 36 lines
- `ServerDetails.svelte.test.js` - Skipped
- `ActivePlayerMessage.svelte.test.js` - Skipped

**Test Plan:**
1. Fix store mocking pattern for Svelte 5 compatibility
2. Unskip and update ServerStatus tests
3. Unskip and update remaining component tests
4. Add missing component tests (AsyncButton, Spinner, ServerVersion)
5. Add snapshot tests for visual regression

### Package: @minecraft/data (`libs/data/`)

**Modules to Test:**

| Module | File | Priority | Test Type | Notes |
|--------|------|----------|-----------|-------|
| status API | `api/status.ts` | High | Unit | Mock fetch, response parsing |
| details API | `api/details.ts` | High | Unit | Mock fetch, response parsing |
| start API | `api/start.ts` | High | Unit | Mock fetch, error handling |
| stop API | `api/stop.ts` | High | Unit | Mock fetch, error handling |
| syncDns API | `api/syncDns.ts` | Medium | Unit | Mock fetch, error handling |
| worlds API | `api/worlds.ts` | Medium | Unit | Mock fetch, list/get operations |
| stores | `stores.ts` | High | Unit | Store reactivity, refresh logic |
| types | `types/*.ts` | Low | Type | TypeScript compilation |

**Test Plan:**
1. Create mock fetch utilities for API testing
2. Test each API function with success/error scenarios
3. Test store reactivity and derived state
4. Test type exports and interfaces

### App: Manager (`apps/manager/`)

**Modules to Test:**

| Module | File | Priority | Test Type | Notes |
|--------|------|----------|-----------|-------|
| data.service | `data.service.js` | High | Unit | Already has tests |
| App component | `App.svelte` | Medium | Integration | Full app rendering |

**Existing Tests:**
- `test/data.service.test.js` - Active, working tests

**Test Plan:**
1. Expand data.service tests for edge cases
2. Add App.svelte integration test
3. Add E2E test for full user flow

### App: Worlds (`apps/worlds/`)

**Routes to Test:**

| Route | File | Priority | Test Type | Notes |
|-------|------|----------|-----------|-------|
| Home | `routes/+page.svelte` | High | Integration | World list display |
| Layout | `routes/+layout.svelte` | Medium | Unit | Navigation, structure |
| World detail | Dynamic routes | Medium | Integration | Individual world view |

**Components to Test:**
- Card, Header, Breadcrumbs (if they exist)
- Loading states
- Error boundaries

**Test Plan:**
1. Test route components with mock data
2. Test SvelteKit load functions
3. Add navigation tests
4. Add responsive layout tests

---

## Go CLI (minecraftctl)

### No Tests Currently Exist

**Recommended Testing Framework:**
- Standard `testing` package
- Consider adding `github.com/stretchr/testify` for assertions
- Consider `github.com/spf13/afero` for filesystem mocking

### Package: pkg/worlds/version (`version.go`)

**Priority: Very High** (Pure functions, easy to test)

| Function | Test Cases |
|----------|------------|
| `CompareVersions(v1, v2)` | Equal versions, v1 < v2, v1 > v2, different lengths (1.20 vs 1.20.1), edge cases |
| `parseVersion(v)` | Valid versions, invalid formats, empty string |

**Example Tests:**
```go
func TestCompareVersions(t *testing.T) {
    tests := []struct{v1, v2 string; want int}{
        {"1.20.1", "1.20.1", 0},
        {"1.20.1", "1.20.2", -1},
        {"1.21.0", "1.20.11", 1},
        {"1.20", "1.20.0", 0},
    }
    // ...
}
```

### Package: pkg/util (`paths.go`)

**Priority: Very High** (Pure functions)

| Function | Test Cases |
|----------|------------|
| `ExpandPath(path)` | Home dir (~), env vars, absolute paths, relative paths |
| `AbsPath(path)` | Combined expansion and absolute conversion |

### Package: pkg/nbt (`reader.go`)

**Priority: High** (Has testdata, isolated)

| Function | Test Cases |
|----------|------------|
| `ReadLevelDat(path)` | Valid level.dat (testdata exists), corrupted file, missing file |
| `GetVersionName()` | Old format (compound), new format (DataVersion int) |

**Testdata Available:** `testdata/default/world/level.dat`

### Package: pkg/jars (`jars.go`)

**Priority: High** (File operations, can be tested)

| Function | Test Cases |
|----------|------------|
| `ListJars(jarsDir)` | Empty dir, multiple jars, non-jar files |
| `GetJarInfo(version, jarsDir)` | Existing jar, missing jar, checksum lookup |
| `computeSHA256(path)` | Known hash verification |
| `LoadChecksums()` | Valid file, missing file, malformed lines |
| `SaveChecksum()` | New entry, update existing |
| `VerifyJar()` | Matching hash, mismatched hash, missing checksum |

### Package: pkg/config (`config.go`)

**Priority: High** (Complex, many edge cases)

| Function | Test Cases |
|----------|------------|
| Config loading | YAML parsing, env var override, flag override |
| `MapConfig` parsing | Full config, minimal config, defaults |
| `MapOptions` handling | Bool shadows, string shadows, nil values |
| Validation | Required fields, invalid values |

**Testdata Available:** `testdata/sample-map-config.yml`

### Package: pkg/lock (`lock.go`)

**Priority: Medium** (System interaction)

| Function | Test Cases |
|----------|------------|
| `Lock()` | Acquire lock, release lock |
| `TryLock(timeout)` | Timeout expiry, successful acquire |
| `LockWithOptions()` | Non-blocking, timeout options |
| Signal handling | SIGINT/SIGTERM cleanup |

### Package: pkg/worlds (`worlds.go`)

**Priority: Medium** (Filesystem heavy)

| Function | Test Cases |
|----------|------------|
| `difficultyName(id)` | Valid IDs (0-3), invalid IDs |
| `gameTypeName(id)` | Valid IDs (0-3), invalid IDs |
| `ExpandWorldPattern(pattern)` | Glob patterns, no matches, single match |
| `GetWorldInfo()` | Valid world, missing level.dat |

### Package: pkg/maps (`build.go`, `manifest.go`)

**Priority: Medium** (External dependency on uNmINeD)

| Function | Test Cases |
|----------|------------|
| `Manifest` JSON | Serialization, deserialization |
| `addMapOptions()` | All option combinations |
| `Build()` | Would need uNmINeD mock or integration test |

### Package: pkg/rcon (`client.go`)

**Priority: Low** (Requires running server)

| Function | Test Cases |
|----------|------------|
| `NewClient()` | Connection handling (integration test only) |
| Mock-based tests | Command sending, response parsing |

### CLI Commands (`cmd/minecraftctl/`)

**Priority: Medium** (Cobra testing utilities)

| Command | Test Type |
|---------|-----------|
| `world list` | Unit with mock |
| `world create` | Integration |
| `jar list` | Unit with mock |
| `map build` | Integration |

---

## Lambda Functions

### Recommended Testing Framework

- **pytest** - Test runner
- **moto** - AWS service mocking (EC2, Route53, S3)
- **pytest-asyncio** - Async test support
- **httpx** - FastAPI test client alternative
- Add to each `pyproject.toml` under `[project.optional-dependencies]` or `[tool.uv.dev-dependencies]`

### Lambda: control (`lambda/control/`)

**Framework:** FastAPI + Mangum

**Files to Test:**

| File | Module | Priority | Notes |
|------|--------|----------|-------|
| `app/main.py` | API routes | High | All endpoints |
| `app/config.py` | Configuration | Medium | Env var loading |
| `app/aws_utils.py` | AWS operations | High | EC2, Route53 calls |

**Endpoints to Test:**

| Endpoint | Method | Test Cases |
|----------|--------|------------|
| `/status` | GET | Running instance, stopped instance, no instance |
| `/start` | POST | Successful start, already running, error handling |
| `/stop` | POST | Successful stop, already stopped, error handling |
| `/sync-dns` | POST | Successful sync, Route53 errors |
| CORS | All | Preflight requests, headers |

**Test Plan:**
```python
# Example test structure
from fastapi.testclient import TestClient
from moto import mock_aws

@mock_aws
def test_status_running():
    # Create mock EC2 instance
    # Call /status endpoint
    # Assert response
```

### Lambda: details (`lambda/details/`)

**Framework:** Plain Lambda handler

**Files to Test:**

| File | Module | Priority | Notes |
|------|--------|----------|-------|
| `app/main.py` | Handler | High | mcstatus integration |

**Test Cases:**

| Function | Test Cases |
|----------|------------|
| `handler()` | Server online, server offline, connection error |
| Response format | Player list, version info, MOTD |
| Error handling | Timeout, DNS failure |

**Mocking Strategy:**
- Mock `mcstatus.JavaServer` for predictable responses
- Test actual network calls as integration tests

### Lambda: worlds (`lambda/worlds/`)

**Framework:** Plain Lambda handler

**Files to Test:**

| File | Module | Priority | Notes |
|------|--------|----------|-------|
| `app/main.py` | Handler | High | S3 + URL signing |

**Test Cases:**

| Function | Test Cases |
|----------|------------|
| `handler()` | List worlds, get single world |
| S3 integration | Read manifest, presigned URL generation |
| Error handling | Missing manifest, S3 errors |

**Mocking Strategy:**
```python
from moto import mock_aws

@mock_aws
def test_list_worlds():
    # Create mock S3 bucket with manifest
    # Call handler
    # Assert enriched response with presigned URLs
```

---

## Packer Scripts

### Recommended Testing Framework

- **bats-core** (Bash Automated Testing System)
- **shellcheck** (Static analysis)
- Add CI step for shell script validation

### Scripts to Test

#### High Priority

| Script | File | Test Focus |
|--------|------|------------|
| install_minecraft_jars.sh | `scripts/shared/` | Download, SHA256 verification, error handling |
| autoshutdown.sh | `scripts/minecraft/autoshutdown/` | Player detection, SSH detection, shutdown logic |
| create-world.sh | `scripts/minecraft/create-world/` | Directory creation, minecraftctl calls |

#### Medium Priority

| Script | File | Test Focus |
|--------|------|------------|
| install_base_deps.sh | `scripts/base/` | Package installation (integration) |
| create-minecraft-user.sh | `scripts/base/` | User/group creation |
| Backup scripts | `scripts/minecraft/` | S3 sync, file operations |

### Testing Strategy

**Unit Tests (bats):**
```bash
# Example: test_autoshutdown.bats
@test "detects active players from RCON output" {
    # Mock rcon-cli output
    # Source script functions
    # Assert player count detection
}

@test "skips shutdown when SSH sessions active" {
    # Mock who command
    # Assert no shutdown triggered
}
```

**Static Analysis:**
```bash
shellcheck scripts/**/*.sh
```

**Integration Tests:**
- Test in Docker container mimicking AMI environment
- Validate systemd unit file syntax
- Test full provisioning flow

---

## CI/CD Integration

### Current Workflows

| Workflow | File | Current Tests | Proposed Additions |
|----------|------|---------------|-------------------|
| lambdas.yml | `.github/workflows/` | Syntax only | pytest + moto |
| go.yml | `.github/workflows/` | `go test ./...` | Coverage reporting |
| packer.yml | `.github/workflows/` | HCL syntax | shellcheck, bats |

### Proposed CI Pipeline

```yaml
# Combined test workflow
name: Tests

on: [push, pull_request]

jobs:
  ui-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: pnpm/action-setup@v2
      - run: pnpm install
      - run: pnpm test --filter @minecraft/ui
      - run: pnpm test --filter @minecraft/data
      - run: pnpm test --filter manager
      - run: pnpm test --filter worlds

  go-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
      - run: go test -v -coverprofile=coverage.out ./...
      - run: go tool cover -html=coverage.out -o coverage.html

  lambda-tests:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        lambda: [control, details, worlds]
    steps:
      - uses: actions/checkout@v4
      - uses: astral-sh/setup-uv@v1
      - run: cd lambda/${{ matrix.lambda }} && uv sync --dev
      - run: cd lambda/${{ matrix.lambda }} && uv run pytest

  packer-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - run: shellcheck packer/scripts/**/*.sh
      - run: npm install -g bats
      - run: bats packer/tests/
```

---

## Priority Matrix

### Phase 1: Foundation (Quick Wins) ✅ COMPLETE

| Component | Task | Effort | Impact | Status |
|-----------|------|--------|--------|--------|
| Go CLI | Add version.go tests | Low | High | ✅ Done |
| Go CLI | Add paths.go tests | Low | High | ✅ Done |
| Go CLI | Add nbt/reader.go tests | Low | High | ✅ Done |
| Lambdas | Add pytest + moto to dev deps | Low | Medium | ✅ Done |
| Packer | Add shellcheck to CI | Low | Medium | ✅ Done |

### Phase 2: Core Coverage ✅ COMPLETE

| Component | Task | Effort | Impact | Status |
|-----------|------|--------|--------|--------|
| UI | Fix and unskip @minecraft/ui component tests | Medium | High | ✅ Done (20 tests) |
| UI | Add @minecraft/data API tests | Medium | High | ✅ Done (21 tests) |
| Go CLI | Add jars.go tests | Medium | High | ✅ Done |
| Go CLI | Add config.go tests | Medium | High | ✅ Done |
| Lambdas | Test control lambda endpoints | Medium | High | ✅ Done (19 tests) |

### Phase 3: Integration ✅ COMPLETE

| Component | Task | Effort | Impact | Status |
|-----------|------|--------|--------|--------|
| UI | Add E2E tests for Manager app | High | Medium | Deferred (basic tests exist) |
| UI | Add route tests for Worlds app | Medium | Medium | ✅ Basic tests exist |
| Go CLI | Add worlds.go tests | Medium | Medium | ✅ Done |
| Lambdas | Test details and worlds lambdas | Medium | Medium | ✅ Done (28 tests) |
| Packer | Add bats tests for critical scripts | High | Medium | ✅ Done (53 tests) |

### Phase 4: Comprehensive (Future Work)

| Component | Task | Effort | Impact | Status |
|-----------|------|--------|--------|--------|
| Go CLI | CLI command tests with Cobra | Medium | Low | Not started |
| Go CLI | Add maps.go tests (mock uNmINeD) | High | Low | Not started |
| Go CLI | Add rcon tests (integration) | High | Low | Not started |
| Packer | Full provisioning integration tests | Very High | Low | Not started |
| UI | Add comprehensive E2E tests | High | Medium | Not started |

---

## Success Metrics

| Metric | Initial | Current | Target | Status |
|--------|---------|---------|--------|--------|
| Go CLI coverage | 0% | 37% | 70% | 🟡 On track |
| Go CLI tests | 0 | 282 | 300+ | ✅ Achieved |
| UI lib tests | 0 | 41 | 50+ | 🟡 On track |
| Lambda tests | 0 | 47 | 50+ | ✅ Achieved |
| Packer script lint | No | Yes | Yes | ✅ Achieved |
| Packer bats tests | 0 | 53 | 30+ | ✅ Exceeded |
| CI integration | Partial | Full | Full | ✅ Achieved |
| Total test count | 0 | 427 | 400+ | ✅ Exceeded |

---

## File Structure for Tests

```
minecraft-server-host/
├── lambda/
│   ├── control/
│   │   ├── app/
│   │   └── tests/           # NEW
│   │       ├── conftest.py
│   │       ├── test_main.py
│   │       └── test_aws_utils.py
│   ├── details/
│   │   └── tests/           # NEW
│   └── worlds/
│       └── tests/           # NEW
├── minecraftctl/
│   ├── pkg/
│   │   ├── worlds/
│   │   │   ├── version.go
│   │   │   └── version_test.go  # NEW
│   │   ├── jars/
│   │   │   └── jars_test.go     # NEW
│   │   ├── config/
│   │   │   └── config_test.go   # NEW
│   │   ├── nbt/
│   │   │   └── reader_test.go   # NEW
│   │   └── util/
│   │       └── paths_test.go    # NEW
│   └── testdata/
├── minecraft-ui/
│   ├── libs/
│   │   ├── ui/src/lib/components/
│   │   │   └── *.svelte.test.js  # FIX EXISTING
│   │   └── data/src/lib/
│   │       └── api/*.test.ts     # NEW
│   └── apps/
│       ├── manager/test/
│       └── worlds/src/
│           └── routes/*.test.ts  # NEW
└── packer/
    ├── scripts/
    └── tests/                    # NEW
        ├── test_autoshutdown.bats
        └── test_install_jars.bats
```

---

## Implementation Progress

### Completed Work (December 2025)

**Go CLI (minecraftctl):**
- Created 282 tests across 13 test files
- Achieved 37% code coverage
- Added coverage reporting to CI (go.yml)
- Tests cover: version comparison, path utilities, NBT parsing, jars management, config loading, worlds, maps, and lock handling

**Lambda Functions:**
- Added 47 tests total (control: 19, details: 13, worlds: 15)
- Using pytest + moto for AWS mocking
- Added pytest step to lambdas.yml CI workflow
- Tests cover: all endpoints, error handling, AWS integration

**Packer Scripts:**
- Created 53 tests (30 functional bats tests + 23 shellcheck validations)
- Fixed 3 shellcheck issues in scripts
- Added bats and shellcheck to packer.yml CI workflow
- Tests cover: autoshutdown logic, map rebuilding, world creation

**UI Libraries:**
- Fixed 20 skipped tests in @minecraft/ui
- Added 21 new tests for @minecraft/data API
- Updated test script to run actual tests instead of skip
- Used vi.hoisted() pattern for proper mock hoisting in browser tests

**CI Integration:**
- go.yml: Added coverage reporting
- lambdas.yml: Added pytest step with moto
- packer.yml: Added bats and shellcheck jobs
- web-apps.yml: Already runs `pnpm -r test` (now includes all 45 UI tests)

---

## Next Steps

### Recommended Future Improvements

1. **Increase Go CLI coverage to 70%**
   - Add CLI command tests using Cobra test utilities
   - Add maps.go tests with uNmINeD mocking
   - Add rcon integration tests

2. **Add comprehensive E2E tests**
   - Manager app full user flow testing
   - Worlds app navigation testing
   - Cross-browser testing

3. **Packer integration tests**
   - Docker-based provisioning tests
   - Systemd unit file validation
   - Full AMI build verification

4. **Coverage reporting dashboard**
   - Add coverage badges to README
   - Configure Codecov or similar
   - Set coverage thresholds in CI

This plan should be updated as implementation progresses and new testing needs are discovered.
