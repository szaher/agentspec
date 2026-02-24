# version

Print the AgentSpec CLI version information.

## Usage

```bash
agentspec version
```

## Description

The `version` command prints the current version of the `agentspec` binary along with the build date and Git commit hash. This is useful for debugging, filing bug reports, and verifying that the correct version is installed.

The output includes three fields: the semantic version number, the date the binary was built, and the short commit hash from which it was built.

## Flags

This command has no additional flags.

## Examples

```bash
# Print version information
agentspec version

# Use in a script to check the installed version
agentspec version | head -1
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Version information printed successfully |
| `1` | An error occurred |
