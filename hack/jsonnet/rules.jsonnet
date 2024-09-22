// Import and configuration
local rules = (import 'github.com/kubernetes-monitoring/kubernetes-mixin/mixin.libsonnet') + {
  _config+:: {
    cadvisorSelector: 'job="integrations/kubernetes/cadvisor"',
    kubeStateMetricsSelector: 'job="integrations/kubernetes/kube-state-metrics"',
    kubeletSelector: 'job="integrations/kubernetes/kubelet"',
  }
};

// PrometheusRule manifest
local newPrometheusRule(rules) = {
  apiVersion: 'monitoring.coreos.com/v1',
  kind: 'PrometheusRule',
  metadata: {
    name: "kubernetes-rules",
  },
  spec: rules,
};

// Generate Rules
{
  'rules.json': newPrometheusRule(rules.prometheusRules)
}
