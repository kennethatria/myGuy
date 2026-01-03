# Engineering Documentation

This directory contains all technical documentation for the MyGuy platform, including architectural decisions, design documents, completed fixes, and current priorities.

## 📂 Folder Structure

| Folder | Purpose | Contents |
|--------|---------|----------|
| **`❗-current-focus.md`** | **Start here!** | High-level summary of current engineering priorities, recent work, and links to relevant documents. This is the single source of truth for "what we are working on." |
| **`01-proposed/`** | Future work and proposals | Architecture Decision Records (ADRs), design documents, roadmaps, and TODOs for upcoming features and improvements. |
| **`02-reference/`** | Stable documentation | Evergreen architectural and process documentation that describes how the system is currently built. |
| **`03-completed/`** | Historical records | Fix logs, investigations, and reports for completed work. Provides context and learning for future development. |

---

## 📝 Document Naming Conventions

Documents use prefixes to indicate their type and purpose:

### Proposed Work (`01-proposed/`)
| Prefix | Type | Description | Example |
|--------|------|-------------|---------|
| `ADR-` | Architecture Decision Record | Documents architectural decisions with context, options considered, and rationale. | `ADR-backend-testing-strategy.md` |
| `DESIGN-` | Design Document | Detailed technical designs for features or systems. | `DESIGN-browser-push-notifications.md` |
| `ROADMAP-` | Roadmap | Strategic planning documents outlining future work. | `ROADMAP-mvp-prioritization.md` |
| `TODO-` | Action Items | Lists of specific tasks or improvements to be completed. | `TODO-chat-functionality-review.md` |

### Reference Documentation (`02-reference/`)
| Prefix | Type | Description | Example |
|--------|------|-------------|---------|
| `ARCH-` | Architecture | High-level system architecture and design patterns. | `ARCH-chat-service-architecture.md` |
| `REF-` | Reference | Process documentation, checklists, and guides. | `REF-deployment-workflow.md` |

### Completed Work (`03-completed/`)
| Prefix | Type | Description | Example |
|--------|------|-------------|---------|
| `FIXLOG-` | Fix Log | Detailed records of bugs fixed and improvements made. | `FIXLOG-cross-database-queries.md` |
| `INVESTIGATION-` | Investigation | Analysis and research into issues or features. | `INVESTIGATION-message-reading-bugs.md` |
| `REPORT-` | Report | Summary reports on system state or major initiatives. | `REPORT-chat-service-critical-issues.md` |

---

## 🔄 Document Lifecycle

Documents move through the following lifecycle:

```
┌─────────────────┐
│  New Proposal   │
│  (01-proposed/) │
└────────┬────────┘
         │
         ├──> Approved & Designed ──> Implementation Begins
         │
         ├──> Completed ──────────┐
         │                        │
         v                        v
┌─────────────────┐      ┌─────────────────┐
│  Still Relevant │      │   Completed     │
│  (02-reference/)│      │  (03-completed/)│
└─────────────────┘      └─────────────────┘
```

### Lifecycle Rules

1. **New Ideas**: Create documents in `01-proposed/` using the appropriate prefix (ADR-, DESIGN-, ROADMAP-, TODO-)

2. **Implementation**: During implementation, create `FIXLOG-` or `INVESTIGATION-` documents in `03-completed/` to track progress and decisions

3. **Completion**:
   - **One-time work** (fixes, features): Document moves to or stays in `03-completed/`
   - **Ongoing reference** (architecture, processes): Create or update documents in `02-reference/`

4. **Obsolete Proposals**: Documents in `01-proposed/` that are no longer relevant should be:
   - Moved to `03-completed/` with a note explaining why they weren't pursued, OR
   - Deleted if they provide no historical value

---

## 🎯 Quick Navigation

### I want to...

- **See what we're currently working on** → Read `❗-current-focus.md`
- **Understand the system architecture** → Browse `02-reference/ARCH-*.md`
- **Propose a new feature or improvement** → Create a new document in `01-proposed/`
- **Learn how something was fixed** → Search `03-completed/FIXLOG-*.md`
- **Understand why a decision was made** → Look for `01-proposed/ADR-*.md` or `03-completed/ADR-*.md`
- **See what's planned for the future** → Browse `01-proposed/ROADMAP-*.md`
- **Learn deployment processes** → Check `02-reference/REF-deployment-*.md`

---

## ✍️ Writing Guidelines

### For ADRs (Architecture Decision Records)
Follow this structure:
1. **Context**: What is the problem or requirement?
2. **Options Considered**: What alternatives did we evaluate?
3. **Decision**: What did we choose?
4. **Rationale**: Why did we make this choice?
5. **Consequences**: What are the implications (positive and negative)?

### For Design Documents
Include:
1. **Overview**: High-level summary
2. **Goals**: What we're trying to achieve
3. **Non-Goals**: What we're explicitly not doing
4. **Architecture**: Technical design with diagrams
5. **Implementation Plan**: Step-by-step approach
6. **Testing Strategy**: How we'll verify it works
7. **Rollout Plan**: How we'll deploy it

### For Fix Logs
Include:
1. **Summary**: Brief description of what was fixed
2. **Problem**: What was broken and how it manifested
3. **Root Cause**: Why it was broken
4. **Solution**: What was changed
5. **Files Modified**: List of changed files with line numbers
6. **Testing**: How the fix was verified
7. **Impact**: What changed for users

### For Investigations
Include:
1. **Objective**: What we're investigating
2. **Findings**: What we discovered
3. **Analysis**: What the findings mean
4. **Recommendations**: What we should do
5. **References**: Links to relevant code, docs, or issues

---

## 📊 Current Statistics

As of the latest update:

- **Proposed Documents**: 9 (ADRs, designs, roadmaps, TODOs)
- **Reference Documents**: 4 (architecture, deployment, testing)
- **Completed Work**: 12 (fix logs, investigations, reports)

---

## 🔗 External Links

For service-specific documentation, see:
- [Backend README](../backend/README.md)
- [Store Service README](../store-service/README.md)
- [Chat Service README](../chat-websocket-service/README.md)
- [Frontend README](../frontend/README.md)

For the main project overview, see:
- [Main Project README](../README.md)

---

## 🤝 Contributing

When adding new documentation:

1. **Choose the right location**:
   - Future work → `01-proposed/`
   - Stable reference → `02-reference/`
   - Completed work → `03-completed/`

2. **Use the correct prefix**: See [Document Naming Conventions](#-document-naming-conventions)

3. **Follow the writing guidelines**: See [Writing Guidelines](#️-writing-guidelines)

4. **Update `❗-current-focus.md`**: If the document represents a significant priority or completion

5. **Link related documents**: Cross-reference related docs to build a knowledge graph

---

**Last Updated**: 2026-01-03
