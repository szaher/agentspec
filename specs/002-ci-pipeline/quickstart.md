# Quickstart: Verifying the CI Pipeline

**Goal**: Confirm the CI pipeline is working correctly after implementation.

## Step 1: Check the Workflow File Exists

```bash
cat .github/workflows/ci.yml
```

The file should exist and contain steps for build, test, lint, and example validation.

## Step 2: Push a Commit and Observe CI

```bash
git add .github/workflows/ci.yml
git commit -m "Add CI pipeline"
git push origin 002-ci-pipeline
```

Go to the repository on GitHub. Navigate to the "Actions" tab. A workflow run should appear within seconds.

## Step 3: Verify All Steps Pass

The workflow run should show these steps completing successfully:

1. Checkout
2. Set up Go
3. Build binary
4. Run tests
5. Lint
6. Validate examples
7. Format-check examples
8. Smoke test (plan + apply + idempotency)

## Step 4: Verify PR Status Checks

Open a pull request from the branch. The CI status check should appear on the PR page before merging is allowed.

## Step 5: Verify Failure Detection

To confirm the pipeline catches errors, temporarily introduce a build failure:

```bash
echo "invalid go code" >> cmd/agentz/main.go
git commit -am "test: introduce build failure"
git push
```

The CI run should fail at the build step. Revert the change afterward:

```bash
git revert HEAD
git push
```

## Verification Checklist

- [ ] Workflow file exists at `.github/workflows/ci.yml`
- [ ] CI triggers on push to any branch
- [ ] CI triggers on pull requests to main
- [ ] Build step compiles the binary successfully
- [ ] Test step runs all tests with `-count=1`
- [ ] Lint step passes with golangci-lint
- [ ] All example files validate without errors
- [ ] All example files pass format check
- [ ] Smoke test completes plan + apply + idempotency cycle
- [ ] CI completes within 10 minutes
- [ ] Failed commits show clear error messages
