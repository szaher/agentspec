# Quickstart: Graph Visualization

**Feature**: 016-graph-visualization

## Scenario 1: Interactive Web Visualization

```bash
# Visualize a single file — browser opens automatically
agentspec graph examples/multi-agent-router.ias

# Output:
# Serving graph at http://127.0.0.1:8686
# Press Ctrl+C to stop

# Browser opens with interactive graph showing:
# - 4 agent nodes (blue)
# - 5 skill nodes (purple)
# - 4 prompt nodes (green)
# - 1 deploy target node (slate)
# - Labeled edges: "uses prompt", "uses skill", "delegates to"
# - Click any node → sidebar shows details
# - Scroll to zoom, drag to pan
```

## Scenario 2: Mermaid Export for Documentation

```bash
# Generate Mermaid and embed in README
agentspec graph examples/multi-agent-router.ias --format mermaid > architecture.md

# Expected output (example):
# graph LR
#   agent_router([router]):::agent
#   agent_specialist([specialist]):::agent
#   prompt_system[system]:::prompt
#   skill_search[search]:::skill
#   agent_router -->|uses prompt| prompt_system
#   agent_router -->|uses skill| skill_search
#   agent_router -.->|delegates to| agent_specialist
#   classDef agent fill:#4A9EFF,stroke:#333,color:#fff
#   classDef prompt fill:#4ADE80,stroke:#333,color:#fff
#   classDef skill fill:#A78BFA,stroke:#333,color:#fff
```

## Scenario 3: DOT Export for Image Generation

```bash
# Generate DOT and render as PNG
agentspec graph examples/research-swarm.ias --format dot --output graph.dot
dot -Tpng graph.dot -o graph.png

# Expected graph.dot (excerpt):
# digraph agentspec {
#   rankdir=LR;
#   node [fontname="sans-serif"];
#   "agent:coordinator" [shape=box,style=rounded,label="coordinator"];
#   "skill:web-search" [shape=component,label="web-search"];
#   "agent:coordinator" -> "skill:web-search" [label="uses skill"];
# }
```

## Scenario 4: Directory Visualization

```bash
# Visualize entire multi-file project
agentspec graph examples/multi-file-agent/

# Browser opens showing:
# - File nodes for main.ias, skills/search.ias, skills/respond.ias
# - Import edges between files
# - Cross-file references (agent in main.ias → skill in search.ias)
# - Nodes grouped/annotated by source file
```

## Scenario 5: Filtered Graph

```bash
# Hide file nodes and orphaned entities
agentspec graph examples/production-agent.ias --no-files --no-orphans

# Shows only entities with connections, without the file layer
```

## Scenario 6: Light Theme and Custom Port

```bash
# Use light theme on port 9090
agentspec graph examples/multi-agent-router.ias --theme light --port 9090

# Output:
# Serving graph at http://127.0.0.1:9090
```

## Scenario 7: Headless/CI Usage

```bash
# Generate Mermaid in CI without browser
agentspec graph . --format mermaid --no-open > graph.md

# Verify DOT is valid
agentspec graph . --format dot | dot -Tsvg > /dev/null && echo "Valid DOT"
```

## Scenario 8: Error Resilience

```bash
# Directory with one broken file
agentspec graph examples/

# stderr: examples/broken.ias:15:3: unexpected token "}"
# Browser still opens with graph from valid files
# Warning banner in UI: "1 file had parse errors"
```

## Validation Checklist

- [ ] `agentspec graph examples/multi-agent-router.ias` opens browser with correct graph
- [ ] `agentspec graph examples/multi-agent-router.ias --format mermaid` produces valid Mermaid
- [ ] `agentspec graph examples/multi-agent-router.ias --format dot` produces valid DOT
- [ ] `agentspec graph examples/multi-file-agent/` shows cross-file relationships
- [ ] `agentspec graph examples/research-swarm.ias` shows pipeline steps in order
- [ ] Ctrl+C cleanly stops the web server
- [ ] `--no-open` prevents browser from opening
- [ ] `--theme light` renders with light background
- [ ] `--output graph.dot` writes to file instead of stdout
- [ ] Parse errors are shown on stderr, graph renders valid files
