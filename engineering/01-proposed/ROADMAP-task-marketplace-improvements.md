# Roadmap: Task Marketplace Improvements

**Date:** January 18, 2026
**Status:** Planning
**Related:** `ANALYSIS-task-marketplace-functionality.md`

This roadmap provides an actionable implementation plan for Task Marketplace improvements, organized by priority and sprint cycles.

---

## Overview

The Task Marketplace is **feature-complete for MVP** with core functionality operational. This roadmap focuses on post-MVP enhancements to improve user experience, prevent abuse, and enable growth.

**Current State:**
- ✅ Complete task lifecycle (create → apply → assign → complete → review)
- ✅ Advanced search, filtering, pagination
- ✅ Messaging integration
- ✅ Review system
- ⚠️ **0% test coverage** (P1 priority - must fix before adding features)

---

## Immediate Priority: Backend Testing Foundation

**Status:** 📋 Tracked as P1 in `❗-current-focus.md`

**Problem:** Backend service has 0% test coverage, presenting significant regression risk.

**Action:** Use `store-service` (92% coverage) as blueprint to implement:
- Unit tests for services layer (UserService, TaskService, ReviewService)
- Integration tests for API handlers
- Mock repositories for isolated testing
- CI pipeline with coverage enforcement (minimum 70%)

**Effort:** 2-3 weeks

**Files to Create:**
- `backend/internal/services/task_service_test.go`
- `backend/internal/services/user_service_test.go`
- `backend/internal/services/review_service_test.go`
- `backend/internal/api/handlers_test.go`
- `backend/internal/repositories/mocks/` (mock implementations)

**Reference:** `ADR-backend-testing-strategy.md`

**⚠️ BLOCKING: No new task marketplace features should be added until testing foundation is in place.**

---

## Phase 1: Post-MVP Critical (Sprint 1-2)

**Timeline:** 1-2 months after MVP launch
**Prerequisites:** Backend testing foundation complete

### Sprint 1: Discovery & Anti-Abuse (1 week)

#### 1.1 Task Categories & Tags

**Priority:** P1
**Effort:** 3-5 days
**Value:** Improves discoverability, enables targeted browsing

**Backend Tasks:**
1. Create Category model (id, name, slug, description, icon_name)
2. Create Tag model (id, name)
3. Add many-to-many relationships: Task ↔ Tags
4. Add `category_id` foreign key to Task model
5. Update TaskRepository to support `category_id` and `tags[]` filters
6. Add endpoints:
   - `GET /api/v1/categories` (list all)
   - `GET /api/v1/tags` (list popular tags)
   - Update `GET /api/v1/tasks` to accept `category_id` and `tags[]` params

**Frontend Tasks:**
1. Create CategorySelector component
2. Create TagInput component (autocomplete)
3. Update CreateTaskView to include category selector and tag input
4. Update TaskListView to include category/tag filters
5. Add category/tag badges to task cards

**Testing:**
- Unit tests for category/tag filtering logic
- Integration tests for category/tag endpoints
- E2E test: Create task with category/tags, filter by category

**Seed Data:**
Create initial categories:
- Delivery & Errands
- Tutoring & Education
- Home Services (Handyman, Cleaning, etc.)
- Design & Creative
- Writing & Translation
- Tech & Programming
- Events & Entertainment
- Business & Admin
- Other

**Files to Create:**
- `backend/internal/models/category.go`
- `backend/internal/models/tag.go`
- `backend/migrations/add_categories_tags.sql`
- `frontend/src/components/CategorySelector.vue`
- `frontend/src/components/TagInput.vue`

**Files to Modify:**
- `backend/internal/models/task.go`
- `backend/internal/repositories/task_repository.go`
- `backend/internal/api/handlers.go`
- `frontend/src/views/tasks/CreateTaskView.vue`
- `frontend/src/views/tasks/TaskListView.vue`

---

#### 1.2 Application Limit & Spam Prevention

**Priority:** P1
**Effort:** 4-8 hours
**Value:** Prevents application spam, improves signal-to-noise ratio

