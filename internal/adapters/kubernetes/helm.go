package kubernetes

import (
	"fmt"
	"os"
	"path/filepath"
)

// HelmChart holds generated Helm chart files.
type HelmChart struct {
	ChartYAML      string
	ValuesYAML     string
	DeploymentYAML string
	ServiceYAML    string
}

// GenerateHelmChart creates a Helm chart for a compiled agent.
func GenerateHelmChart(name, imageTag string, port int) *HelmChart {
	return &HelmChart{
		ChartYAML:      generateChartYAML(name),
		ValuesYAML:     generateValuesYAML(name, imageTag, port),
		DeploymentYAML: generateDeploymentTemplate(name),
		ServiceYAML:    generateServiceTemplate(name),
	}
}

// WriteHelmChart writes Helm chart files to the output directory.
func WriteHelmChart(chart *HelmChart, outDir string) error {
	templatesDir := filepath.Join(outDir, "templates")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		return fmt.Errorf("creating templates directory: %w", err)
	}

	files := map[string]string{
		filepath.Join(outDir, "Chart.yaml"):            chart.ChartYAML,
		filepath.Join(outDir, "values.yaml"):           chart.ValuesYAML,
		filepath.Join(templatesDir, "deployment.yaml"): chart.DeploymentYAML,
		filepath.Join(templatesDir, "service.yaml"):    chart.ServiceYAML,
	}

	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}

	return nil
}

func generateChartYAML(name string) string {
	return fmt.Sprintf(`apiVersion: v2
name: %s
description: AgentSpec-compiled agent Helm chart
type: application
version: 0.1.0
appVersion: "1.0.0"
`, name)
}

func generateValuesYAML(name, imageTag string, port int) string {
	// Parse image and tag
	image := imageTag
	tag := "latest"
	if i := len(imageTag) - 1; i > 0 {
		for j := i; j >= 0; j-- {
			if imageTag[j] == ':' {
				image = imageTag[:j]
				tag = imageTag[j+1:]
				break
			}
		}
	}

	return fmt.Sprintf(`replicaCount: 1

image:
  repository: %s
  pullPolicy: IfNotPresent
  tag: "%s"

service:
  type: ClusterIP
  port: %d

containerPort: %d

resources:
  limits:
    cpu: 500m
    memory: 256Mi
  requests:
    cpu: 100m
    memory: 128Mi

livenessProbe:
  httpGet:
    path: /healthz
    port: http
  initialDelaySeconds: 5
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /healthz
    port: http
  initialDelaySeconds: 3
  periodSeconds: 5

autoscaling:
  enabled: false
  minReplicas: 1
  maxReplicas: 10
  targetCPUUtilizationPercentage: 80
`, image, tag, port, port)
}

func generateDeploymentTemplate(name string) string {
	return fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "%s.fullname" . }}
  labels:
    app.kubernetes.io/name: {{ include "%s.name" . }}
    app.kubernetes.io/managed-by: {{ .Release.Service }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels:
      app.kubernetes.io/name: {{ include "%s.name" . }}
  template:
    metadata:
      labels:
        app.kubernetes.io/name: {{ include "%s.name" . }}
    spec:
      containers:
        - name: {{ .Chart.Name }}
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          ports:
            - name: http
              containerPort: {{ .Values.containerPort }}
              protocol: TCP
          livenessProbe:
            {{- toYaml .Values.livenessProbe | nindent 12 }}
          readinessProbe:
            {{- toYaml .Values.readinessProbe | nindent 12 }}
          resources:
            {{- toYaml .Values.resources | nindent 12 }}
`, name, name, name, name)
}

func generateServiceTemplate(name string) string {
	return fmt.Sprintf(`apiVersion: v1
kind: Service
metadata:
  name: {{ include "%s.fullname" . }}
  labels:
    app.kubernetes.io/name: {{ include "%s.name" . }}
    app.kubernetes.io/managed-by: {{ .Release.Service }}
spec:
  type: {{ .Values.service.type }}
  ports:
    - port: {{ .Values.service.port }}
      targetPort: http
      protocol: TCP
      name: http
  selector:
    app.kubernetes.io/name: {{ include "%s.name" . }}
`, name, name, name)
}
