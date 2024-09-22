// Import dashboards and configuration
local dashboards = (import 'github.com/kubernetes-monitoring/kubernetes-mixin/mixin.libsonnet') + {
  _config+:: {
    grafanaK8s+:: {
      grafanaTimezone: 'browser',
      dashboardTags: ['kubernetes'],
    },
    showMultiCluster: true,
    cadvisorSelector: 'job="integrations/kubernetes/cadvisor"',
    kubeStateMetricsSelector: 'job="integrations/kubernetes/kube-state-metrics"',
    kubeletSelector: 'job="integrations/kubernetes/kubelet"',
  }
} + 
// Remove enwanted dashboards
{
  grafanaDashboards: std.mergePatch(super.grafanaDashboards, {
    'apiserver.json': null,
    // 'kubelet.json': null,
    'controller-manager.json': null,
    'scheduler.json': null,
    'proxy.json': null,
    'windows.json': null,
  })
}; 

// ConfigMap manifest
local newConfigMap(filename) = {
  local name = std.split(filename, ".")[0],
  
  apiVersion: 'v1',
  kind: 'ConfigMap',
  metadata: {
    name: 'dashboard-'+name,
    labels: {
      grafana_dashboard: '1'
    },
    annotations: {
      "k8s-sidecar-target-directory": "/tmp/dashboards/Kubernetes Dashboards"
    }
  },
  data: {
    [filename]: std.manifestJson(dashboards.grafanaDashboards[filename])
  },
};

// Generate dashboards ConfigMap
{ 
  [filename]: newConfigMap(filename) 
  for filename in std.objectFields(dashboards.grafanaDashboards)
} 
