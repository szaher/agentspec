## 7. Dependency Strategy

AgentSpec should own:

- DSL
- packaging
- operator
- ADLC
- scheduler abstraction
- evaluation engine
- governance workflows

External integrations remain pluggable:

- models
- vector stores
- identity providers
- telemetry sinks