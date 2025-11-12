# Implementation Plan: Bootstrapping Script

**Branch**: `001-bootstrapping-script` | **Date**: 2025-11-11 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-bootstrapping-script/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

A bootstrapping script that automates the complete setup of Kalypso Scheduler infrastructure, allowing platform engineers to either create new resources (AKS cluster, control-plane repository, gitops repository) or integrate with existing ones. The script provides both interactive and non-interactive modes, validates prerequisites, handles errors gracefully with rollback capabilities, and generates comprehensive documentation.

## Technical Context

**Language/Version**: Bash 4.0+ (for maximum portability across macOS/Linux)  
**Primary Dependencies**: Azure CLI, kubectl, git, GitHub CLI (gh), jq, curl  
**Storage**: Local filesystem for configuration state, GitHub for repositories, Azure for cluster state  
**Target Platform**: macOS (Darwin) and Linux (Ubuntu/Debian/RHEL-based distributions)  
**Project Type**: CLI script with supporting utilities  
**Performance Goals**: Complete new installation in <15 minutes, existing cluster installation in <5 minutes  
**Constraints**: Must be idempotent, must support non-interactive mode for CI/CD, must validate all prerequisites before execution  
**Scale/Scope**: Single script with modular functions, ~1000-1500 lines of code, support for multiple execution modes (create vs. bring-your-own)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

✅ **Kubebuilder Framework**: N/A - This is a bootstrapping script, not a controller. The script will install and configure Kubebuilder-based components, but is not itself a controller.

✅ **Declarative API Design**: N/A - Not applicable to bash script. The script creates declarative CRDs for Kalypso Scheduler.

✅ **Separation of Concerns**: PASS - Script will be modular with clear function boundaries (cluster creation, repository setup, installation, validation).

✅ **API Versioning**: N/A - No API versioning in bash script.

✅ **User Story Recognition**: PASS - Spec correctly uses the single user story from input and creates comprehensive acceptance scenarios rather than decomposing into sub-stories.

✅ **Development Workflow**: PASS - Will follow standard PR review and CI requirements.

✅ **Quality Gates**: PASS - Will include ShellCheck for static analysis and proper error handling.

**Result**: ✅ NO VIOLATIONS - Proceeding to Phase 0.

---

**Post-Design Re-evaluation** (Phase 1 Complete):

All constitutional principles remain satisfied after completing the design phase:

✅ **Separation of Concerns**: The designed project structure in `scripts/bootstrap/` maintains clear modular separation with dedicated files for prerequisites, cluster operations, repository management, installation, validation, rollback, and utilities.

✅ **User Story Recognition**: The spec correctly uses the single user story provided and expands it with 8 comprehensive acceptance scenarios rather than creating derivative user stories.

✅ **Development Workflow**: The design includes comprehensive validation and follows standard PR/CI/CD patterns.

✅ **Quality Gates**: Research.md documents validation strategy, data-model.md provides structured validation rules, and cli-interface.md specifies comprehensive error handling and exit codes.

**Final Result**: ✅ NO NEW VIOLATIONS INTRODUCED - Design phase complete and constitutional.

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
# Bootstrapping script and documentation
scripts/
├── bootstrap/
│   ├── bootstrap.sh              # Main bootstrapping script entry point
│   ├── lib/
│   │   ├── prerequisites.sh      # Prerequisite validation functions
│   │   ├── cluster.sh            # AKS cluster creation/validation
│   │   ├── repositories.sh       # GitHub repository creation/validation
│   │   ├── installer.sh          # Kalypso Scheduler installation
│   │   ├── validation.sh         # Post-installation validation
│   │   ├── rollback.sh           # Cleanup and rollback functions
│   │   └── utils.sh              # Common utilities and logging
│   └── templates/
│       ├── control-plane/        # Minimal control-plane repo structure
│       │   ├── main/             # Main branch content (promotion source)
│       │   │   ├── .environments/
│       │   │   │   └── dev.yaml
│       │   │   ├── .github/
│       │   │   │   └── workflows/
│       │   │   │       ├── ci.yaml
│       │   │   │       ├── cd.yaml
│       │   │   │       └── check-promote.yaml
│       │   │   ├── templates/
│       │   │   │   └── default-template.yaml
│       │   │   ├── workloads/
│       │   │   │   └── sample-workload-registration.yaml
│       │   │   └── README.md
│       │   └── dev/              # Dev branch content (environment-specific)
│       │       ├── cluster-types/
│       │       │   └── default.yaml
│       │       ├── configs/
│       │       │   └── default-config.yaml
│       │       ├── scheduling-policies/
│       │       │   └── default-policy.yaml
│       │       ├── base-repo.yaml
│       │       ├── gitops-repo.yaml
│       │       └── README.md
│       └── gitops/               # Minimal gitops repo structure
│           ├── main/             # Main branch (empty/minimal)
│           │   └── README.md
│           └── dev/              # Dev branch (cluster directories + workflow)
│               ├── .github/
│               │   └── workflows/
│               │       └── check-promote.yaml
│               └── README.md

docs/
└── bootstrap/
    ├── BOOTSTRAP.md              # Comprehensive bootstrapping manual
    └── TROUBLESHOOTING.md        # Common issues and solutions
```

**Structure Decision**: Single project structure with modular bash library files. The script follows a clear separation of concerns with dedicated modules for each major operation (prerequisites, cluster, repositories, installation). Template files provide seed content for created repositories.

