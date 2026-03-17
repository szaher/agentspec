import os
import zipfile
import re
from pathlib import Path

base = Path("agentspec_docs")
sections_dir = base / "agentspec_sections"

base.mkdir(exist_ok=True)
sections_dir.mkdir(exist_ok=True)

roadmap = """
# AgentSpec Roadmap Toward AgentOS and Full ADLC

## 1. Adoption and Developer Experience
Focus on making AgentSpec installable and usable in minutes.
Provide packaged releases, Homebrew install, Docker images, and starter templates.

Deliver runnable examples like:
- support bot
- RAG assistant
- research swarm
- incident-response agent
- GPU batch agent
- multi-agent router

## 2. Kubernetes Operator and Control Plane
Introduce an AgentSpec operator with CRDs:

- Agent
- Task
- Session
- Workflow
- MemoryClass
- ToolBinding
- Policy
- Schedule
- EvalRun
- Release

Replace simple manifest generation with reconciliation-driven lifecycle management.

## 3. Distributed State and Reconciliation
Move beyond `.agentspec.state.json`.

Support backends:

- local JSON (dev)
- SQLite (single-node)
- Postgres (production)

The operator reconciles desired state from this control-plane database.

## 4. Full Agent Development Lifecycle (ADLC)

Lifecycle stages:

Design → Develop → Test → Evaluate → Package → Release → Deploy → Observe → Improve → Retire

Capabilities:

- replayable sessions
- dataset versioning
- regression testing
- promotion pipelines
- experiment tracking

## 5. Native Scheduling Layer

Add scheduling primitives to IntentLang:

- queues
- priority
- deadlines
- parallelism limits
- placement rules
- retries
- resource budgets

Scheduling must be a **core subsystem**.

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

## 8. Adoption Wedge

Position AgentSpec as:

GitOps for agents
+ Kubernetes-native scheduling
+ full ADLC lifecycle

Focus on enterprise-grade:

- governance
- auditability
- reproducibility

## 9. Phased Implementation Plan

Phase 1:
- packaging
- documentation
- starter examples

Phase 2:
- operator
- CRDs

Phase 3:
- ADLC lifecycle features

Phase 4:
- scheduler service
- scheduling DSL

Phase 5:
- gang scheduling support

Phase 6:
- full AgentOS control plane
""".strip()


# write roadmap
roadmap_path = base / "agentspec_roadmap.md"
roadmap_path.write_text(roadmap)

# split sections
sections = re.split(r"\n(?=## )", roadmap)

files = []
for i, sec in enumerate(sections):
    if sec.startswith("# "):
        continue

    title = sec.split("\n")[0].replace("## ", "")
    filename = f"{i:02d}_{re.sub(r'[^a-zA-Z0-9]+','_',title).lower()}.md"

    path = sections_dir / filename
    path.write_text(sec.strip())
    files.append(filename)

# index
index = "# AgentSpec Sections\n\n"
for f in files:
    index += f"- {f}\n"

(sections_dir / "README.md").write_text(index)

# splitter utility
splitter = """
import os
import re
import argparse

parser = argparse.ArgumentParser()
parser.add_argument("markdown_file")
parser.add_argument("--output-dir", default="sections")
args = parser.parse_args()

os.makedirs(args.output_dir, exist_ok=True)

with open(args.markdown_file) as f:
    content = f.read()

sections = re.split(r"\\n(?=## )", content)

for i, sec in enumerate(sections):
    if sec.startswith("# "):
        continue

    title = sec.split("\\n")[0].replace("## ", "")
    filename = f"{i:02d}_{re.sub(r'[^a-zA-Z0-9]+','_',title).lower()}.md"

    with open(os.path.join(args.output_dir, filename), "w") as out:
        out.write(sec.strip())

print("Sections written to", args.output_dir)
"""

(base / "split_markdown_sections.py").write_text(splitter.strip())

# zip bundle
zip_path = "agentspec_bundle.zip"

with zipfile.ZipFile(zip_path, "w") as z:
    for root, dirs, files in os.walk(base):
        for file in files:
            full = os.path.join(root, file)
            z.write(full)

print("Generated:")
print(" - agentspec_docs/")
print(" - agentspec_bundle.zip")
