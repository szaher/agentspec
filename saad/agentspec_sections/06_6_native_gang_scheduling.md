## 6. Native Gang Scheduling

Support a scheduling DSL:

schedule:
  queue: research
  priority: high
  gang:
    min_members: 4
    timeout: 60s

Backends:

- local scheduler
- Kubernetes native gang scheduling
- Kueue
- Volcano