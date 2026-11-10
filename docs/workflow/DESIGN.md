# Workflow Engine (BPM) — Design

Status: draft, pending review. No code written yet.

Scope: a brand-new, self-contained feature ("workflow engine") that lets users define
business processes (leave request, raise request, equipment request, purchase request...)
and run many parallel instances, each with its own state + data. It only reuses shared
infrastructure (auth/session, user/workspace/role, DB connection, logging, notification,
file storage, design system UI) and does not modify any existing feature.

Repos involved:
- Backend: `smallbites` (Go) — this document lives at `smallbites/docs/workflow/DESIGN.md`.
- Frontend: `smallbitesapp` (Flutter) — `docs/workflow/` doesn't exist there; frontend
  design decisions are described here too, code will land under `lib/ui/screens/workflow*`.

## 0. Decisions locked in during Phase 0 review

These override the generic task template where they conflict, per explicit user answers:

1. **No automated test suite required** for this module (unlike the usual "test dày" bar).
   The engine will still be written as pure, dependency-injected Go (no HTTP coupling) so
   it *could* be tested later, but writing the tests themselves is out of scope for now.
2. **Permissions reuse the existing workspace role model** (`workspace_role`: `holder` |
   `admin` | `member`, defined in `db/migrations/20260503052445_enums.sql`). No new
   permission-key table (`workflow.admin` etc. become role checks, see §2).
3. **Org hierarchy for "manager" resolution**: there is no `manager_id`/department column
   anywhere in the schema (confirmed by grep — not present). Per user: a member's manager
   is "the admins" of the workspace; an admin's manager is "the holder" of the workspace;
   there is no 1:1 "direct manager" relationship. This is modeled as a **role-tier chain**:
   `member → admin → holder`, holder has no manager above. See §1.3.
4. **Canvas/diagram library for the process designer**: decided at Phase 3 — hand-built with
   `InteractiveViewer` + `CustomPainter` + `GestureDetector` (pan/zoom, free node dragging, a
   dedicated drag-to-connect handle per node using `RenderBox.globalToLocal` for canvas-space
   coordinates independent of the current pan/zoom transform). No pub.dev dependency added.
