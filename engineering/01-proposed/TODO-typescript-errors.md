# TypeScript Type Errors - Technical Debt Tracker

**Status:** 📋 Tracked Technical Debt
**Priority:** P2 - Should be resolved before production launch
**Date Identified:** January 4, 2026
**Total Errors:** 62 type errors across 10 files

---

## Overview

During the unified booking feature implementation, TypeScript type checking revealed 62 pre-existing type errors across the frontend codebase. While these errors don't prevent the application from building and running (Vite build succeeds), they represent technical debt that should be addressed for better type safety and maintainability.

**Important:** None of these errors are related to the new booking feature implementation. The booking components (BookingMessageBubble, updated MessageThread, ChatWidget, and chat store) all have correct typing.

---

## Error Summary by File

| File | Error Count | Severity | Category |
|------|-------------|----------|----------|
| `TaskDetailView.vue` | 24 | High | Null checks, property naming |
| `TaskListView.vue` | 4 | Medium | Implicit any types |
| `StoreItemView.vue` | 11 | High | Type narrowing issues |
| `ApplicationDetail.vue` | 4 | Medium | Property naming, type mismatch |
| `ChatWidget.vue` | 6 | Low | Event signatures, NodeJS namespace |
| `MessageThread.vue` | 1 | Low | NodeJS namespace |
| `ProfileView.vue` | 4 | Medium | Type mismatches |
| `UserProfileView.vue` | 1 | Low | Type mismatch |
| `App.vue` | 1 | High | Missing property |
| `tasks.ts` (store) | 1 | Low | Implicit any |

---

## Detailed Error List

### 1. App.vue (1 error)

**Error:** Missing `isAuthenticated` property on auth store
```
src/App.vue(22,20): error TS2339: Property 'isAuthenticated' does not exist on type 'Store<"auth", ...'.
```

**Issue:** The code references `authStore.isAuthenticated` but the property doesn't exist in the store definition.

**Fix Strategy:**
- Add `isAuthenticated` computed property to auth store, OR
- Replace with `authStore.user !== null` check

**Priority:** High (affects authentication flow)

---

### 2. TaskDetailView.vue (24 errors)

**Category 1: Property Naming (camelCase vs snake_case)**
```
src/views/tasks/TaskDetailView.vue(64,17): error TS2551: Property 'createdBy' does not exist on type '...'. Did you mean 'created_by'?
src/views/tasks/TaskDetailView.vue(73,19): error TS2551: Property 'assignedTo' does not exist on type '...'. Did you mean 'assigned_to'?
```

**Issue:** Template uses camelCase but TypeScript interface defines snake_case properties.

**Fix Strategy:**
- Update template to use snake_case (`task.created_by`, `task.assigned_to`), OR
- Add camelCase aliases to Task interface

**Count:** 6 errors

---

**Category 2: Null Safety Checks**
```
src/views/tasks/TaskDetailView.vue(101,58): error TS18047: '__VLS_ctx.task' is possibly 'null'.
src/views/tasks/TaskDetailView.vue(420,9): error TS18047: 'task.value' is possibly 'null'.
```

**Issue:** Code accesses `task` properties without checking if task is null.

**Fix Strategy:**
- Add null checks: `v-if="task"` in template
- Use optional chaining: `task?.property`
- Add non-null assertions where guaranteed: `task!.property`

**Count:** 12 errors

---

**Category 3: Type Mismatches**
```
src/views/tasks/TaskDetailView.vue(117,16): error TS2739: Type '{ id: number; applicant: { id: number; username: string; }; proposedFee: number; ... }' is missing the following properties from type 'Application': task_id, applicant_id, proposed_fee
```

**Issue:** Application object from API uses snake_case but local type uses different casing.

**Fix Strategy:**
- Map API response to match TypeScript interface
- Update interface to match API response format

**Count:** 2 errors

---

**Category 4: Comparison Issues**
```
src/views/tasks/TaskDetailView.vue(135,34): error TS2367: This comparison appears to be unintentional because the types '"in_progress" | "completed"' and '"assigned"' have no overlap.
```

**Issue:** Comparing task.status with invalid value "assigned" (not in TaskStatus enum).

