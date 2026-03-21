# GPU Batch

An agent that dispatches batch inference tasks to an external GPU cluster API. Demonstrates the async task dispatch pattern: validate input, submit a job, poll for completion, and fetch results.

## Architecture Overview

```
Inference Request
    |
    v
batch-agent (agent)
    |
    +---> validate-input   -- validates and preprocesses the payload
    +---> submit-job       -- submits to GPU cluster API, returns job_id
    +---> poll-status      -- checks job status until complete
    +---> fetch-results    -- downloads output from completed job
```

The agent orchestrates an asynchronous workflow: it validates input data, submits a job to a remote GPU endpoint, polls until the job completes (or fails), and retrieves the results. This pattern is common for large-scale ML inference, image generation, and batch processing workloads.

## Prerequisites

1. Build the AgentSpec CLI from the repository root:

   ```bash
   go build -o agentspec ./cmd/agentspec
   ```

2. Set the required environment variable:

   ```bash
   export GPU_CLUSTER_API_KEY="your-gpu-cluster-api-key"
   ```

3. Ensure the following tool binaries are available on `$PATH` (or stubbed for testing):
   - `input-validator`
   - `gpu-job-submit`
   - `gpu-job-status`
   - `gpu-results-fetch`

## Step-by-Step Run Instructions

```bash
# 1. Validate the AgentSpec
./agentspec validate examples/gpu-batch/gpu-batch.ias

# 2. Preview planned changes
./agentspec plan examples/gpu-batch/gpu-batch.ias

# 3. Apply the changes
./agentspec apply examples/gpu-batch/gpu-batch.ias --auto-approve

# 4. Run the agent with a sample inference request
./agentspec dev examples/gpu-batch/gpu-batch.ias --input "Run image classification on dataset-2026-03"

# 5. Export artifacts
./agentspec export examples/gpu-batch/gpu-batch.ias --out-dir ./output
```

## Customization Tips

- **Add retry logic**: Define a `retry-job` skill that resubmits failed jobs with exponential backoff.
- **Add cost estimation**: Insert a `estimate-cost` skill before `submit-job` to calculate expected GPU hours and abort if over budget.
- **Multi-model dispatch**: Add a `model_id` parameter to the prompt or input to dispatch jobs to different model endpoints (e.g., stable-diffusion, llama, whisper).
- **Add environment overlays**: Use `environment` blocks to point dev at a mock GPU API and production at the real cluster.
- **Pipeline extension**: Wrap the agent in a `pipeline` with upstream data-preparation agents and downstream result-evaluation agents.