**Backend Tasks:**
1. Add unique compound index on `(task_id, applicant_id)` in Application model
2. Update ApplyForTask service to return clear error if duplicate application
3. Add GET endpoint: `GET /api/v1/tasks/:id/my-application` (returns user's application if exists)

**Frontend Tasks:**
1. Call `GET /api/v1/tasks/:id/my-application` on TaskDetailView load
2. If application exists, show "Already Applied" badge instead of "Apply" button
3. Display user's application details (proposed fee, message, status)

**Testing:**
- Unit test: ApplyForTask with duplicate application
- Integration test: POST /tasks/:id/apply twice with same user
- E2E test: Apply for task, refresh page, verify "Already Applied" state

**Files to Modify:**
- `backend/internal/models/task.go` (add unique index)
- `backend/internal/services/task_service.go` (handle ErrDuplicateApplication)
- `backend/internal/api/handlers.go` (add GET /tasks/:id/my-application)
- `frontend/src/views/tasks/TaskDetailView.vue`

---

### Sprint 2: Trust & Transparency (1 week)

#### 2.1 Task Edit Restrictions

**Priority:** P1
**Effort:** 4-6 hours
**Value:** Prevents bait-and-switch, builds user trust

**Backend Tasks:**
1. Update UpdateTask service to check if task has applications
2. If applications exist:
   - Allow editing only: `is_messages_public` (privacy toggle)
   - Reject edits to: `title`, `description`, `fee`, `deadline`
   - Return error: "Cannot edit task with existing applications"

**Frontend Tasks:**
1. Fetch application count when loading TaskDetailView
2. If count > 0, disable edit form fields (except privacy toggle)
3. Show warning message: "Editing locked: X applications received"
4. Add tooltip explaining why editing is locked

**Testing:**
- Unit test: UpdateTask with applications → should reject
- Integration test: Create task, apply, attempt to update fee → 403
- E2E test: Full workflow with edit attempt after application

**Files to Modify:**
- `backend/internal/services/task_service.go`
- `frontend/src/views/tasks/TaskDetailView.vue`

---

#### 2.2 Deadline Approaching Notifications

**Priority:** P1
**Effort:** 2-3 days
**Value:** Reduces missed deadlines, improves completion rate

**Backend Tasks:**
1. Create NotificationService (email or in-app)
2. Create cron job (runs every hour):
   - Find tasks with status='in_progress' and deadline within [48h, 24h, 6h]
   - Check if notification already sent for this threshold
   - Send notification to creator and assignee
3. Add NotificationLog model (user_id, task_id, type, sent_at)
4. Integration: SendGrid or AWS SES for email delivery

**Frontend Tasks:**
1. Add "Notification Preferences" section to profile settings
2. Toggle options: Email notifications ON/OFF, notification timing preferences

**Testing:**
- Unit test: Cron job logic (mock time)
- Integration test: Create task with deadline in 23h, run cron, verify notification sent
- Manual test: Verify email delivery and content

**Infrastructure:**
- Set up SendGrid or AWS SES account
- Add SMTP credentials to .env

**Files to Create:**
- `backend/internal/services/notification_service.go`
- `backend/internal/models/notification_log.go`
- `backend/cmd/cron/deadline_notifier.go` (cron job)
- Email templates (HTML + plain text)

**Files to Modify:**
- `backend/internal/models/user.go` (add notification_preferences JSON field)
- `frontend/src/views/ProfileView.vue` (add notification settings)

---

## Phase 2: User Experience Enhancements (Sprint 3-5)

**Timeline:** 2-4 months after MVP launch
**Prerequisites:** Phase 1 complete, user feedback gathered

### Sprint 3: Favorites & Matching (1.5 weeks)

#### 3.1 Saved/Favorited Tasks

**Priority:** P2
**Effort:** 1-2 days
**Value:** Allows users to bookmark interesting tasks

**Backend Tasks:**
1. Create SavedTask model (id, user_id, task_id, saved_at)
2. Add repository: SavedTaskRepository
3. Add endpoints:
   - `POST /api/v1/user/saved-tasks/:taskId` (save task)
   - `DELETE /api/v1/user/saved-tasks/:taskId` (unsave)
   - `GET /api/v1/user/saved-tasks` (list saved tasks with pagination)

**Frontend Tasks:**
1. Add "Save" icon (bookmark) to TaskListView cards
2. Toggle saved state on click
3. Add "Saved Gigs" tab to DashboardView
4. Show saved tasks with "Unsave" option

**Testing:**
- Unit test: Save/unsave logic
- Integration test: POST /user/saved-tasks/:id
- E2E test: Save task from list, verify in dashboard

**Files to Create:**
- `backend/internal/models/saved_task.go`
- `backend/internal/repositories/saved_task_repository.go`
- `backend/internal/services/saved_task_service.go`

**Files to Modify:**
- `backend/internal/api/handlers.go`
- `frontend/src/views/tasks/TaskListView.vue`
- `frontend/src/views/tasks/DashboardView.vue`
- `frontend/src/stores/tasks.ts`

---

#### 3.2 Skill Requirements & Matching

**Priority:** P2
**Effort:** 5-7 days
**Value:** Enables skill-based search and matching

**Backend Tasks:**
1. Create Skill model (id, name, slug, category)
2. Create many-to-many relationships:
   - Task ↔ Skills (required skills for task)
   - User ↔ Skills (user's skills)
3. Add endpoints:
   - `GET /api/v1/skills` (list all skills)
   - `POST /api/v1/user/skills` (add skills to profile)
   - `GET /api/v1/users/:id/skills` (get user's skills)
4. Update task filtering: `GET /api/v1/tasks?skills[]=1&skills[]=5`
5. Add skill match calculation:
   - When fetching applications, return `skill_match_percentage` for each applicant
   - Formula: (matching skills / required skills) * 100

**Frontend Tasks:**
1. Create SkillSelector component (multi-select with autocomplete)
2. Update CreateTaskView: Add "Required Skills" section
3. Update ProfileView: Add "My Skills" section
4. Update TaskListView: Add skill filter dropdown
5. Update ApplicationDetail: Show skill match percentage badge
6. Add skill badges to task cards and user profiles

**Testing:**
- Unit test: Skill match calculation
- Integration test: Filter tasks by skills
- E2E test: Create task with skills, apply with matching user, verify match %

**Seed Data:**
Popular skills by category:
- Tech: JavaScript, Python, React, Node.js, SQL, etc.
- Design: Photoshop, Figma, Illustration, UX Design, etc.
- Writing: Copywriting, Technical Writing, Editing, etc.

**Files to Create:**
- `backend/internal/models/skill.go`
- `backend/internal/services/skill_service.go`
- `frontend/src/components/SkillSelector.vue`
- `backend/migrations/add_skills.sql`

**Files to Modify:**
- `backend/internal/models/task.go`
- `backend/internal/models/user.go`
- `backend/internal/repositories/task_repository.go`
- `backend/internal/api/handlers.go`
- `frontend/src/views/tasks/CreateTaskView.vue`
- `frontend/src/views/ProfileView.vue`
- `frontend/src/views/tasks/TaskListView.vue`
- `frontend/src/components/ApplicationDetail.vue`

---

### Sprint 4: Files & Templates (1 week)

#### 4.1 File Attachments

**Priority:** P2
**Effort:** 5-7 days
**Value:** Enables sharing requirements and deliverables

**Backend Tasks:**
1. Create TaskAttachment model (id, task_id, uploaded_by, file_name, file_path, file_size, mime_type, uploaded_at)
2. Add file upload endpoints:
   - `POST /api/v1/tasks/:id/attachments` (upload file)
   - `GET /api/v1/tasks/:id/attachments` (list files)
   - `DELETE /api/v1/tasks/:id/attachments/:fileId` (delete file, uploader only)
3. Integrate cloud storage (AWS S3 or Google Cloud Storage)
4. Implement virus scanning (ClamAV or third-party API)
5. Validation:
   - Max file size: 10MB per file
   - Allowed types: PDF, images (jpg, png), zip, docs (docx, xlsx)
   - Max 5 files per task

**Frontend Tasks:**
1. Create FileUpload component (drag-and-drop + file picker)
2. Add file upload section to CreateTaskView
3. Add attachments section to TaskDetailView (list + download links)
4. Show file previews for images
5. Add file upload to ChatWindow (for deliverables)

**Testing:**
- Unit test: File validation logic
- Integration test: Upload valid/invalid files
- E2E test: Upload file, view in task details, download

**Security:**
- Generate signed URLs for private file access (S3 presigned URLs)
- Store files with randomized names to prevent enumeration
- Validate file types by content (not just extension)

**Infrastructure:**
- Set up S3 bucket or GCS bucket
- Configure CORS for file uploads
- Set up CloudFront or Cloud CDN (optional)

**Files to Create:**
- `backend/internal/models/task_attachment.go`
- `backend/internal/services/file_storage_service.go`
- `backend/internal/repositories/task_attachment_repository.go`
- `frontend/src/components/FileUpload.vue`

**Files to Modify:**
- `backend/internal/api/handlers.go`
- `frontend/src/views/tasks/CreateTaskView.vue`
- `frontend/src/views/tasks/TaskDetailView.vue`
- `frontend/src/components/chat/ChatWindow.vue`

---

#### 4.2 Task Templates

**Priority:** P2
**Effort:** 2-3 days
**Value:** Saves time for repeat task posters

**Backend Tasks:**
1. Create TaskTemplate model (id, user_id, name, title, description, default_fee, category_id, tags, skills)
2. Add endpoints:
   - `POST /api/v1/user/task-templates` (save template)
   - `GET /api/v1/user/task-templates` (list user's templates)
   - `DELETE /api/v1/user/task-templates/:id`

**Frontend Tasks:**
1. Add "Save as Template" checkbox to CreateTaskView
2. Add "Use Template" dropdown to CreateTaskView (populates form)
3. Add "My Templates" tab to DashboardView

**Testing:**
- Unit test: Template CRUD operations
- E2E test: Save template, use template to create task

**Files to Create:**
- `backend/internal/models/task_template.go`
- `backend/internal/repositories/task_template_repository.go`
- `backend/internal/services/task_template_service.go`

**Files to Modify:**
- `backend/internal/api/handlers.go`
- `frontend/src/views/tasks/CreateTaskView.vue`
- `frontend/src/views/tasks/DashboardView.vue`

---

### Sprint 5: Advanced Search (1 week)

#### 5.1 Location-Based Search

**Priority:** P2
**Effort:** 5-7 days
**Value:** Enables local service marketplace

**Backend Tasks:**
1. Add location fields to Task model:
   - `is_remote` (boolean, default true)
   - `city` (string, nullable)
   - `country` (string, nullable)
   - `latitude`, `longitude` (float, nullable for future proximity search)
2. Add location filtering to TaskRepository:
   - Filter by `is_remote`
   - Filter by `city` and/or `country`
   - (Future) Proximity search: tasks within X km of user location
3. Integrate geocoding service (Google Maps Geocoding API or Mapbox)

**Frontend Tasks:**
1. Add location section to CreateTaskView:
   - Toggle: "Remote" or "In-Person"
   - If in-person: City and Country selectors (autocomplete)
2. Add location filter to TaskListView:
   - Toggle: "Remote Only" / "In-Person Only" / "All"
   - City/Country filters
3. Add location badge to task cards

**Testing:**
- Unit test: Location filtering logic
- Integration test: Filter tasks by city
- E2E test: Create in-person task, filter by location

**Infrastructure:**
- Set up geocoding API key (Google or Mapbox)
- Add API key to .env

**Privacy Considerations:**
- Don't show exact addresses, only city-level
- Add privacy notice: "Your exact address will not be shared publicly"

**Files to Modify:**
- `backend/internal/models/task.go`
- `backend/internal/repositories/task_repository.go`
- `backend/internal/api/handlers.go`
- `frontend/src/views/tasks/CreateTaskView.vue`
- `frontend/src/views/tasks/TaskListView.vue`

---

## Phase 3: Platform Maturity (6-12 months)

**Timeline:** 6-12 months after MVP launch
**Prerequisites:** User base growth, legal/compliance resources

**Note:** These features require significant infrastructure, legal compliance, and third-party integrations. Defer until platform has proven product-market fit.

### 3.1 Payment Integration & Escrow

**Effort:** 3-4 weeks
**Dependencies:**
- Legal: Terms of Service, Refund Policy, Dispute Resolution Policy
- Compliance: PCI-DSS (if storing card data), KYC/AML
- Finance: Merchant account setup, tax handling

**High-Level Implementation:**
1. Integrate Stripe or PayPal
2. Escrow flow:
   - Creator funds task when accepting application
   - Funds held in platform escrow account
   - Released to assignee when creator marks task complete
   - Automatic release after X days if no dispute
3. Platform fee: 10-15% of task fee
4. Payout system: Assignees withdraw to bank account
5. Refund handling: Partial/full refunds based on dispute resolution

**Files to Create:**
- `backend/internal/services/payment_service.go`
- `backend/internal/models/transaction.go`
- `backend/internal/models/escrow.go`
- `frontend/src/views/payment/` (payment UI)

---

### 3.2 Dispute Resolution System

**Effort:** 4-5 weeks
**Dependencies:** Payment integration, admin dashboard, customer support team

**High-Level Implementation:**
1. Add "Open Dispute" option for completed tasks
2. Dispute workflow:
   - Either party opens dispute with reason
   - Task status → "disputed"
   - Evidence submission period (7 days): messages, files
   - Admin review and decision
   - Resolution: refund, partial payment, full payment to assignee
3. Admin dashboard: Review disputes, view evidence, make decisions

**Files to Create:**
- `backend/internal/models/dispute.go`
- `backend/internal/services/dispute_service.go`
- `frontend/src/views/admin/DisputeDashboard.vue`

---

### 3.3 User Verification & Trust Badges

**Effort:** 3-4 weeks
**Dependencies:** Third-party identity verification API (e.g., Stripe Identity, Onfido)

**High-Level Implementation:**
1. ID verification: Upload passport/driver's license, selfie
2. Background checks (optional, premium feature)
3. Trust badges: "ID Verified", "Background Checked", "Top Rated" (100+ reviews with 4.5+ avg)
4. Display badges on user profile, task cards, applications

**Files to Create:**
- `backend/internal/services/verification_service.go`
- `backend/internal/models/verification.go`
- `frontend/src/views/VerificationView.vue`

---

### 3.4 Milestone-Based Tasks

**Effort:** 2-3 weeks
**Dependencies:** Payment integration

**High-Level Implementation:**
1. Allow task creator to define milestones at creation:
   - Milestone 1: "Design mockups" - 30% ($150)
   - Milestone 2: "Development" - 50% ($250)
   - Milestone 3: "Final delivery" - 20% ($100)
2. Each milestone has separate approval and payment flow
3. Assignee submits milestone for review
4. Creator approves → payment released for that milestone

**Files to Create:**
- `backend/internal/models/milestone.go`
- `backend/internal/services/milestone_service.go`
- `frontend/src/components/MilestoneManager.vue`

---

### 3.5 Hourly Pricing & Time Tracking

**Effort:** 2-3 weeks (hourly pricing) + 2 weeks (time tracking) = 4-5 weeks total

**High-Level Implementation:**
1. Add `pricing_type` field to Task: "fixed" or "hourly"
2. Add `hourly_rate` field
3. Time tracking:
   - Assignee clicks "Start Timer" / "Stop Timer"
   - TimeEntry model stores start_time, end_time, duration
   - Creator can review and approve/dispute time logs
4. Payment: hourly_rate * approved_hours

**Files to Create:**
- `backend/internal/models/time_entry.go`
- `backend/internal/services/time_tracking_service.go`
- `frontend/src/components/TimeTracker.vue`

**Files to Modify:**
- `backend/internal/models/task.go`
- `frontend/src/views/tasks/CreateTaskView.vue`
- `frontend/src/views/tasks/TaskDetailView.vue`

---

## Technical Debt & Performance

### Database Indexes (P2)

**Effort:** 1 day

**Add indexes to prevent full table scans:**
```sql
-- Task table
CREATE INDEX idx_tasks_status_created_at ON tasks(status, created_at);
CREATE INDEX idx_tasks_created_by ON tasks(created_by);
CREATE INDEX idx_tasks_assigned_to ON tasks(assigned_to);
CREATE INDEX idx_tasks_deadline ON tasks(deadline);
CREATE INDEX idx_tasks_category_id ON tasks(category_id);

-- Application table
CREATE INDEX idx_applications_task_applicant ON applications(task_id, applicant_id);
CREATE INDEX idx_applications_applicant_id ON applications(applicant_id);

-- Review table
CREATE INDEX idx_reviews_reviewed_user ON reviews(reviewed_user_id);
```

---

### Rate Limiting (P2)

**Effort:** 1-2 days

**Prevent abuse and DDoS:**
1. Add rate limiting middleware (e.g., `github.com/ulule/limiter`)
2. Limits:
   - General API: 100 requests/minute per user
   - Task creation: 10 tasks/hour per user
   - Applications: 20 applications/hour per user
   - Search: 30 requests/minute per user

**Files to Create:**
- `backend/internal/middleware/rate_limiter.go`

---

### API Documentation (P2)

**Effort:** 2-3 days

**Generate OpenAPI/Swagger spec:**
1. Add swaggo annotations to handlers
2. Generate `docs/swagger.json`
3. Host Swagger UI at `/api/docs`

**Files to Modify:**
- `backend/internal/api/handlers.go` (add annotations)

---

## Success Metrics

### Phase 1 Success Criteria

- Task categories adopted: 70%+ of new tasks have category assigned
- Application spam reduced: <5% of tasks receive duplicate applications
- Task edit abuse: 0 reported bait-and-switch incidents
- Deadline compliance: 80%+ of tasks completed before deadline (up from baseline)

### Phase 2 Success Criteria

- Saved tasks usage: 30%+ of users save at least 1 task
- Skill matching: 50%+ of applications have 80%+ skill match
- File attachments: 40%+ of tasks include at least 1 file
- Location filtering: 25%+ of searches use location filter

### Phase 3 Success Criteria

- Payment adoption: 90%+ of tasks use platform payment (once implemented)
- Dispute rate: <3% of completed tasks
- Verification rate: 50%+ of users complete ID verification
- Hourly tasks: 20%+ of tasks use hourly pricing

---

## Risk Assessment

### Technical Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Migration breaks existing data | Low | High | Thorough testing, rollback plan, database backups |
| Performance degradation | Medium | Medium | Add indexes, load testing before deploy |
| Third-party API outage (payment, geocoding) | Medium | High | Implement fallbacks, retry logic, circuit breakers |
| File storage costs exceed budget | Low | Medium | Set strict file size limits, implement CDN caching |

### Business Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Features not adopted by users | Medium | Medium | Gather user feedback before building, A/B test |
| Scope creep delays MVP | High | Medium | Strict adherence to roadmap, defer P3 items |
| Payment integration compliance issues | Low | High | Consult legal counsel, use PCI-compliant providers |
| Dispute resolution overhead | Medium | Medium | Clear policies, automate where possible |

---

## Conclusion

This roadmap provides a clear path from MVP to platform maturity:

**Immediate:** Fix backend testing foundation (P1 blocker)

**Phase 1 (1-2 months):** Anti-abuse, discovery, and transparency features

**Phase 2 (2-4 months):** User experience enhancements based on feedback

**Phase 3 (6-12 months):** Payment, trust, and advanced features for scaling

**Key Principles:**
- ✅ Do not add features until backend testing is in place
- ✅ Prioritize based on user feedback and MVP learnings
- ✅ Defer complex features (payment, disputes) until platform validates product-market fit
- ✅ Maintain test coverage as features are added (target: 70%+)

**Next Steps:**
1. Complete backend testing foundation
2. Gather user feedback from MVP launch
3. Prioritize Phase 1 features based on feedback
4. Execute sprints with continuous deployment

---

**Document Version:** 1.0
**Last Updated:** January 18, 2026
**Next Review:** After MVP launch + 30 days