**Fix Strategy:**
- Remove or fix invalid comparison
- Update TaskStatus type if "assigned" should be valid

**Count:** 1 error

---

**Category 5: Type Assignments**
```
src/views/tasks/TaskDetailView.vue(416,5): error TS2322: Type 'Task' is not assignable to type 'Task | { id: number; ... } | null'.
```

**Issue:** Task types from different sources don't match.

**Fix Strategy:**
- Unify Task type definitions across codebase
- Use consistent snake_case or camelCase throughout

**Count:** 3 errors

**Priority:** High (core feature with many type safety issues)

---

### 3. TaskListView.vue (4 errors)

**Error:** Implicit 'any' types in pagination logic
```
src/views/tasks/TaskListView.vue(293,9): error TS7034: Variable 'rangeWithDots' implicitly has type 'any[]'
src/views/tasks/TaskListView.vue(294,7): error TS7034: Variable 'l' implicitly has type 'any'
```

**Issue:** Variables don't have explicit type annotations.

**Fix Strategy:**
```typescript
const rangeWithDots: (number | string)[] = [];
let l: number = lastPage;
```

**Priority:** Low (pagination still works, just needs explicit types)

---

### 4. StoreItemView.vue (11 errors)

**Error:** Type narrowing issue with `item` ref
```
src/views/store/StoreItemView.vue(18,27): error TS2339: Property 'images' does not exist on type 'never'.
src/views/store/StoreItemView.vue(30,81): error TS2339: Property 'title' does not exist on type 'never'.
```

**Issue:** TypeScript infers `item` as `never` type, likely due to incorrect ref initialization.

**Root Cause:** `const item = ref(null);` should be `const item = ref<StoreItem | null>(null);`

**Fix Strategy:**
```typescript
// Current (incorrect):
const item = ref(null);

// Fixed:
const item = ref<StoreItem | null>(null);
```

**Priority:** High (affects store item display)

---

### 5. ApplicationDetail.vue (4 errors)

**Category 1: Property Naming**
```
src/components/ApplicationDetail.vue(15,100): error TS2551: Property 'proposedFee' does not exist on type 'Application'. Did you mean 'proposed_fee'?
src/components/ApplicationDetail.vue(159,113): error TS2551: Property 'applicantId' does not exist on type 'Application'. Did you mean 'applicant_id'?
```

**Issue:** Template uses camelCase but interface defines snake_case.

**Fix Strategy:**
- Use snake_case in template: `application.proposed_fee`, `application.applicant_id`

**Count:** 2 errors

---

**Category 2: Message Type Mismatch**
```
src/components/ApplicationDetail.vue(202,5): error TS2322: Type 'Message[]' is not assignable to type 'Message[] | { id: number; senderId: number; ... }[]'.
```

**Issue:** Two different Message type definitions exist.

**Fix Strategy:**
- Use single Message type from `@/stores/messages`
- Remove duplicate Message interface

**Count:** 1 error

**Priority:** Medium

---

### 6. ChatWidget.vue (6 errors)

**Category 1: Event Signature Mismatches**
```
src/components/messages/ChatWidget.vue(79,16): error TS2322: Type '(messageId: number, content: string) => void' is not assignable to type '(content: string) => any'.
src/components/messages/ChatWidget.vue(80,16): error TS2322: Type '(messageId: number) => void' is not assignable to type '() => any'.
```

**Issue:** MessageBubble component's event signatures don't match ChatWidget's handler functions.

**Fix Strategy:**
- Update MessageBubble events to match expected signatures, OR
- Create wrapper functions in ChatWidget

**Count:** 2 errors

---

**Category 2: NodeJS Namespace**
```
src/components/messages/ChatWidget.vue(130,27): error TS2503: Cannot find namespace 'NodeJS'.
```

**Issue:** `typingTimeout` ref uses `NodeJS.Timeout` but NodeJS types not imported.

**Fix Strategy:**
```typescript
// Add to top of file:
/// <reference types="node" />

// OR use browser-compatible type:
const typingTimeout = ref<ReturnType<typeof setTimeout>>();
```

