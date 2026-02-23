# File Protocol

For each run:

- `.../iter-001/inbox/goal.md`
- `.../iter-001/inbox/constraints.md`
- `.../iter-001/outbox/planner.log`
- `.../iter-001/outbox/implementer.log`
- `.../iter-001/outbox/verify.log`
- `.../iter-001/outbox/judge.log`
- `.../iter-001/summary.md`
- `.../report.json`
- `.../report.md`

Environment variables available to planner/implementer/verifier/judge commands:

- `OCX_LAB_RUN_DIR`
- `OCX_LAB_ITER_DIR`
- `OCX_LAB_WORKSPACE`
- `OCX_LAB_GOAL_FILE`
- `OCX_LAB_PLAN_FILE`
- `OCX_LAB_IMPL_FILE`
- `OCX_LAB_JUDGE_FILE`
