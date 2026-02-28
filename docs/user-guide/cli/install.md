# install

Install a package from a Git repository.

## Usage

```bash
agentspec install <package@version>
```

## Description

The `install` command resolves a package from its Git repository, downloads it to the local cache, and updates the lock file. This is used to bring in reusable AgentPack packages that your specs depend on.

The package reference follows the format `<source>@<version>`, where `source` is a Git repository URL and `version` is a Git tag or branch. If no version is specified, the latest available version is used.

The command also resolves transitive dependencies declared in the package's `agentpack.yaml` manifest, downloading them automatically.

## Package Reference Format

```
<git-url>@<version>
```

| Part | Description |
|------|-------------|
| `<git-url>` | Git repository URL (e.g. `github.com/org/my-agents`) |
| `<version>` | Git tag, branch, or omit for latest |

## Local Package Cache

Downloaded packages are stored in a local cache directory so they do not need to be re-downloaded on subsequent installs. The cache is managed automatically by the CLI.

## Lock File

The `install` command maintains an `.agentspec.lock` file in the current directory. This file records the exact version and checksum of every installed package (including transitive dependencies) to ensure reproducible builds.

The lock file is a JSON file with the following structure:

```json
{
  "version": 1,
  "dependencies": [
    {
      "source": "github.com/org/my-agents",
      "version": "v1.2.0",
      "hash": "sha256:abc123...",
      "resolved_path": "/home/user/.agentspec/cache/..."
    }
  ]
}
```

## Examples

```bash
# Install a specific version
agentspec install github.com/org/my-agents@v1.2.0

# Install the latest version
agentspec install github.com/org/my-agents

# Install with verbose output to see dependency resolution
agentspec install -v github.com/org/shared-tools@v0.5.0
```

## Output

```
Resolving github.com/org/my-agents@v1.2.0...
Installed github.com/org/my-agents@v1.2.0 to /home/user/.agentspec/cache/...
  dependency: github.com/org/shared-tools@v0.3.0
Lock file updated.
```

## Exit Codes

| Code | Meaning |
|------|---------|
| `0` | Package installed successfully |
| `1` | An error occurred (resolution failure, network error, etc.) |

## See Also

- [CLI: publish](publish.md) -- Publish an AgentPack to a Git remote
- [CLI: validate](validate.md) -- Validate .ias files including imports