**Count:** 1 error

---

**Category 3: Undefined Safety**
```
src/components/messages/ChatWidget.vue(138,32): error TS2345: Argument of type 'number | undefined' is not assignable to parameter of type 'number'.
```

**Issue:** Conversation IDs can be undefined but functions expect number.

**Fix Strategy:**
- Add null checks before function calls
- Use non-null assertion if guaranteed: `conversationId!`

**Count:** 3 errors

**Priority:** Low (widget functions correctly despite errors)

---

### 7. MessageThread.vue (1 error)

**Error:** NodeJS namespace not found
```
src/components/messages/MessageThread.vue(104,27): error TS2503: Cannot find namespace 'NodeJS'.
```

**Issue:** Same as ChatWidget - `typingTimeout` ref type.

**Fix Strategy:**
```typescript
const typingTimeout = ref<ReturnType<typeof setTimeout>>();
```

**Priority:** Low

---

### 8. ProfileView.vue (4 errors)

**Error:** Review type mismatches
```
src/views/profile/ProfileView.vue(112,10): error TS2322: Type '{ id: number; reviewer: {...}; rating: number; ... }[]' is not assignable to type 'Review[]'.
```

**Issue:** API returns review objects with different structure than Review interface.

**Fix Strategy:**
- Map API response to Review interface
- Create separate ApiReview type for raw API data

**Priority:** Medium

---

### 9. UserProfileView.vue (1 error)

**Error:** User type mismatch
```
src/views/profile/UserProfileView.vue(149,5): error TS2322: Type 'User' is not assignable to type 'User | { id: number; ... } | null'.
```

**Issue:** Multiple User type definitions across codebase.

**Fix Strategy:**
- Consolidate to single User type definition
- Export from centralized location (e.g., `@/types/user.ts`)

**Priority:** Low

---

### 10. tasks.ts Store (1 error)

**Error:** Implicit any type
```
src/stores/tasks.ts(263,56): error TS7006: Parameter 'task' implicitly has an 'any' type.
```

**Issue:** Function parameter missing type annotation.

**Fix Strategy:**
```typescript
// Add explicit type:
.filter((task: Task) => ...)
```

**Priority:** Low

---

## Root Cause Analysis

### Primary Issues:

1. **Inconsistent Property Naming** (20+ errors)
   - Backend API uses snake_case (e.g., `created_by`)
   - Some TypeScript interfaces use camelCase (e.g., `createdBy`)
   - Templates mix both styles

2. **Missing Null Safety** (15+ errors)
   - Refs initialized as `ref(null)` without generic type
   - Code accesses properties without null checks
   - Template uses values before loading completes

3. **Duplicate Type Definitions** (5+ errors)
   - Multiple Message interfaces
   - Multiple User interfaces
   - Types not centralized

4. **Missing Type Annotations** (10+ errors)
   - Implicit 'any' types
   - Missing generic parameters on refs
   - NodeJS namespace not imported

---

## Recommended Fix Strategy

### Phase 1: Quick Wins (Low Effort, High Impact)
**Estimated Time:** 2-4 hours

1. **Add Type Annotations to Refs**
   ```typescript
   // Current:
   const item = ref(null);

   // Fixed:
   const item = ref<StoreItem | null>(null);
   ```
   - Files: `StoreItemView.vue`, `TaskDetailView.vue`
   - Impact: Fixes ~15 errors

2. **Fix NodeJS Namespace**
   ```typescript
   const typingTimeout = ref<ReturnType<typeof setTimeout>>();
   ```
   - Files: `ChatWidget.vue`, `MessageThread.vue`
   - Impact: Fixes 2 errors

3. **Add isAuthenticated to Auth Store**
   ```typescript
   const isAuthenticated = computed(() => user.value !== null);
   ```
   - Files: `stores/auth.ts`, `App.vue`
   - Impact: Fixes 1 error

---

### Phase 2: Property Naming Standardization (Medium Effort)
**Estimated Time:** 4-6 hours

1. **Standardize on snake_case** (matches backend API)
   - Update all templates to use snake_case
   - Update TypeScript interfaces to match API
   - Files: `TaskDetailView.vue`, `ApplicationDetail.vue`
   - Impact: Fixes ~10 errors

