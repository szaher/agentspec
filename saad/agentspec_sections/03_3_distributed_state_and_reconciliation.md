## 3. Distributed State and Reconciliation
Move beyond `.agentspec.state.json`.

Support backends:

- local JSON (dev)
- SQLite (single-node)
- Postgres (production)

The operator reconciles desired state from this control-plane database.