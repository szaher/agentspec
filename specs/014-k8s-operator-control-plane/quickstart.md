# Quickstart: AgentSpec Kubernetes Operator

## Scenario 1: Deploy a Single Agent (US1)

### Prerequisites
- Kubernetes 1.28+ cluster (minikube, kind, or cloud)
- `kubectl` configured
- `agentspec` CLI installed

### Steps

1. **Install the operator and CRDs**:
   ```bash
   kubectl apply -f config/crd/bases/
   kubectl apply -f config/rbac/
   kubectl apply -f config/manager/
   ```

2. **Write an IntentLang agent spec** (`my-agent.ias`):
   ```
   package "my-agent" version "0.1.0" lang "2.0"

   prompt "assistant-prompt" {
     content "You are a helpful assistant."
   }

   agent "assistant" {
     uses prompt "assistant-prompt"
     model "claude-sonnet-4-20250514"
     strategy "react"
     max_turns 5
   }

   secret "api-key" {
     env(ANTHROPIC_API_KEY)
   }
   ```

3. **Generate CRD manifests**:
   ```bash
   agentspec generate crds my-agent.ias --output-dir ./manifests --namespace default
   ```

4. **Create the API key secret**:
   ```bash
   kubectl create secret generic api-key --from-literal=ANTHROPIC_API_KEY=$ANTHROPIC_API_KEY
   ```

5. **Apply the agent**:
   ```bash
   kubectl apply -f ./manifests/
   ```

6. **Verify the agent is ready**:
   ```bash
   kubectl get agents
   # NAME        MODEL                      PHASE   AGE
   # assistant   claude-sonnet-4-20250514   Ready   30s

   kubectl describe agent assistant
   # Conditions:
   #   Type: Ready, Status: True, Reason: Provisioned
   ```

### Expected Result
- Agent CR shows `Ready` phase within 30 seconds
- Operator emits `Provisioning` and `Ready` events
- `kubectl get agents` shows the agent with model and phase columns

---

## Scenario 2: Multi-Agent Workflow (US2)

### Steps

1. **Create multiple agents** (assumes agents from Scenario 1 pattern):
   ```bash
   kubectl apply -f manifests/researcher-agent.yaml
   kubectl apply -f manifests/writer-agent.yaml
   kubectl apply -f manifests/reviewer-agent.yaml
   ```

2. **Apply a workflow**:
   ```yaml
   apiVersion: agentspec.io/v1alpha1
   kind: Workflow
   metadata:
     name: content-pipeline
   spec:
     steps:
     - name: research
       agentRef: researcher
       input: "Research the topic: Kubernetes operators"
     - name: write
       agentRef: writer
       dependsOn: ["research"]
     - name: review
       agentRef: reviewer
       dependsOn: ["write"]
     failFast: true
   ```

3. **Monitor progress**:
   ```bash
   kubectl get workflow content-pipeline -w
   # NAME               PHASE     STEP       AGE
   # content-pipeline   Running   research   10s
   # content-pipeline   Running   write      45s
   # content-pipeline   Completed review     1m20s
   ```

### Expected Result
- Steps execute in dependency order (research → write → review)
- Workflow status shows per-step progress and durations
- Final output available in workflow status

---

## Scenario 3: Policy Enforcement (US5)

### Steps

1. **Create a policy**:
   ```yaml
   apiVersion: agentspec.io/v1alpha1
   kind: Policy
   metadata:
     name: team-budget
   spec:
     costBudget:
       maxDailyCost: "10.00"
       currency: USD
     allowedModels:
     - "claude-sonnet-4-20250514"
     - "claude-haiku-4-5-20251001"
     rateLimits:
       requestsPerMinute: 60
       tokensPerMinute: 100000
   ```

2. **Try to deploy an agent with a disallowed model**:
   ```yaml
   apiVersion: agentspec.io/v1alpha1
   kind: Agent
   metadata:
     name: expensive-agent
   spec:
     model: "claude-opus-4-6"
     policyRef: team-budget
   ```

3. **Check the result**:
   ```bash
   kubectl describe agent expensive-agent
   # Conditions:
   #   Type: Ready, Status: False, Reason: PolicyViolation
   #   Message: Model "claude-opus-4-6" not in allowed list for policy "team-budget"
   ```

### Expected Result
- Agent rejected with `PolicyViolation` status
- No runtime resources provisioned
- Clear error message explaining the violation

---

## Scenario 4: Scheduled Evaluation (US6 + US8)

### Steps

1. **Create an EvalRun template and schedule**:
   ```yaml
   apiVersion: agentspec.io/v1alpha1
   kind: Schedule
   metadata:
     name: nightly-eval
   spec:
     schedule: "0 2 * * *"
     timezone: "UTC"
     targetRef:
       kind: EvalRun
       name: assistant-eval
     concurrencyPolicy: Forbid
     successfulTasksHistoryLimit: 7
   ```

2. **Verify the schedule**:
   ```bash
   kubectl get schedules
   # NAME          SCHEDULE    NEXT RUN              SUSPEND
   # nightly-eval  0 2 * * *   2026-03-22T02:00:00Z  false
   ```

### Expected Result
- Schedule creates EvalRun resources at the configured time
- Results accumulate with history maintained per `successfulTasksHistoryLimit`
- Missed schedules are recorded in status