2. **Create API Response Mappers** (alternative approach)
   - Keep camelCase in frontend
   - Map API responses: `created_by` → `createdBy`
   - More work but better developer experience
   - Impact: Fixes ~10 errors

**Recommendation:** Use snake_case throughout for consistency with backend. Frontend-only camelCase adds unnecessary complexity.

---

### Phase 3: Type Consolidation (Higher Effort)
**Estimated Time:** 3-5 hours

1. **Centralize Type Definitions**
   ```
   src/types/
     ├── user.ts       # Single User type
     ├── message.ts    # Single Message type
     ├── task.ts       # Single Task type
     └── review.ts     # Single Review type
   ```

2. **Remove Duplicate Interfaces**
   - Search for duplicate type definitions
   - Update all imports to use centralized types
   - Impact: Fixes ~8 errors

3. **Add Null Safety Checks**
   - Add `v-if` guards in templates
   - Use optional chaining in computed properties
   - Impact: Fixes ~12 errors

---

### Phase 4: Strict Type Checking (Optional - Future Enhancement)
**Estimated Time:** 8-12 hours

1. **Enable Stricter TypeScript Checks**
   ```json
   // tsconfig.json
   {
     "compilerOptions": {
       "strict": true,
       "noImplicitAny": true,
       "strictNullChecks": true,
       "strictPropertyInitialization": true
     }
   }
   ```

2. **Fix All Resulting Errors**
   - This will likely reveal additional type issues
   - But results in much better type safety

---

## Prioritized Action Plan

### Before MVP Launch (P1 - Critical)
- [ ] Fix `App.vue` isAuthenticated error (blocks authentication checks)
- [ ] Fix `StoreItemView.vue` type narrowing (blocks store functionality)
- [ ] Fix `TaskDetailView.vue` null safety (high error count, core feature)

**Estimated Time:** 4-6 hours
**Impact:** Resolves ~40 errors, fixes critical user flows

---

### Before Production Launch (P2 - Important)
- [ ] Standardize property naming (snake_case vs camelCase)
- [ ] Consolidate type definitions
- [ ] Fix all remaining null safety issues
- [ ] Add proper type annotations to all refs

**Estimated Time:** 10-15 hours
**Impact:** Resolves all remaining errors, improves maintainability

---

### Future Enhancement (P3 - Nice to Have)
- [ ] Enable strict TypeScript mode
- [ ] Add comprehensive type tests
- [ ] Document type patterns in CONTRIBUTING.md

**Estimated Time:** 8-12 hours
**Impact:** Prevents future type errors, improves developer experience

---

## Testing Strategy

After fixing type errors:

1. **Type Check**
   ```bash
   npm run type-check
   # Should show 0 errors
   ```

2. **Build Verification**
   ```bash
   npm run build
   # Verify no regression
   ```

3. **Runtime Testing**
   - Test all affected pages manually
   - Verify no runtime TypeScript errors in console
   - Run existing unit tests

4. **Regression Prevention**
   - Add pre-commit hook for type checking
   - Configure CI to fail on type errors
   - Document type patterns for team

---

## References

- **Current Build Output:** All 62 errors documented above
- **TypeScript Handbook:** https://www.typescriptlang.org/docs/handbook/
- **Vue 3 TypeScript Guide:** https://vuejs.org/guide/typescript/overview.html
- **Pinia TypeScript:** https://pinia.vuejs.org/core-concepts/#typescript

---

## Progress Tracking

**Status:** 📋 Not Started
**Last Updated:** January 4, 2026
**Responsible:** TBD

### Completion Checklist:
- [ ] Phase 1: Quick Wins (2-4 hours)
- [ ] Phase 2: Property Naming (4-6 hours)
- [ ] Phase 3: Type Consolidation (3-5 hours)
- [ ] All type errors resolved
- [ ] Documentation updated
- [ ] CI configured to prevent regression

---

**Document Version:** 1.0
**Created:** January 4, 2026
**Purpose:** Track and plan resolution of pre-existing TypeScript type errors
