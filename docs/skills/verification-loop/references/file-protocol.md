# File Protocol

For each run:

- `.../iter-001/inbox/goal.md`
- `.../iter-001/inbox/constraints.md`
- `.../iter-001/inbox/context-pack.md`
- `.../iter-001/inbox/session-ref.json`
- `.../iter-001/outbox/planner.log`
- `.../iter-001/outbox/implementer.log`
- `.../iter-001/outbox/verify.log`
- `.../iter-001/outbox/inspector.log`
- `.../iter-001/outbox/inspector.json`
- `.../iter-001/summary.md`
- `.../report.json`
- `.../report.md`
- `.../review-checklist.md`

Environment variables available to planner/implementer/verifier/inspector commands:

- `OCX_LAB_RUN_DIR`
- `OCX_LAB_ITER_DIR`
- `OCX_LAB_WORKSPACE`
- `OCX_LAB_GOAL_FILE`
- `OCX_LAB_CONSTRAINTS_FILE`
- `OCX_LAB_CONTEXT_PACK_FILE`
- `OCX_LAB_CONTEXT_PACK_SHA256`
- `OCX_LAB_PLAN_FILE`
- `OCX_LAB_IMPL_FILE`
- `OCX_LAB_VERIFY_LOG_FILE`
- `OCX_LAB_INSPECTOR_LOG_FILE`
- `OCX_LAB_INSPECTOR_JSON_FILE`
- `OCX_LAB_SESSION_REF_FILE`
- `OCX_LAB_SESSION_PATH_FILE`
