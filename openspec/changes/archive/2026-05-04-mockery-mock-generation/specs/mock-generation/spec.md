## ADDED Requirements

### Requirement: Mockery configuration file
The project SHALL include a `.mockery.yaml` configuration file at the repository root that defines all interfaces to be mocked.

#### Scenario: Configuration specifies all repository interfaces
- **WHEN** `.mockery.yaml` is read
- **THEN** it contains entries for all 9 repository interfaces: SourceRepository, KnowledgeRepository, NotebookRepository, ConversationRepository, ArtifactRepository, SentenceRepository, ImageRepository, UserRepository, CacheRepository

#### Scenario: Configuration specifies external model interfaces
- **WHEN** `.mockery.yaml` is read
- **THEN** it contains an entry for the `ToolCallingChatModel` interface from the Eino model package

#### Scenario: Configuration defines output paths
- **WHEN** mockery generates mocks
- **THEN** repository mocks are output to `internal/mocks/repositories/`
- **AND** model mocks are output to `internal/mocks/models/`

### Requirement: Makefile target for mock generation
The project SHALL provide a `make mocks` target that regenerates all mock implementations.

#### Scenario: Running make mocks
- **WHEN** a developer runs `make mocks`
- **THEN** mockery generates all configured mock files
- **AND** existing mock files are overwritten with fresh versions
- **AND** the command exits with code 0 on success

#### Scenario: Mockery not installed
- **WHEN** a developer runs `make mocks` without mockery installed
- **THEN** a helpful error message is displayed with installation instructions

### Requirement: Generated mock file format
Each generated mock file SHALL be a valid Go source file using the `testify/mock` package.

#### Scenario: Generated mock is testify-compatible
- **WHEN** a mock is generated for SourceRepository
- **THEN** it defines a struct with `mock.Mock` embedded
- **AND** implements all methods of the SourceRepository interface
- **AND** each method calls `m.Called(...)` and returns appropriate types

#### Scenario: Generated files contain code generation header
- **WHEN** a generated mock file is read
- **THEN** the first line is a comment indicating it is auto-generated
- **AND** includes `DO NOT EDIT` directive

### Requirement: Test migration to generated mocks
All test files SHALL import generated mocks instead of defining inline mock types.

#### Scenario: Test imports generated mock
- **WHEN** a test needs to mock SourceRepository
- **THEN** it imports from `internal/mocks/repositories`
- **AND** creates an instance via `mocks.NewSourceRepository(t)`
- **AND** uses standard testify assertions (`On`, `Return`, `AssertExpectations`)

#### Scenario: No inline mock definitions remain
- **WHEN** the test codebase is searched for `type Mock.* struct`
- **THEN** no hand-written mock types are found in test files
- **AND** all mocks are imported from `internal/mocks/`
