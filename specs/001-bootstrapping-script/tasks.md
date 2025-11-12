---
description: "Implementation tasks for bootstrapping script feature"
---

# Tasks: Bootstrapping Script

**Input**: Design documents from `/specs/001-bootstrapping-script/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/cli-interface.md, quickstart.md

**Tech Stack**: Bash 4.0+, Azure CLI 2.50+, kubectl 1.25+, GitHub CLI 2.0+, Helm 3.0+, jq 1.6+, ShellCheck 0.9+

**User Story**: As a platform engineer I need a bootstrapping script that enables me to:
- Create or bring my own AKS cluster where Kalypso scheduler will be installed
- Bring my own control-plane git repo or create a minimal control plane repo with a single dev environment, single cluster type, single scheduling policy and a single config map
- Bring my own gitops repo or create almost empty gitops repo containing only GitHub workflow and readme
- Install Kalypso Scheduler

**Independent Test**: Run the bootstrapping script in different modes (create new vs. bring your own) and verify that all components are properly configured and Kalypso Scheduler is operational.

---

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[US1]**: User Story 1 (the single bootstrapping script user story)
- Include exact file paths in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [ ] T001 Create directory structure `scripts/bootstrap/` with subdirectories `lib/`, `templates/control-plane/{main,dev}/`, `templates/gitops/{main,dev}/`
- [ ] T002 [P] Create main entry point script `scripts/bootstrap/bootstrap.sh` with shebang, script header, and version constant
- [ ] T003 [P] Initialize documentation structure `docs/bootstrap/` with placeholder files `BOOTSTRAP.md` and `TROUBLESHOOTING.md`
- [ ] T004 [P] Create `.gitignore` entries for `$HOME/.kalypso/` and log files
- [ ] T005 [P] Set up ShellCheck configuration file `.shellcheckrc` with project-specific rules

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story implementation

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [ ] T006 Implement `scripts/bootstrap/lib/utils.sh` with logging functions (log_info, log_warn, log_error, log_debug), color output, timestamp formatting
- [ ] T007 [P] Implement state management functions in `scripts/bootstrap/lib/utils.sh` (save_state, load_state, update_state, backup_state)
- [ ] T008 [P] Implement JSON processing helpers in `scripts/bootstrap/lib/utils.sh` using jq (get_json_field, set_json_field, validate_json)
- [ ] T009 Implement `scripts/bootstrap/lib/prerequisites.sh` with tool detection functions (check_bash_version, check_tool_installed, check_tool_version)
- [ ] T010 [P] Implement authentication validation in `scripts/bootstrap/lib/prerequisites.sh` (check_azure_auth, check_github_auth, get_azure_subscription)
- [ ] T011 Implement comprehensive prerequisite validation in `scripts/bootstrap/lib/prerequisites.sh` (validate_prerequisites function that orchestrates all checks)
- [ ] T012 Implement CLI argument parsing in `scripts/bootstrap/bootstrap.sh` using getopts with all flags from contracts/cli-interface.md
- [ ] T013 Implement configuration file loading in `scripts/bootstrap/bootstrap.sh` (parse YAML using yq or jq, merge with CLI flags)
- [ ] T014 Implement configuration validation in `scripts/bootstrap/bootstrap.sh` (validate cluster name format, repository URLs, namespace names, required conditionals)
- [ ] T015 Implement interactive prompts in `scripts/bootstrap/bootstrap.sh` for missing required values (cluster name, create vs. existing, repository options)
- [ ] T016 Implement configuration confirmation display in `scripts/bootstrap/bootstrap.sh` showing effective configuration before execution
- [ ] T017 Implement error handling framework in `scripts/bootstrap/bootstrap.sh` (trap ERR, handle_error function, exit code constants)
- [ ] T018 [P] Implement `scripts/bootstrap/lib/rollback.sh` with cleanup orchestration (cleanup_cluster, cleanup_repositories, cleanup_installation, full_rollback)

**Checkpoint**: Foundation ready - user story implementation can now begin

---

## Phase 3: User Story Implementation - Bootstrapping Script (Priority: P1) 🎯 MVP

**Goal**: Complete bootstrapping script that enables platform engineers to set up Kalypso Scheduler infrastructure quickly

**Independent Test**: Run script in multiple scenarios (create all new, bring existing cluster, bring existing repos) and verify successful Kalypso Scheduler installation

### Cluster Operations (US1)

- [ ] T019 [P] [US1] Implement cluster creation in `scripts/bootstrap/lib/cluster.sh` (create_aks_cluster function using az aks create)
- [ ] T020 [P] [US1] Implement cluster validation in `scripts/bootstrap/lib/cluster.sh` (validate_existing_cluster function using kubectl)
- [ ] T021 [US1] Implement kubeconfig integration in `scripts/bootstrap/lib/cluster.sh` (get_cluster_credentials, merge_kubeconfig, set_context)
- [ ] T022 [US1] Implement cluster readiness checks in `scripts/bootstrap/lib/cluster.sh` (wait_for_cluster_ready, check_system_pods)
- [ ] T023 [US1] Implement cluster idempotency in `scripts/bootstrap/lib/cluster.sh` (check if cluster exists before creating)

### Repository Operations - Control Plane (US1)

- [ ] T024 [P] [US1] Create control-plane main branch templates in `scripts/bootstrap/templates/control-plane/main/` (.environments/dev.yaml)
- [ ] T025 [P] [US1] Create control-plane main branch workflow templates in `scripts/bootstrap/templates/control-plane/main/.github/workflows/` (ci.yaml, cd.yaml, check-promote.yaml)
- [ ] T026 [P] [US1] Create control-plane main branch Kalypso templates in `scripts/bootstrap/templates/control-plane/main/templates/` (default-template.yaml)
- [ ] T027 [P] [US1] Create control-plane main branch sample workload in `scripts/bootstrap/templates/control-plane/main/workloads/` (sample-workload-registration.yaml)
- [ ] T028 [P] [US1] Create control-plane main branch README in `scripts/bootstrap/templates/control-plane/main/README.md`
- [ ] T029 [P] [US1] Create control-plane dev branch templates in `scripts/bootstrap/templates/control-plane/dev/` (cluster-types/default.yaml, configs/default-config.yaml, scheduling-policies/default-policy.yaml)
- [ ] T030 [P] [US1] Create control-plane dev branch reference files in `scripts/bootstrap/templates/control-plane/dev/` (base-repo.yaml, gitops-repo.yaml)
- [ ] T031 [P] [US1] Create control-plane dev branch README in `scripts/bootstrap/templates/control-plane/dev/README.md`
- [ ] T032 [US1] Implement control-plane repository creation in `scripts/bootstrap/lib/repositories.sh` (create_control_plane_repo using gh repo create)
- [ ] T033 [US1] Implement control-plane template population in `scripts/bootstrap/lib/repositories.sh` (populate_control_plane_templates with variable substitution using envsubst)
- [ ] T034 [US1] Implement control-plane main branch initialization in `scripts/bootstrap/lib/repositories.sh` (create main branch, commit templates, push)
- [ ] T035 [US1] Implement control-plane dev branch initialization in `scripts/bootstrap/lib/repositories.sh` (create dev branch from main, commit dev templates, push)
- [ ] T036 [US1] Implement control-plane repository validation in `scripts/bootstrap/lib/repositories.sh` (validate_control_plane_repo for existing repos)
- [ ] T037 [US1] Implement control-plane repository idempotency in `scripts/bootstrap/lib/repositories.sh` (check if repo exists before creating)

### Repository Operations - GitOps (US1)

- [ ] T038 [P] [US1] Create gitops main branch README template in `scripts/bootstrap/templates/gitops/main/README.md`
- [ ] T039 [P] [US1] Create gitops dev branch workflow template in `scripts/bootstrap/templates/gitops/dev/.github/workflows/check-promote.yaml`
- [ ] T040 [P] [US1] Create gitops dev branch README in `scripts/bootstrap/templates/gitops/dev/README.md`
- [ ] T041 [US1] Implement gitops repository creation in `scripts/bootstrap/lib/repositories.sh` (create_gitops_repo using gh repo create)
- [ ] T042 [US1] Implement gitops template population in `scripts/bootstrap/lib/repositories.sh` (populate_gitops_templates)
- [ ] T043 [US1] Implement gitops main branch initialization in `scripts/bootstrap/lib/repositories.sh` (create main branch, commit README, push)
- [ ] T044 [US1] Implement gitops dev branch initialization in `scripts/bootstrap/lib/repositories.sh` (create dev branch from main, commit workflow, push)
- [ ] T045 [US1] Implement gitops repository validation in `scripts/bootstrap/lib/repositories.sh` (validate_gitops_repo for existing repos)
- [ ] T046 [US1] Implement gitops repository idempotency in `scripts/bootstrap/lib/repositories.sh` (check if repo exists before creating)

### Kalypso Installation (US1)

- [ ] T047 [P] [US1] Implement namespace creation in `scripts/bootstrap/lib/installer.sh` (create_namespace using kubectl)
- [ ] T048 [US1] Implement Helm installation in `scripts/bootstrap/lib/installer.sh` (install_kalypso using helm upgrade --install)
- [ ] T049 [US1] Implement custom values file support in `scripts/bootstrap/lib/installer.sh` (merge default and custom values)
- [ ] T050 [US1] Implement installation wait logic in `scripts/bootstrap/lib/installer.sh` (wait for Helm release to be deployed)

### Validation (US1)

- [ ] T051 [P] [US1] Implement pod health checks in `scripts/bootstrap/lib/validation.sh` (check_pods_running)
- [ ] T052 [P] [US1] Implement CRD registration checks in `scripts/bootstrap/lib/validation.sh` (verify_crds_installed)
- [ ] T053 [P] [US1] Implement controller readiness checks in `scripts/bootstrap/lib/validation.sh` (check_controller_ready)
- [ ] T054 [US1] Implement comprehensive validation orchestration in `scripts/bootstrap/lib/validation.sh` (validate_installation function)

### Main Orchestration (US1)

- [ ] T056 [US1] Implement main execution flow in `scripts/bootstrap/bootstrap.sh` (orchestrate: validate prerequisites → load config → cluster → repos → install → validate)
- [ ] T057 [US1] Implement state file initialization in `scripts/bootstrap/bootstrap.sh` (create initial state with in_progress status)
- [ ] T058 [US1] Implement state updates throughout execution in `scripts/bootstrap/bootstrap.sh` (update after each major step)
- [ ] T059 [US1] Implement success reporting in `scripts/bootstrap/bootstrap.sh` (final state update, summary output, next steps)
- [ ] T060 [US1] Implement error handling integration in `scripts/bootstrap/bootstrap.sh` (trap errors, offer rollback, update state to failed)

### Rollback Implementation (US1)

- [ ] T061 [US1] Implement Helm uninstall in `scripts/bootstrap/lib/rollback.sh` (cleanup_installation using helm uninstall)
- [ ] T062 [US1] Implement namespace deletion in `scripts/bootstrap/lib/rollback.sh` (delete namespace if created)
- [ ] T063 [US1] Implement kubeconfig cleanup in `scripts/bootstrap/lib/rollback.sh` (remove context if added)
- [ ] T064 [US1] Implement cluster deletion in `scripts/bootstrap/lib/rollback.sh` (cleanup_cluster using az aks delete)
- [ ] T065 [US1] Implement repository deletion in `scripts/bootstrap/lib/rollback.sh` (cleanup_repositories with user confirmation using gh repo delete)
- [ ] T066 [US1] Implement state file cleanup in `scripts/bootstrap/lib/rollback.sh` (remove or mark as rolled_back)

### Utility Commands (US1)

- [ ] T067 [P] [US1] Implement `--status` command in `scripts/bootstrap/bootstrap.sh` (read and display state file)
- [ ] T068 [P] [US1] Implement `--clean` command in `scripts/bootstrap/bootstrap.sh` (trigger full_rollback)
- [ ] T069 [P] [US1] Implement `--validate-only` command in `scripts/bootstrap/bootstrap.sh` (run validate_prerequisites and exit)
- [ ] T070 [P] [US1] Implement `--help` command in `scripts/bootstrap/bootstrap.sh` (display usage information)
- [ ] T071 [P] [US1] Implement `--version` command in `scripts/bootstrap/bootstrap.sh` (display script version)

**Checkpoint**: User Story 1 should be fully functional and testable independently

---

## Phase 4: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect the overall bootstrap experience

- [ ] T072 [P] Write comprehensive user documentation in `docs/bootstrap/BOOTSTRAP.md` (Quick Start, Prerequisites, Authentication, Usage Scenarios, Configuration Reference, Architecture, Troubleshooting, Advanced Topics, Uninstallation, FAQ)
- [ ] T073 [P] Write troubleshooting guide in `docs/bootstrap/TROUBLESHOOTING.md` (Common errors, solutions, diagnostic commands)
- [ ] T074 [P] Create example configuration files in `docs/bootstrap/examples/` (create-all-new.yaml, existing-cluster.yaml, existing-repos.yaml, ci-cd.yaml)
- [ ] T075 Code cleanup and ShellCheck validation across all scripts (ensure all scripts pass shellcheck)
- [ ] T076 Add comprehensive inline comments in all library files explaining complex logic
- [ ] T077 [P] Implement log rotation in `scripts/bootstrap/lib/utils.sh` (keep last 10 log files in ~/.kalypso/)
- [ ] T078 [P] Add timing information to log output in `scripts/bootstrap/lib/utils.sh` (show duration for long operations)
- [ ] T079 Implement better error messages throughout all scripts (specific, actionable, with suggestions for resolution)
- [ ] T080 Add progress indicators for long-running operations in `scripts/bootstrap/bootstrap.sh` (spinner or percentage for cluster creation, installation)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user story work
- **User Story (Phase 3)**: Depends on Foundational phase completion
- **Polish (Phase 4)**: Depends on User Story completion

### Within User Story (Phase 3)

**Cluster Operations** (T019-T023):
- T021 depends on T019 or T020 (need cluster before getting credentials)
- T022 depends on T021 (need credentials before readiness checks)
- T023 can run before T019 (idempotency check happens first)

**Control-Plane Repository** (T024-T037):
- T024-T031: All template creation tasks can run in parallel
- T032 must run before T033 (need repo before populating)
- T033 must run before T034, T035 (need templates before committing)
- T034 must run before T035 (dev branch based on main)
- T036, T037 can run in parallel with creation tasks (different code paths)

**GitOps Repository** (T038-T046):
- T038-T040: All template creation tasks can run in parallel
- T041 must run before T042 (need repo before populating)
- T042 must run before T043, T044 (need templates before committing)
- T043 must run before T044 (dev branch based on main)
- T045, T046 can run in parallel with creation tasks (different code paths)

**Installation** (T047-T050):
- T047 must complete before T048 (namespace needed for Helm install)
- T049 runs alongside T048 (values merged during installation)
- T050 depends on T048 (wait after install initiated)

**Validation** (T051-T054):
- T051, T052, T053 can run in parallel (independent checks)
- T054 orchestrates T051-T053 (runs them in sequence or parallel)

**Main Orchestration** (T056-T060):
- T056 is the main flow that orchestrates everything
- T057 must run at start of T056
- T058 runs throughout T056 after each step
- T059 runs at successful completion of T056
- T060 is integrated into T056's error handling

**Rollback** (T061-T066):
- These run in reverse order of creation when triggered:
  - T061 (uninstall Helm) → T062 (delete namespace) → T063 (cleanup kubeconfig) → T064 (delete cluster) → T065 (delete repos with confirmation) → T066 (cleanup state)

**Utility Commands** (T067-T071):
- All can run in parallel (independent commands)

### Parallel Opportunities

**Phase 1 (Setup)**: T002, T003, T004, T005 can all run in parallel

**Phase 2 (Foundational)**:
- Batch 1: T007, T008, T010 (independent utility functions)
- Batch 2: After T006 complete (utils.sh logging needed by others)
- T018 can run in parallel with T009-T017 (different file)

**Phase 3 (User Story)**:
- All template creation tasks (T024-T031, T038-T040) can run in parallel
- After cluster ready: Repository operations and installation preparation can run in parallel
- Validation checks (T051, T052, T053) can run in parallel
- Utility commands (T067-T071) are independent

**Phase 4 (Polish)**:
- T072, T073, T074, T075, T076, T077, T078 can all run in parallel (documentation and code quality)

---

## Implementation Strategy

### MVP First (Complete User Story)

1. Complete Phase 1: Setup (T001-T005)
2. Complete Phase 2: Foundational (T006-T018) - CRITICAL blocking phase
3. Complete Phase 3: User Story Implementation (T019-T071)
4. Complete Phase 4: Polish (T072-T080) for production readiness

### Recommended Task Order for Single Developer

**Week 1: Foundation**
- Day 1: T001-T005 (Setup)
- Day 2-3: T006-T011 (Utils and prerequisites)
- Day 4-5: T012-T018 (Config parsing, validation, error handling, rollback framework)

**Week 2: Core Infrastructure**
- Day 1-2: T019-T023 (Cluster operations)
- Day 3: T024-T031, T038-T040 (All templates in parallel)
- Day 4-5: T032-T037 (Control-plane repo operations)

**Week 3: Installation & Validation**
- Day 1: T041-T046 (GitOps repo operations)
- Day 2: T047-T050 (Kalypso installation)
- Day 3: T051-T055 (Validation)
- Day 4-5: T056-T060 (Main orchestration)

**Week 4: Rollback & Utilities**
- Day 1: T061-T066 (Rollback implementation)
- Day 2: T067-T071 (Utility commands)
- Day 3-5: T072-T080 (Documentation and polish)

### Parallel Team Strategy (3 developers)

**Developer A - Infrastructure Track**:
- Phase 1: Setup (T001-T005)
- Phase 2: Utils, prerequisites, config (T006-T017)
- Phase 3: Cluster operations (T019-T023)
- Phase 3: Main orchestration (T056-T060)

**Developer B - Repository Track**:
- Phase 2: State management (T007)
- Phase 3: Control-plane templates (T024-T031)
- Phase 3: Control-plane operations (T032-T037)
- Phase 3: GitOps templates (T038-T040)
- Phase 3: GitOps operations (T041-T046)

**Developer C - Installation & Validation Track**:
- Phase 2: Rollback framework (T018)
- Phase 3: Installation (T047-T050)
- Phase 3: Validation (T051-T054)
- Phase 3: Rollback implementation (T061-T066)
- Phase 3: Utility commands (T067-T071)

**All Developers - Polish Phase**: T072-T080 (divide documentation and polish tasks)

---

## Notes

- [P] tasks target different files and can run in parallel by multiple developers
- [US1] label indicates all tasks belong to the single bootstrapping user story
- Each task should result in a working, tested component
- Commit after each completed task or logical group of parallel tasks
- State file updates (T057, T058) are critical for idempotency and rollback
- Template tasks (T024-T031, T038-T040) should include variable placeholders documented in data-model.md
- ShellCheck validation (T075) should be run continuously during development, not just at the end

---

## Task Summary

- **Total Tasks**: 80
- **Phase 1 (Setup)**: 5 tasks
- **Phase 2 (Foundational)**: 13 tasks (BLOCKING)
- **Phase 3 (User Story)**: 53 tasks
  - Cluster: 5 tasks
  - Control-Plane Repo: 14 tasks
  - GitOps Repo: 9 tasks
  - Installation: 4 tasks
  - Validation: 4 tasks
  - Orchestration: 5 tasks
  - Rollback: 6 tasks
  - Utilities: 5 tasks
- **Phase 4 (Polish)**: 9 tasks

**Parallel Opportunities**: ~30 tasks marked [P] can run in parallel across different files

**Estimated Effort**: 3-4 weeks for single developer, 1.5-2 weeks for team of 3

**MVP Scope**: All 80 tasks (single user story, fully complete bootstrap capability)
