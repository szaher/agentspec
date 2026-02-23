// Package kubernetes implements the Kubernetes adapter for the AgentSpec toolchain.
package kubernetes

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/szaher/designs/agentz/internal/ir"
)

// Manifests holds generated Kubernetes resource manifests.
type Manifests struct {
	Namespace  map[string]interface{}
	Deployment map[string]interface{}
	Service    map[string]interface{}
	ConfigMap  map[string]interface{}
	HPA        map[string]interface{}
}

// GenerateManifests creates Kubernetes manifests from IR resources and deploy target config.
func GenerateManifests(resources []ir.Resource, config map[string]interface{}) *Manifests {
	namespace := stringFromConfig(config, "namespace", "default")
	replicas := intFromConfig(config, "replicas", 1)
	port := intFromConfig(config, "port", 8080)
	image := stringFromConfig(config, "image", "agentspec-runtime:latest")

	name := "agentspec-runtime"

	labels := map[string]interface{}{
		"app.kubernetes.io/name":       name,
		"app.kubernetes.io/managed-by": "agentspec",
	}

	m := &Manifests{}

	// Namespace
	if namespace != "default" {
		m.Namespace = map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name":   namespace,
				"labels": labels,
			},
		}
	}

	// ConfigMap with runtime config
	configData, _ := json.Marshal(resources)
	m.ConfigMap = map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      name + "-config",
			"namespace": namespace,
			"labels":    labels,
		},
		"data": map[string]interface{}{
			"runtime-config.json": string(configData),
		},
	}

	// Deployment
	m.Deployment = map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
			"labels":    labels,
		},
		"spec": map[string]interface{}{
			"replicas": replicas,
			"selector": map[string]interface{}{
				"matchLabels": map[string]interface{}{
					"app.kubernetes.io/name": name,
				},
			},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": labels,
				},
				"spec": map[string]interface{}{
					"containers": []map[string]interface{}{
						{
							"name":  name,
							"image": image,
							"ports": []map[string]interface{}{
								{
									"containerPort": port,
									"protocol":      "TCP",
								},
							},
							"livenessProbe": map[string]interface{}{
								"httpGet": map[string]interface{}{
									"path": "/healthz",
									"port": port,
								},
								"initialDelaySeconds": 5,
								"periodSeconds":       10,
							},
							"readinessProbe": map[string]interface{}{
								"httpGet": map[string]interface{}{
									"path": "/healthz",
									"port": port,
								},
								"initialDelaySeconds": 3,
								"periodSeconds":       5,
							},
							"volumeMounts": []map[string]interface{}{
								{
									"name":      "config",
									"mountPath": "/app",
									"readOnly":  true,
								},
							},
						},
					},
					"volumes": []map[string]interface{}{
						{
							"name": "config",
							"configMap": map[string]interface{}{
								"name": name + "-config",
							},
						},
					},
				},
			},
		},
	}

	// Add resource limits if specified
	if res, ok := config["resources"]; ok {
		if resMap, ok := res.(map[string]interface{}); ok {
			spec, _ := m.Deployment["spec"].(map[string]interface{})
			tmpl, _ := spec["template"].(map[string]interface{})
			tmplSpec, _ := tmpl["spec"].(map[string]interface{})
			containers, _ := tmplSpec["containers"].([]map[string]interface{})
			if len(containers) > 0 {
				containers[0]["resources"] = resMap
			}
		}
	}

	// Service
	m.Service = map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
			"labels":    labels,
		},
		"spec": map[string]interface{}{
			"selector": map[string]interface{}{
				"app.kubernetes.io/name": name,
			},
			"ports": []map[string]interface{}{
				{
					"port":       port,
					"targetPort": port,
					"protocol":   "TCP",
				},
			},
			"type": "ClusterIP",
		},
	}

	// HPA if autoscale config exists
	if autoscale, ok := config["autoscale"]; ok {
		if asMap, ok := autoscale.(map[string]interface{}); ok {
			minReplicas := intFromMap(asMap, "min_replicas", 1)
			maxReplicas := intFromMap(asMap, "max_replicas", 10)
			targetCPU := intFromMap(asMap, "target_cpu", 80)

			m.HPA = map[string]interface{}{
				"apiVersion": "autoscaling/v2",
				"kind":       "HorizontalPodAutoscaler",
				"metadata": map[string]interface{}{
					"name":      name,
					"namespace": namespace,
					"labels":    labels,
				},
				"spec": map[string]interface{}{
					"scaleTargetRef": map[string]interface{}{
						"apiVersion": "apps/v1",
						"kind":       "Deployment",
						"name":       name,
					},
					"minReplicas": minReplicas,
					"maxReplicas": maxReplicas,
					"metrics": []map[string]interface{}{
						{
							"type": "Resource",
							"resource": map[string]interface{}{
								"name": "cpu",
								"target": map[string]interface{}{
									"type":               "Utilization",
									"averageUtilization": targetCPU,
								},
							},
						},
					},
				},
			}
		}
	}

	return m
}

// WriteManifests writes all Kubernetes manifests to the output directory.
func WriteManifests(m *Manifests, outDir string) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return err
	}

	write := func(name string, obj map[string]interface{}) error {
		if obj == nil {
			return nil
		}
		data, err := json.MarshalIndent(obj, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal %s: %w", name, err)
		}
		data = append(data, '\n')
		return os.WriteFile(filepath.Join(outDir, name+".json"), data, 0644)
	}

	if err := write("namespace", m.Namespace); err != nil {
		return err
	}
	if err := write("configmap", m.ConfigMap); err != nil {
		return err
	}
	if err := write("deployment", m.Deployment); err != nil {
		return err
	}
	if err := write("service", m.Service); err != nil {
		return err
	}
	if err := write("hpa", m.HPA); err != nil {
		return err
	}
	return nil
}

func stringFromConfig(config map[string]interface{}, key, defaultVal string) string {
	if config == nil {
		return defaultVal
	}
	if v, ok := config[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}

func intFromConfig(config map[string]interface{}, key string, defaultVal int) int {
	if config == nil {
		return defaultVal
	}
	return intFromMap(config, key, defaultVal)
}

func intFromMap(m map[string]interface{}, key string, defaultVal int) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return defaultVal
}
