# Quickstart: Agent Compilation & Deployment

**Feature**: 006-agent-compile-deploy

## 1. Define Your Agent

Create `support-agent.ias`:

```
package "support-agent" version "1.0.0" lang "3.0"

import "./skills/faq.ias"

prompt system_prompt {
  content "You are a helpful customer support agent. Answer questions
    accurately and politely. If you cannot help, escalate to a human."
}

skill create_ticket {
  description "Create a support ticket"
  input { subject string, body string, priority int }
  output { ticket_id string }
  tool http {
    method POST
    url "${config.ticket_api_url}/tickets"
  }
}

agent support {
  prompt system_prompt
  model "claude-sonnet-4-20250514"
  loop react max_turns 10

  config {
    anthropic_api_key string required secret
      "Anthropic API key"
    ticket_api_url string default "https://api.example.com"
      "Ticket system API base URL"
  }

  on input {
    if input.category == "billing" {
      use skill faq_search with { topic: "billing" }
    } else {
      use skill faq_search with { topic: "general" }
    }

    if steps.faq_search.output.confidence < 0.5 {
      use skill create_ticket with {
        subject: input.content,
        body: "Auto-escalated: low confidence response",
        priority: 3
      }
    }
  }

  validate {
    rule no_pii error max_retries 2
      "Must not expose personal information"
      when not (output matches "\\b\\d{3}-\\d{2}-\\d{4}\\b")
  }

  eval {
    case greeting
      input "Hi, I need help with my account"
      expect "Friendly greeting with offer to assist"
      scoring semantic threshold 0.8
  }
}
```

Create `skills/faq.ias`:

```
package "support-agent" version "1.0.0" lang "3.0"

skill faq_search {
  description "Search the FAQ knowledge base"
  input { topic string }
  output { answer string, confidence float }
  tool http {
    method GET
    url "https://faq.example.com/search"
    query { q: input.topic }
  }
}
```

## 2. Compile

```bash
# Compile to standalone binary
agentspec compile support-agent.ias

# Output:
#   ✓ Parsed 2 files (1 import resolved)
#   ✓ Validated 1 agent, 2 skills, 1 validation rule
#   ✓ Compiled to standalone binary
#   Output: ./build/support-agent (15.2 MB)
```

Cross-compile for Linux:

```bash
agentspec compile --platform linux/amd64 support-agent.ias
```

## 3. Run

```bash
# Set required config
export AGENTSPEC_SUPPORT_ANTHROPIC_API_KEY="sk-ant-..."

# Run the agent
./build/support-agent --port 8080

# Output:
#   Agent "support" ready at http://0.0.0.0:8080
#   Frontend: http://0.0.0.0:8080/
#   API key required (set AGENTSPEC_API_KEY or use --no-auth)
```

## 4. Interact

### Via API

```bash
export AGENTSPEC_API_KEY="my-secret-key"

curl -X POST http://localhost:8080/v1/agents/support/invoke \
  -H "Authorization: Bearer $AGENTSPEC_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"input": "How do I reset my password?"}'
```

### Via Built-in Frontend

Open `http://localhost:8080/` in a browser. Enter your API key when prompted. Chat with the agent directly.

## 5. Evaluate

```bash
agentspec eval support-agent.ias
# Output:
#   ✓ greeting  score: 0.92 (threshold: 0.80)
#   Results: 1/1 passed (100%)
```

## 6. Package for Deployment

### Docker

```bash
agentspec package --format docker --tag support-agent:1.0.0 ./build/support-agent
# Output: Docker image support-agent:1.0.0 (45 MB)

docker run -e AGENTSPEC_API_KEY=my-key \
  -e AGENTSPEC_SUPPORT_ANTHROPIC_API_KEY=sk-ant-... \
  -p 8080:8080 support-agent:1.0.0
```

### Kubernetes

```bash
agentspec package --format kubernetes --output ./k8s/ ./build/support-agent
# Output:
#   ./k8s/deployment.yaml
#   ./k8s/service.yaml
#   ./k8s/configmap.yaml

kubectl apply -f ./k8s/
```

## 7. Compile to Framework Code

```bash
# Generate CrewAI project
agentspec compile --target crewai --output ./crewai-project support-agent.ias
# Output:
#   ./crewai-project/
#   ├── pyproject.toml
#   ├── main.py
#   ├── crew.py
#   ├── config/agents.yaml
#   ├── config/tasks.yaml
#   └── tools/__init__.py

cd crewai-project
pip install -e .
python main.py
```

## 8. Publish a Package

```bash
# Create a reusable skills package
agentspec publish ./my-skills-package/
# Output:
#   Published github.com/myuser/support-skills@v1.0.0
#   Checksum: sha256:a1b2c3d4...

# Others can now import it:
# import "github.com/myuser/support-skills" version "1.0.0"
```