5. **No i18n.** Strings are hardcoded (matching the rest of the app's current convention),
   no `.arb`/translation layer introduced for this module.
6. **wf_job worker**: implemented as a `time.NewTicker` polling goroutine, same pattern as
   `AttendanceReminderService` / task deadline reminder — not backed by watermill (which is
   in-memory/non-durable) or Redis (present in docker-compose but unused by any Go code
   today, not introduced here either).
7. **Frontend module split**: confirmed as proposed in the Phase 0 report — see §6.
8. **Expression/condition engine**: no existing Go expression library in the repo, and the
   task requires no `eval`/arbitrary code execution and no new dependency without asking.
   Decision (best judgment, no library added): a small hand-written JSON-AST tree-walking
   interpreter, pure Go, with a node/depth budget and a context deadline. See §1.4.

## 1.1 Data model

All tables use the project's existing conventions (confirmed from `attendance_correction_requests`
and friends): `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`, `created_at`/`updated_at`
`TIMESTAMPTZ NOT NULL DEFAULT NOW()` + the shared `update_updated_at_column()` trigger,
status columns as `TEXT NOT NULL CHECK (... IN (...))` (not a new Postgres ENUM per table —
matches the more recent/common style in the codebase; `workspace_role` is the one exception
because it's a cross-cutting FK-typed column). Every definition/instance is scoped to a
`workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE`, matching every
other feature in the codebase.

### wf_definition
| Column | Type | Notes |
|---|---|---|
| id | UUID PK | |
| workspace_id | UUID NOT NULL FK workspaces(id) ON DELETE CASCADE | |
| key | VARCHAR(255) NOT NULL | unique within workspace |
| name | VARCHAR(255) NOT NULL | |
| description | TEXT | |
| category | VARCHAR(100) | |
| icon | VARCHAR(100) | Solar icon name, per FE icon convention |
| owner_id | UUID NOT NULL FK users(id) | creator/owner |
| status | TEXT NOT NULL DEFAULT 'active' CHECK IN ('active','disabled') | on/off switch, independent of version lifecycle |
| created_at / updated_at | | |

`UNIQUE (workspace_id, key)`. Index: `(workspace_id, status)`.

### wf_definition_version
| Column | Type | Notes |
|---|---|---|
| id | UUID PK | |
| definition_id | UUID NOT NULL FK wf_definition(id) ON DELETE CASCADE | |
| version | INT NOT NULL | increasing per definition, starts at 1 |
| graph | JSONB NOT NULL | nodes + edges, see §1.2 |
| form_schema | JSONB NOT NULL DEFAULT '{}' | see §1.5 |
| status | TEXT NOT NULL DEFAULT 'draft' CHECK IN ('draft','published','archived') | |
| published_at | TIMESTAMPTZ | |
| published_by | UUID FK users(id) | |
| created_at / updated_at | | |

`UNIQUE (definition_id, version)`. Index: `(definition_id, status)`.

**Invariant** (enforced in the service layer, not a DB trigger — consistent with how the
rest of the codebase enforces invariants imperatively): once `status = 'published'`, `graph`
and `form_schema` become immutable. Editing means cloning into a new `draft` version with
`version = max(version)+1`.

### wf_instance
| Column | Type | Notes |
|---|---|---|
| id | UUID PK | |
| definition_version_id | UUID NOT NULL FK wf_definition_version(id) | pinned at start, never changes — satisfies "running instance keeps the version it started with" |
| workspace_id | UUID NOT NULL FK workspaces(id) ON DELETE CASCADE | denormalized from definition for indexing |
| business_key | VARCHAR(255) | optional external correlation key |
| initiator_id | UUID NOT NULL FK users(id) | |
| status | TEXT NOT NULL DEFAULT 'running' CHECK IN ('running','waiting','completed','canceled','rejected','error') | |
| variables | JSONB NOT NULL DEFAULT '{}' | process variables incl. initial form submission |
| error_message | TEXT | populated when status = 'error' |
| step_count | INT NOT NULL DEFAULT 0 | infinite-loop guard, incremented every `advance()`, capped (see §1.6) |
| started_at | TIMESTAMPTZ NOT NULL DEFAULT NOW() | |
| ended_at | TIMESTAMPTZ | |
| version | INT NOT NULL DEFAULT 1 | optimistic locking (`WHERE id=? AND version=?`) |
| created_at / updated_at | | |

Indexes: `(workspace_id, status)`, `(initiator_id)`, `(definition_version_id)`.

### wf_token
Supports parallel branches (fork/join); each token points at the node it currently occupies.

| Column | Type | Notes |
|---|---|---|
| id | UUID PK | |
| instance_id | UUID NOT NULL FK wf_instance(id) ON DELETE CASCADE | |
| parent_token_id | UUID FK wf_token(id) | set for tokens spawned by a `parallelGateway` fork, used to join |
| node_id | VARCHAR(255) NOT NULL | current node key (matches a node id in `graph`) |
| status | TEXT NOT NULL DEFAULT 'active' CHECK IN ('active','waiting','completed','canceled') | `waiting` = blocked on a task/timer/join |
| version | INT NOT NULL DEFAULT 1 | optimistic locking per token |
| created_at / updated_at | | |

Indexes: `(instance_id, status)`, `(parent_token_id)`.

### wf_task
One row per *actionable item* for one candidate. For a multi-approver `approval` node
(any/all/quorum), the engine creates **one `wf_task` row per candidate approver**, all
sharing the same `(token_id, node_id)` — that pair is the "decision group" the engine tallies
against `decision_mode` (no separate grouping column needed since a token stays `waiting` at
that node until the group resolves).

| Column | Type | Notes |
|---|---|---|
| id | UUID PK | |
| instance_id | UUID NOT NULL FK wf_instance(id) ON DELETE CASCADE | |
| token_id | UUID NOT NULL FK wf_token(id) ON DELETE CASCADE | decision-group key together with node_id |
| node_id | VARCHAR(255) NOT NULL | |
| name | VARCHAR(255) NOT NULL | |
| type | TEXT NOT NULL CHECK IN ('userTask','form','approval') | `form` added post-review (FE feedback): a distinct node type from `userTask`, dedicated to collecting a scoped subset of the version's form_schema fields (via `fieldKeys` in its config) — identical to `userTask` at the engine level (see §1.2), same "any" tally |
| assignee_id | UUID FK users(id) | resolved single candidate for this row |
| decision_mode | TEXT CHECK IN ('any','all','quorum') | only set for `approval` |
| quorum_percent | INT | only set when decision_mode='quorum' |
| status | TEXT NOT NULL DEFAULT 'pending' CHECK IN ('pending','claimed','approved','rejected','request-change','reassigned','canceled','expired') | `request-change` added post-review to match the task brief's approve/reject/request-change trio |
| due_at | TIMESTAMPTZ | |
| form_data | JSONB NOT NULL DEFAULT '{}' | |
| comment | TEXT | required on reject, per FE spec |
| decided_at | TIMESTAMPTZ | |
| decided_by | UUID FK users(id) | |
| delegated_from | UUID FK users(id) | set when reassigned/delegated |
| created_at / updated_at | | |

Indexes: `(assignee_id, status)`, `(instance_id)`, `(due_at) WHERE status = 'pending'`.

### wf_history
Append-only. No `updated_at` (rows are never mutated).

| Column | Type | Notes |
|---|---|---|
| id | UUID PK | |
| instance_id | UUID NOT NULL FK wf_instance(id) ON DELETE CASCADE | |
| token_id | UUID FK wf_token(id) | |
| task_id | UUID FK wf_task(id) | |
| event_type | TEXT NOT NULL | `node_entered`, `node_exited`, `task_created`, `task_decided`, `variable_changed`, `error`, `instance_started`, `instance_ended` |
| node_id | VARCHAR(255) | |
| actor_id | UUID FK users(id) | null for system/automatic transitions |
| data_before | JSONB | |
| data_after | JSONB | |
| message | TEXT | |
| created_at | TIMESTAMPTZ NOT NULL DEFAULT NOW() | |

Index: `(instance_id, created_at)`. This is what drives the FE timeline view.

### wf_comment
| Column | Type |
|---|---|
| id | UUID PK |
| instance_id | UUID NOT NULL FK wf_instance(id) ON DELETE CASCADE |
| author_id | UUID NOT NULL FK users(id) |
| content | TEXT NOT NULL |
| created_at / updated_at | |

Index: `(instance_id, created_at)`.

### wf_attachment
| Column | Type | Notes |
|---|---|---|
| id | UUID PK | |
| instance_id | UUID NOT NULL FK wf_instance(id) ON DELETE CASCADE | |
| task_id | UUID FK wf_task(id) | optional, if attached at a specific task/form step |
| uploader_id | UUID NOT NULL FK users(id) | |
| file_name | VARCHAR(255) NOT NULL | |
| file_url | VARCHAR(2048) NOT NULL | MinIO object URL, reusing the existing presigned-upload flow (`pkg/minio`, `POST /v1/uploads/presigned/url`) |
| content_type | VARCHAR(100) | |
| size_bytes | BIGINT | |
| created_at | TIMESTAMPTZ NOT NULL DEFAULT NOW() | |

Index: `(instance_id)`.

### wf_job
Backs timer / retry / escalation, drained by a polling worker (§3).

| Column | Type | Notes |
|---|---|---|
| id | UUID PK | |
| instance_id | UUID NOT NULL FK wf_instance(id) ON DELETE CASCADE | |
| token_id | UUID FK wf_token(id) | |
| node_id | VARCHAR(255) | |
| job_type | TEXT NOT NULL CHECK IN ('timer','retry','escalation') | |
| payload | JSONB NOT NULL DEFAULT '{}' | |
| scheduled_at | TIMESTAMPTZ NOT NULL | |
| attempts | INT NOT NULL DEFAULT 0 | |
| max_attempts | INT NOT NULL DEFAULT 5 | |
| status | TEXT NOT NULL DEFAULT 'pending' CHECK IN ('pending','processing','done','failed','dead_letter') | |
| last_error | TEXT | |
| created_at / updated_at | | |

Index: `(status, scheduled_at)` for the worker's claim query. Claim query design (sqlc):
`UPDATE wf_job SET status='processing' WHERE id IN (SELECT id FROM wf_job WHERE status='pending' AND scheduled_at <= now() ORDER BY scheduled_at LIMIT $1 FOR UPDATE SKIP LOCKED) RETURNING *` —
`FOR UPDATE SKIP LOCKED` is a native Postgres feature (no new dependency) and gives safe
concurrent claiming if the app ever runs >1 replica.

## 1.2 Node types

| Node | Engine behavior |
|---|---|
| `start` | Instance creation only; carries the initial form schema reference (points at `wf_definition_version.form_schema`). |
| `userTask` | Resolves a candidate set (§1.3) → creates `wf_task` row(s) with `type='userTask'`; token goes `waiting`; advances on `completeTask()`. |
| `form` | Added post-review (FE feedback: a distinct block was wanted for "gather information", separate from `userTask`). Identical to `userTask` at the engine level — same candidate resolution, same "any" tally — but its config carries `fieldKeys: []string` scoping which of the version's `form_schema` fields the FE shows/collects at this specific step (a rendering hint; not server-enforced in this pass). |
| `approval` | Same as `userTask` but `decision_mode` in {`any`,`all`,`quorum`}; creates one `wf_task` per resolved candidate sharing `(token_id, node_id)`; each decision is tallied — `any`: first approve advances, first reject rejects; `all`: advances only once every row is `approved`, any `rejected` rejects immediately; `quorum`: advances once `approved / total >= quorum_percent`, rejects if reject-count makes quorum unreachable. |
| `exclusiveGateway` | Evaluates each outgoing edge's condition (§1.4) in declared order, takes the first true one; a `default` edge (no condition) is required and used if none match. |
| `parallelGateway` | **Fork**: current token completes, N new tokens created (one per outgoing edge) with `parent_token_id` set. **Join**: a node reached by >1 incoming edge blocks until all sibling tokens (same `parent_token_id`) reach it, then they complete and a single new token continues. |
| `serviceTask` | Automatic step: either an HTTP webhook call, a variable mutation, or an internal service call, declared in node config; runs inside `advance()`; failure schedules a `wf_job(job_type='retry')` with backoff (§1.6) instead of failing the instance outright. |
| `notification` | Sends via the existing `internal/features/notification.Service.Publish` (in-app/push), template + variables from node config; failure also retries via `wf_job`. |
| `timer` | Creates a `wf_job(job_type='timer')` scheduled for `now()+duration` or a fixed timestamp; token `waiting` until the worker fires it. |
| `subProcess` | Starts a **new** `wf_instance` of another `wf_definition` (latest published version at call time, per node config), current token stays `waiting`; on the sub-instance reaching `completed`/`rejected`, its outcome is written back into the parent token's node's outgoing edge selection. |
| `end` | Terminal; carries an `outcome` (e.g. `approved`/`rejected`/`completed`) written to `wf_instance.status` and `ended_at`. |

Graph JSON shape (stored in `wf_definition_version.graph`):
```json
{
  "nodes": [
    {"id": "n1", "type": "start", "config": {}},
    {"id": "n2", "type": "approval", "config": {
      "assignees": [{"type": "roleTier", "levels": 1}],
      "decisionMode": "all"
    }}
  ],
  "edges": [
    {"id": "e1", "from": "n1", "to": "n2"},
    {"id": "e2", "from": "n2", "to": "n3", "condition": {"...": "..."}, "isDefault": false}
  ]
}
```

## 1.3 Assignee resolution

Candidate-set model: every resolution produces a **set of user ids** (even `userTask`,
which typically resolves to exactly one). Supported source types:

| Type | Resolution |
|---|---|
| `user` | explicit `user_id` from node config |
| `initiator` | the user who started this instance — resolved per-instance from `wf_instance.initiator_id`, not a fixed id authored on the node (added after Phase 3 FE feedback: self-assignment for the requester) |
| `role` | all workspace members with `workspace_role` = the given role (`holder`/`admin`/`member`) |
| `roleTier` (manager resolution) | role-tier chain per §0.3: initiator's role `member` → level 1 = all `admin`s of the workspace; initiator's role `admin` → level 1 = the `holder`; initiator's role `holder` → no level exists. `levels: N` walks N steps up this chain (max 2); if `N` exceeds the chain length → unresolvable. |
| `formField` | value of a `user`-typed field in the instance's `variables`, read at resolution time |
| `expression` | a condition-engine expression (§1.4) that must evaluate to a user id or array of user ids |
| `group`/department | **not supported in v1** — there is no department/org-unit concept in the schema (confirmed absent); referencing this type is a config-validation error at publish time, not a runtime surprise. |

**Unresolvable candidate set** (empty after resolution, e.g. `roleTier` from a `holder`, or
an `expression` that returns nothing): the token's node transitions to an error state, the
instance's `status` becomes `error` with `error_message` set, a `wf_history` entry is
recorded, and a notification is sent to the workspace's `holder` (admin escalation contact),
per the task's explicit requirement.

## 1.4 Expression & condition engine

Stored as a JSON AST (no `eval`, no dynamic code execution), e.g.:
```json
{"op": "and", "args": [
  {"op": ">",  "args": [{"var": "form.days"}, {"const": 2}]},
  {"op": "==", "args": [{"var": "initiator.role"}, {"const": "member"}]}
]}
```
- `{"var": "a.b.c"}` — dotted-path lookup into a context object built from `variables`
  (`form.*`), a snapshot of the initiator (`initiator.id/name/alias/role` — only fields that
  actually exist on `users`/`workspace_users` today; no `initiator.department`, since that
  field doesn't exist anywhere in the schema), and `instance.*` (status, started_at, step_count).
- `{"const": <json value>}` — literal.
- Operators: `==`, `!=`, `>`, `>=`, `<`, `<=`, `and`, `or`, `not`, `in`. Functions: `now`,
  `dateDiff`, `sum`, `len`.
- Implementation: a pure-Go recursive tree-walking evaluator (own package, no dependency
  added). Safety bounds: max AST depth (20) and max node count (500) checked before
  evaluation; evaluation runs under a short `context.WithTimeout` (e.g. 50ms). Any bound
  violation or evaluation error is caught, logged to `wf_history` as an `error` event, and
  transitions the *instance* to `status='error'` — it never panics/crashes the worker.

## 1.5 Form schema

`wf_definition_version.form_schema` shape:
```json
{
  "fields": [
    {
      "key": "days",
      "label": "Number of days",
      "type": "number",
      "required": true,
      "default": 1,
      "validation": {"min": 1, "max": 30},
      "visibleWhen": {"op": "==", "args": [{"var": "form.leaveType"}, {"const": "annual"}]},
      "readonlyWhen": null,
      "permission": {"nodeIds": ["n1"], "mode": "editable"}
    }
  ]
}
```
Field `type`: `text | number | date | dateRange | select | multiselect | user | file | textarea | checkbox`.
`options` for `select`/`multiselect` are either a static array in the schema or `{"source": "api", "endpoint": "..."}`
resolved by the FE at render time. `visibleWhen`/`readonlyWhen` reuse the same condition-AST
format as §1.4. `permission` scopes which node(s) can edit vs. only view a field (e.g. the
initiator fills it at `start`, an approver only sees it read-only at `n2`).

**The same schema renders on FE and is re-validated on BE** — the BE re-runs `required`,
`validation` (min/max/regex/etc.) and `visibleWhen` (a hidden field cannot carry a value)
against submitted `form_data`/`variables` before accepting a `completeTask()`/`start()` call;
the FE is never trusted as the source of truth.

## 1.6 Non-functional requirements

- **Transactions**: every `advance()` step runs inside one Postgres transaction (`pgxpool.Pool.Begin`
  + `queries.WithTx(tx)`, same manual pattern as `workspace.Repository.ExecTx` — no new
  Tx/UnitOfWork abstraction introduced).
- **Idempotency**: each `advance()` call is guarded by optimistic locking — it reads
  `wf_token.version`/`wf_instance.version`, and its `UPDATE` includes `WHERE version = $current`;
  a 0-row update means someone else already advanced this token/instance, so the call is a
  no-op (not an error) — this is what makes concurrent duplicate calls (e.g. a retried job,
  or a network-retried API call) safe to re-run.
- **Optimistic locking for concurrent approvals**: two approvers deciding "simultaneously"
  on the same `(token_id, node_id)` group both write their own `wf_task` row (no shared
  mutable row between them), so there's no write conflict at that level; the *token* is
  advanced by whichever decision satisfies `decision_mode` first, guarded by the token's
  `version` column so only one of them actually performs the transition.
- **Retry + backoff**: `serviceTask`/`notification` failures create/update a `wf_job(job_type='retry')`
  with `scheduled_at = now() + backoff(attempts)` (exponential, e.g. `2^attempts` seconds
  capped at some max); once `attempts >= max_attempts`, status becomes `dead_letter` and the
  job stops being picked up by the poller — surfaced on an admin retry screen (§6).
- **Infinite-loop guard**: `wf_instance.step_count` increments on every `advance()`; a hard
  ceiling (config constant, e.g. 1000) forces the instance to `status='error'` instead of
  looping forever.
- **Version pinning**: `wf_instance.definition_version_id` is set once at `start()` and never
  changes — publishing a new version never affects instances already running.
- **Audit**: every state transition, decision, and variable change writes a `wf_history` row
  with `data_before`/`data_after`, `actor_id`, and timestamp — this *is* the audit trail (no
  separate audit-log table needed, matching that the codebase has no generic audit mechanism
  to plug into).

## 2. Permission mapping (reusing `workspace_role`)

No new permission-key table. Mapped onto the existing 3-tier workspace role:

| Task's permission | Maps to |
|---|---|
| `workflow.admin` (manage definitions: create/edit draft/publish/archive/clone/simulate) | workspace role = `holder` |
| `workflow.create_request` | any workspace member (`holder`/`admin`/`member` — anyone in the workspace) |
| `workflow.approve` | not a static role — dynamic: "you have a `pending`/`claimed` `wf_task` assigned to you" (checked per-request, same idea as the existing `Check Requests` holder-only screen, but here it's per-task not per-role) |
| `workflow.view_all` (admin instance-monitoring screen, retry dead-letters) | workspace role = `holder` |

This mirrors the existing convention exactly (`requireHolder`/`roleCanViewAll`-style
imperative checks in the service layer — no new middleware/decorator).

## 3. Engine module shape (backend)

Pure-logic core, no HTTP dependency, package `internal/features/workflow`:
- `Engine.Start(ctx, definitionVersionID, initiatorID, formData) (*Instance, error)`
- `Engine.Advance(ctx, tokenID) error` — internal, drives automatic nodes forward
- `Engine.CompleteTask(ctx, taskID, actorID, decision, formData, comment) error`
- `Engine.Cancel(ctx, instanceID, actorID, reason) error`
- `Engine.EvaluateCondition(ctx, ast, context) (bool, error)`
- `Engine.ResolveAssignees(ctx, config, instanceCtx) ([]uuid.UUID, error)`

Followed by the same Handler → Service → Repository layering as `attendance`, with the
engine living inside the service layer (or as a sibling package the service calls into —
exact split TBD during implementation, doesn't change the public surface above).

## 4. Background worker

`WorkflowJobWorker.Start(ctx)` — a `time.NewTicker` goroutine (interval TBD, e.g. 10s),
started in `NewServer` next to `AttendanceReminderService`/task deadline reminder. Each tick:
claim due `wf_job` rows (`FOR UPDATE SKIP LOCKED`, §1.6's wf_job index), execute by
`job_type`, update status/attempts/backoff. No watermill, no Redis, no new dependency.

## 5. API surface (sketch, `/v1/workspaces/{workspaceId}/workflows/...`)

- **Admin**: `GET/POST /definitions`, `GET/PATCH /definitions/{id}`, `POST /definitions/{id}/versions`
  (create draft), `POST /definitions/{id}/versions/{v}/publish`, `POST /definitions/{id}/archive`,
  `POST /definitions/{id}/clone`, `POST /definitions/{id}/versions/{v}/validate`,
  `POST /definitions/{id}/versions/{v}/simulate`.
- **User**: `GET /catalog` (creatable definitions), `POST /instances` (start), `GET /instances/mine`,
  `GET /instances/{id}` (detail + timeline from `wf_history`), `POST /instances/{id}/cancel`,
  `POST /instances/{id}/comments`, `POST /instances/{id}/attachments`.
- **Approver**: `GET /tasks/inbox` (my pending tasks), `POST /tasks/{id}/approve|reject|request-change`,
  `POST /tasks/{id}/delegate`, `GET /tasks/{id}/history`.
- **Admin monitoring**: `GET /instances` (all, filter by status), `GET /jobs?status=dead_letter`,
  `POST /jobs/{id}/retry`.

All list endpoints follow the existing `pkg/pagination` convention (`?page&limit`, `Response[T]{Items,HasMore,Page,Limit}`).
Filters follow the existing `schema:"..."` query-binding convention. Errors follow the
existing `{"error": "..."}` envelope via `pkg/httpx`.

## 6. Frontend (sketch)

Confirmed split (Phase 0 §15/§7):
- `lib/ui/screens/workflow` — catalog, create-request form (schema-driven renderer, built
  new for this module since no dynamic form/JSON-schema renderer exists today), "my requests"
  list + detail/timeline (built from `wf_history`).
- `lib/ui/screens/workflow_inbox` — approval inbox, mirroring `check_requests`'s structure
  (local Cubit via `GetIt.I<WorkflowRepository>()`, `RealtimeSource` subscription for
  auto-refresh, `BlocConsumer` + `Toast` for errors).
- `lib/ui/screens/workflow_admin` — process designer (canvas — **library choice pending**,
  to be asked explicitly before adding any dependency) + form builder + version management +
  instance/job admin screen.
- Data layer: `lib/data/models/workflow.dart` (hand-written, matching the plain-class +
  manual `copyWith`/`toJson` convention used by `attendance.dart`, not freezed), `lib/data/repo/workflow_repository.dart`
  (`@LazySingleton`, wraps the swagger-generated client with `safeCall`/`safeCallCached`).
  API client itself is generated into `lib/generated/api/workflow*/` once the backend's
  swagger is updated — never hand-written.
- Reused widgets: `StatusBadge` (task/instance status pills), `RoundedButton`, `AppDropdown`/
  `AppSelectField`/`AppMultiSelectDropdown`/`DateTimeField` (form fields), `AppConfirmDialog`
  (cancel/reject confirmation), `Toast`, `expandable_resource_panel`/`resource_file_item`
  (attachments), `GanttZoom`'s date-bucketing math as a reference for the timeline's linear
  time axis (not for the canvas — no canvas primitive exists in the app yet).
- Entry point wiring: a new tile in `lib/ui/screens/workspace/widgets/workspace_features_section.dart`
  (gated on workspace role, following the exact pattern already used for "Check Requests"/"Schedules"),
  and route additions in `lib/ui/route/app_route.dart` + `lib/ui/route/route_paths.dart`.
  These are the only two *existing* files this module needs to touch, and only to add
  entries — no existing route/logic is modified.
- No i18n, no shared empty/loading/error widgets (follows existing per-screen convention of
  hand-rolled `_EmptyState` etc., to stay consistent with the rest of the app).

## 7. Open items still needing a decision before/at Phase 3

- Canvas/diagram library for the process designer (will present lightweight options then).
- Exact polling interval for the `wf_job` worker and the retry backoff formula's constants.
- Whether `subProcess`'s "current published version" resolution should be re-checked only at
  start or also validated at publish time (dangling reference check, see graph validation
  rules already specified in the task: gateway must have a default edge, every node reachable
  and has a path to `end`, no reference to a non-existent variable).
