# FIXLOG: Documentation Reorganization and Standardization (2026-01-03)

## Summary

This log documents a major reorganization and standardization effort across the entire project's documentation. The primary goal was to enhance clarity, improve navigation, merge redundant content, and establish consistent naming conventions.

## Key Changes and Adjustments

### 1. Renamed `improvements` Folder to `engineering`
-   The top-level `improvements` directory, which contained various documentation, was renamed to `engineering` to better reflect its scope as a central repository for technical documentation, architectural decisions, and process-related information.

### 2. Implemented Status-Based Folder Structure
-   Inside the new `engineering/` directory, a status-based subfolder structure was introduced to clearly delineate the lifecycle and relevance of documents:
    -   `01-proposed/`: For design documents, architectural decision records (ADRs), and roadmaps for future work.
    -   `02-reference/`: For evergreen architectural overviews, deployment processes, and general reference material.
    -   `03-completed/`: For historical fix logs, post-mortems, and investigation reports for completed work.

### 3. Adopted Standardized Naming Conventions
-   All documents within the `engineering/` folder were renamed using a `TYPE-topic.md` convention (e.g., `ADR-backend-testing-strategy.md`, `DESIGN-browser-push-notifications.md`, `FIXLOG-2026-01-03.md`). This makes the purpose of each document immediately clear from its filename.

### 4. Created `❗-current-focus.md` (The "Dashboard")
-   A new file, `engineering/❗-current-focus.md`, was created to serve as the single source of truth for the project's current engineering priorities. It provides an at-a-glance summary of top priorities, upcoming tasks, and recently completed work, with direct links to detailed documents.

### 5. Consolidated Service-Level Documentation
-   **`store-service/`**: Multiple markdown files (e.g., `TESTING.md`, `IMAGE_STORAGE.md`, `BOOKING_FLOW.md`, `ENHANCED_TEST_COVERAGE.md`, various `coverage_summary.md` files) were merged into a single, comprehensive `store-service/README.md`. This `README.md` now covers all aspects of the service, from features and API to testing and key workflows. All original redundant files were deleted.
-   **`chat-websocket-service/`**: Content from `chat-websocket-service/MIGRATIONS.md` and `chat-websocket-service/docs/STORE_MESSAGE_INTEGRATION.md` was merged into the main `chat-websocket-service/README.md`. The `docs/` directory and `MIGRATIONS.md` were subsequently removed. The `README.md` was also updated to correctly reflect the unified `messages` table architecture.
-   **`backend/`**: The detailed `current_state.md` was moved and renamed to `backend/README.md`, providing a high-quality, up-to-date overview for the main Go backend service.

### 6. Streamlined Root Documentation
-   **New Root `README.md`**: A new, concise `README.md` was created at the project root. It now provides a high-level project overview, outlines the architecture, offers a quick start guide using Docker Compose, and acts as a central hub, linking to the `engineering/❗-current-focus.md` and the `README.md`s of individual services.
-   **Product Requirements**: The `myguy_product_requirement.md` file's core information was integrated into the new root `README.md`, and the original file was deleted.
-   **Deployment Checklist**: `DEPLOYMENT_CHECKLIST.md` was moved to `engineering/02-reference/REF-deployment-checklist.md` as it is a process-oriented reference document.

### 7. Clean-up of Ephemeral/Redundant Files
-   `claude.md`, `project.md`, `myguy_product_requirement.md`, `current_state.md`, `DEPLOYMENT_CHECKLIST.md`, and all other redundant markdown files within the service directories and root were either merged or deleted.

## Impact

-   **Enhanced Clarity**: Easier for developers (especially new team members) to understand project scope, architecture, and current focus.
-   **Improved Navigation**: Clear pathways to find specific documentation, whether it's an architectural decision, a service's API, or a historical fix.
-   **Reduced Redundancy**: Elimination of duplicate and conflicting information.
-   **Standardized Approach**: Consistent file naming and folder structure across all documentation.
-   **Single Source of Truth**: The `engineering/❗-current-focus.md` provides an immediate overview of project priorities.
