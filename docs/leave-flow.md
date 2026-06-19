# Leave Flow & Action System

## Overview

The leave system uses a **multi-stage, multi-role approval workflow** driven by a per-leave
JSON log (`Tbl_Leave_Flow`). Each leave type can have a configurable approval chain. When a
leave is applied, the chain is snapshotted into the log. Approvers then act on their own
stage — approve, reject, or withdraw — without touching any other approver's stage.

---

## Table of Contents

1. [Database Tables](#1-database-tables)
2. [Data Models](#2-data-models)
3. [Leave Statuses](#3-leave-statuses)
4. [Stage States](#4-stage-states)
5. [Approval Flow — How It Is Built](#5-approval-flow--how-it-is-built)
6. [Leave Action — Full Flow](#6-leave-action--full-flow)
7. [Action: APPROVE](#7-action-approve)
8. [Action: REJECT](#8-action-reject)
9. [Action: WITHDRAW](#9-action-withdraw)
10. [Get Leaves — Approval Log Enrichment](#10-get-leaves--approval-log-enrichment)
11. [Architecture Layers](#11-architecture-layers)
12. [SOLID Design Decisions](#12-solid-design-decisions)

---

## 1. Database Tables

### `Tbl_Leave`
Stores the leave application. `status` reflects the overall lifecycle state.

| Column        | Type      | Notes                                      |
|---------------|-----------|--------------------------------------------|
| `id`          | UUID PK   |                                            |
| `employee_id` | UUID FK   | → `Tbl_Employee`                           |
| `leave_type_id` | INT FK  | → `Tbl_Leave_type`                         |
| `status`      | TEXT      | `Pending` / `APPROVED` / `REJECTED` / `WITHDRAWAL_PENDING` / `WITHDRAWN` |
| `approved_by` | UUID FK   | Last approver who acted                    |
| `days`        | NUMERIC   | Calculated working days                    |

### `Tbl_Leave_Flow`
One row per leave. Stores the entire approval chain as a JSONB array.

| Column         | Type    | Notes                                         |
|----------------|---------|-----------------------------------------------|
| `id`           | UUID PK |                                               |
| `leave_id`     | UUID FK | → `Tbl_Leave`                                 |
| `approval_log` | JSONB   | `[]LeaveFlowStage` — the full per-stage audit |
| `created_at`   | TIMESTAMP |                                             |
| `updated_at`   | TIMESTAMP |                                             |
| `deleted_at`   | TIMESTAMP | Soft delete                                 |

> **No separate FK column for approver ID** — each stage inside `approval_log` JSON
> carries its own `approved_by` UUID. Names are resolved at read time via a JOIN to
> `Tbl_Employee` — not stored in the JSON column.

### `Tbl_Leave_Approval_Flow`
Template flows configured per leave type.

| Column  | Type    | Notes                          |
|---------|---------|--------------------------------|
| `id`    | UUID PK |                                |
| `name`  | TEXT    | e.g. "Standard", "WFH Flow"   |
| `flow`  | JSONB   | `[]ApprovalStage` — ordered stages with roles |

---

## 2. Data Models

### `ApprovalStage` (template)
```go
type ApprovalStage struct {
    StageNo      int          // 1, 2, 3 …
    ApproverRole ApproverRole // MANAGER | HR | ADMIN | SUPERADMIN
}
```
Multiple entries can share the same `StageNo` — any ONE of those roles approving
settles that stage level.

### `LeaveFlowStage` (live log entry)
```go
type LeaveFlowStage struct {
    StageNo        int
    ApproverRole   ApproverRole
    State          State       // WAITING | APPROVED | REJECTED | WITHDRAWN | SKIPPED
    ApprovedBy     *uuid.UUID  // stored in DB JSON
    ApprovedByName *string     // resolved at read time via JOIN — NOT stored in DB
    Remarks        string
    ActionAt       *time.Time
}
```

### `LeaveFlow` (in-memory / API response)
```go
type LeaveFlow struct {
    ID          uuid.UUID
    LeaveID     uuid.UUID
    ApprovalLog []LeaveFlowStage
    CreatedAt   *time.Time
    UpdatedAt   *time.Time
    DeletedAt   *time.Time
}
```

### `ActionLeaveReq` (HTTP request body)
```go
type ActionLeaveReq struct {
    Action  string // "APPROVE" | "REJECT" | "WITHDRAW"
    Remarks string // optional note
}
```

---

## 3. Leave Statuses

These are the valid values for `Tbl_Leave.status`:

| Status               | Meaning                                                      |
|----------------------|--------------------------------------------------------------|
| `Pending`            | Applied, awaiting first approver                             |
| `APPROVED`           | All approval stages settled — leave is fully approved        |
| `REJECTED`           | Rejected at any stage — immediately final                    |
| `WITHDRAWAL_PENDING` | Partially withdrawn — lower stage withdrew, higher stage has not yet |
| `WITHDRAWN`          | Fully withdrawn by all approvers — balance restored          |

---

## 4. Stage States

Per-stage state inside the `approval_log` JSON:

| State       | Meaning                                                                 |
|-------------|-------------------------------------------------------------------------|
| `WAITING`   | This role has not yet acted                                             |
| `APPROVED`  | This role approved their stage                                          |
| `REJECTED`  | This role rejected (leave status becomes REJECTED immediately)          |
| `SKIPPED`   | Auto-skipped — a sibling at the same `stage_no` already acted, or a lower stage was bypassed |
| `WITHDRAWN` | This role withdrew their approval                                       |

---

## 5. Approval Flow — How It Is Built

**At apply time** (`LeaveFlowService.Create`):

1. The leave type's linked `LeaveApprovalFlow` template is fetched.
2. Stages where the approver role level ≤ the applicant's role level are **filtered out**
   (you cannot be approved by someone at or below your own level).
3. The remaining stages are inserted as `WAITING` entries into `Tbl_Leave_Flow`.

**Example** — template has `[{1,MANAGER},{1,HR},{2,SUPERADMIN}]`, applicant is EMPLOYEE:

```json
approval_log: [
  { "stage_no": 1, "approver_role": "MANAGER",    "state": "WAITING" },
  { "stage_no": 1, "approver_role": "HR",         "state": "WAITING" },
  { "stage_no": 2, "approver_role": "SUPERADMIN", "state": "WAITING" }
]
```

If the applicant is a MANAGER, stage 1 entries are filtered out — only SUPERADMIN remains.

---

## 6. Leave Action — Full Flow

**Endpoint:** `POST /api/leaves/:id/action`  
**Controller:** `controllers.LeaveAction`  
**Service:** `LeaveFlowService.ActionLeave`

```
Request: { "action": "APPROVE", "remarks": "looks good" }

1. GetByID(leaveID)           → fetch Tbl_Leave
2. Status guard               → APPROVE/REJECT only on "Pending"
                                 WITHDRAW only on "APPROVED" or "WITHDRAWAL_PENDING"
3. Self-action guard          → approver cannot act on their own leave
4. GetByLeaveID(leaveID)      → fetch Tbl_Leave_Flow + unmarshal JSON
                                 + enrich ApprovedByName via JOIN
5. ActionValidator            → check role has a stage in the log
                                 check stage.State is valid for the action
6. LeavePolicyRepo.GetById    → fetch LeaveType (needed for IsEarly / balance check)
7. registry.Resolve(action)   → get the right processor (no switch)
8. Build LeaveActionContext    → bundle all data + repos into one object
9. ExecuteTransaction
     └── processor.Process(ctx, tx, lctx)
```

---

## 7. Action: APPROVE

**File:** `service/leave/leaveprocess/approved.go`

**Rules:**
- Stage must be `WAITING`
- All stages with a lower `stage_no` must be `APPROVED` or `SKIPPED` (settled)
- One role approving at a `stage_no` auto-SKIPs all sibling WAITING roles at the same level

**Steps inside the transaction:**

```
1. findStage(flow, role)                 → locate caller's stage
2. previousStagesSettled(log, stageNo)   → enforce ordered processing
3. stampStage → APPROVED                 → set ApprovedBy, ActionAt, Remarks
4. skipStages(log, stageNo, role)        → SKIP siblings at same stage + lower WAITING stages
5. UpdateApprovalLog(tx, leaveID, log)   → persist JSON back to Tbl_Leave_Flow
6. allStagesSettled(log)?
   → NO  : leave stays "Pending" — return (next stage waits)
   → YES : UpdateLeaveStatus → "APPROVED"
           if not IsEarly → DeductBalance (used+, closing-)
```

**Example progression:**

```
Initial:
  stage 1: MANAGER=WAITING, HR=WAITING
  stage 2: SUPERADMIN=WAITING

HR approves:
  stage 1: MANAGER=SKIPPED, HR=APPROVED
  stage 2: SUPERADMIN=WAITING  ← still waiting → leave stays Pending

SUPERADMIN approves:
  stage 1: MANAGER=SKIPPED, HR=APPROVED
  stage 2: SUPERADMIN=APPROVED
  → allStagesSettled = true → leave → APPROVED + balance deducted
```

---

## 8. Action: REJECT

**File:** `service/leave/leaveprocess/reject.go`

**Rules:**
- Stage must be `WAITING`
- **Single final action — no ordering constraint.** Any role in the flow can reject
  at any time as long as their stage is still WAITING.
- Rejection is immediately final — no further stages run.

**Steps inside the transaction:**

```
1. findStage(flow, role)                   → locate caller's stage
2. stampStage → REJECTED                   → set ApprovedBy, ActionAt, Remarks
3. skipAllWaitingStages(log, stageNo, role)→ SKIP all remaining WAITING stages
4. UpdateApprovalLog(tx, leaveID, log)     → persist JSON
5. UpdateLeaveStatus → "REJECTED"          → immediately final, no balance touched
```

**Key difference from APPROVE:** No `previousStagesSettled` check — a MANAGER at stage 1
and a SUPERADMIN at stage 2 can both reject independently whenever their stage is WAITING.

---

## 9. Action: WITHDRAW

**File:** `service/leave/leaveprocess/withdraw.go`

**Rules — mirrors APPROVE exactly:**
- Stage must be `APPROVED` (the caller was the one who approved it)
- Leave must be `APPROVED` or `WITHDRAWAL_PENDING`
- Ordered by stage — lower stages must withdraw before higher stages
- One role withdrawing at a `stage_no` SKIPs sibling APPROVED stages at the same level
- **Balance is only restored by the final (highest `stage_no`) approver**

**Steps inside the transaction:**

```
1. findStage(flow, role)                           → locate caller's stage
2. previousStagesSettledForWithdraw(log, stageNo)  → lower stages must be WITHDRAWN/SKIPPED first
3. stampStage → WITHDRAWN                          → set ApprovedBy, ActionAt, Remarks
4. skipSiblingsForWithdraw(log, stageNo, role)     → SKIP sibling APPROVED at same stage
5. UpdateApprovalLog(tx, leaveID, log)             → persist JSON
6. allStagesSettledForWithdraw(log)?
   → NO  : UpdateLeaveStatus → "WITHDRAWAL_PENDING" (higher approver must still withdraw)
   → YES : UpdateLeaveStatus → "WITHDRAWN"
           if isFinalWithdrawStage AND not IsEarly → RestoreBalance (used-, closing+)
```

**Example progression (2-stage flow):**

```
Initial (fully approved):
  stage 1: MANAGER=APPROVED
  stage 2: SUPERADMIN=APPROVED
  leave: APPROVED

MANAGER withdraws:
  stage 1: MANAGER=WITHDRAWN
  stage 2: SUPERADMIN=APPROVED  ← not settled → WITHDRAWAL_PENDING, no balance restore

SUPERADMIN withdraws:
  stage 1: MANAGER=WITHDRAWN
  stage 2: SUPERADMIN=WITHDRAWN
  → allSettled = true, isFinalWithdrawStage(stage 2) = true
  → leave → WITHDRAWN + balance restored
```

---

## 10. Get Leaves — Approval Log Enrichment

**Endpoint:** `GET /api/leaves?month=6&year=2026`  
**Controller:** `controllers.GetLeaves`  
**Service:** `LeaveFlowService.GetLeaves`

Role-based data scoping:

| Role                        | Data returned                         |
|-----------------------------|---------------------------------------|
| `EMPLOYEE`, `INTERN`        | Own leaves only                       |
| `MANAGER`                   | Leaves of direct reports              |
| `ADMIN`, `HR`, `SUPERADMIN` | All leaves                            |

For each leave in the result, `GetByLeaveID` is called to attach the `approval_log`.

### Approver Name Resolution (no schema change)

`ApprovedByName` is **not stored** in the `approval_log` JSON column. It is resolved at
read time:

```
GetByLeaveID(leaveID):
  1. SELECT * FROM Tbl_Leave_Flow WHERE leave_id = $1
  2. json.Unmarshal(approval_log) → []LeaveFlowStage
  3. collect all non-nil ApprovedBy UUIDs
  4. SELECT id, full_name FROM Tbl_Employee WHERE id IN ($1, $2, ...)
     → one query, returns map[uuid]string
  5. enrich each stage: stage.ApprovedByName = nameMap[stage.ApprovedBy]
  6. return enriched LeaveFlow
```

This keeps `Tbl_Leave_Flow` schema stable — no FK column added — while providing
the name in every API response.

---

## 11. Architecture Layers

```
HTTP Request
    │
    ▼
controllers/leaveFlow.go          ← thin: extract IDs, call service, respond
    │
    ▼
service/leave/leaveflow/
    leaveflow.go                  ← orchestration: fetch, validate, build context, run tx
    │
    ├── service/leave_flow_log.go ← LeaveFlowLog: create/fetch/enrich approval log
    │       └── repositories/leaveFlowLog.go  ← raw DB: CRUD + GetApproverNames JOIN
    │
    ├── repositories/leaveflow.go ← raw DB: InsertLeave, GetByID, GetLeaves queries
    │
    └── service/leave/leaveprocess/
            process.go            ← LeaveActionContext, ProcessorRegistry, helpers
            approved.go           ← ApproveProcessor.Process()
            reject.go             ← RejectProcessor.Process()
            withdraw.go           ← WithdrawProcessor.Process()
```

---

## 12. SOLID Design Decisions

### Single Responsibility
- Each processor file handles exactly one action type.
- `leaveflow.go` only orchestrates — it fetches data and delegates to processors.
- `leave_flow_log.go` only manages the approval log lifecycle (create, read, update, enrich).

### Open/Closed Principle
- `ProcessorRegistry` maps action string → processor.
- Adding a new action (e.g. `ESCALATE`) means only adding a new processor file and one
  `r.Register("ESCALATE", &EscalateProcessor{})` line — no existing code changes.

### Liskov Substitution
- All processors implement `LeaveActionProcessor` — they are interchangeable via the registry.

### Interface Segregation
- `repositories.LeaveFlowLog` — only log-related DB operations.
- `repositories.LeavePolicyRepository` — only leave type operations.
- `service.LeaveFlowLog` — only approval log service operations.

### Dependency Inversion
- `LeaveActionContext` injects `FlowLogRepo` and `CommRepo` as interfaces/concrete deps.
- Processors never import `leaveflow` — dependency goes one way only.

### `LeaveActionContext` — why a context object
The `Process` interface signature is `Process(ctx, tx, *LeaveActionContext)`.
Adding new fields (e.g. IP address for audit, notification service) only requires adding
to the struct — the interface and all existing processors remain untouched.
