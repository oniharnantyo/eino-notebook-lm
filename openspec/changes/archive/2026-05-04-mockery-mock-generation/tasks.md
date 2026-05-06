## 1. Setup

- [ ] 1.1 Install mockery (`go install github.com/vektra/mockery/v3@latest`)
- [ ] 1.2 Create `.mockery.yaml` configuration file with all repository and model interfaces
- [ ] 1.3 Add `mocks` target to Makefile with mockery-not-found error handling

## 2. Generate Mocks

- [ ] 2.1 Create `internal/mocks/repositories/` directory
- [ ] 2.2 Create `internal/mocks/models/` directory
- [ ] 2.3 Run `make mocks` to generate all mock files
- [ ] 2.4 Verify all 10 mock files are generated (9 repos + 1 model)
- [ ] 2.5 Verify generated files compile (`go build ./internal/mocks/...`)

## 3. Migrate Tests

- [ ] 3.1 Update `internal/core/application/agent/catalog_test.go` — replace inline `MockSourceRepository` with generated mock
- [ ] 3.2 Update `internal/core/application/agent/agent_test.go` — replace inline `MockChatModel` with generated mock
- [ ] 3.3 Update `internal/core/application/usecases/response/stages/agent_stage_test.go` — replace inline `MockToolCallingChatModel` and `MockSourceRepository`
- [ ] 3.4 Update `internal/core/application/usecases/source/usecase_test.go` — replace inline mock with generated mock

## 4. Cleanup

- [ ] 4.1 Remove all inline `type Mock.* struct` definitions from test files
- [ ] 4.2 Remove all inline `func (m *Mock...)` method implementations from test files
- [ ] 4.3 Verify no remaining hand-written mocks (`grep -r "type Mock.* struct" --include="*_test.go"`)

## 5. Verification

- [ ] 5.1 Run full test suite: `make test`
- [ ] 5.2 Run linter: `make lint`
- [ ] 5.3 Build application: `make build`
- [ ] 5.4 Commit generated mock files to repository
