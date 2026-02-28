package telemetry

// CompileTags returns standard tags for a compilation operation span.
func CompileTags(target, name string) map[string]string {
	return map[string]string{
		"operation": "compile",
		"target":    target,
		"name":      name,
	}
}

// EvalTags returns standard tags for an eval operation span.
func EvalTags(agentName, caseName string) map[string]string {
	return map[string]string{
		"operation": "eval",
		"agent":     agentName,
		"case":      caseName,
	}
}

// PackageTags returns standard tags for a package operation span.
func PackageTags(format, name string) map[string]string {
	return map[string]string{
		"operation": "package",
		"format":    format,
		"name":      name,
	}
}

// PublishTags returns standard tags for a publish operation span.
func PublishTags(packageName, version string) map[string]string {
	return map[string]string{
		"operation": "publish",
		"package":   packageName,
		"version":   version,
	}
}

// InstallTags returns standard tags for an install operation span.
func InstallTags(source, version string) map[string]string {
	return map[string]string{
		"operation": "install",
		"source":    source,
		"version":   version,
	}
}
