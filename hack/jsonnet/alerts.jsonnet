// Import and configuration
local alerts = (import 'github.com/kubernetes-monitoring/kubernetes-mixin/mixin.libsonnet') + {
  _config+:: {
    cadvisorSelector: 'job="integrations/kubernetes/cadvisor"',
    kubeStateMetricsSelector: 'job="integrations/kubernetes/kube-state-metrics"',
    kubeletSelector: 'job="integrations/kubernetes/kubelet"',
  }
};


// Rules to exclude
local rulePatches = {
  excludedRules: [
    {name: 'kubernetes-system-apiserver',           rules: [{ alert: 'KubeAPIDown'}]},
    {name: 'kubernetes-system-controller-manager',  rules: [{ alert: 'KubeControllerManagerDown'}]},
    {name: 'kubernetes-system-kube-proxy',          rules: [{ alert: 'KubeProxyDown'}]},
    {name: 'kubernetes-system-scheduler',           rules: [{ alert: 'KubeSchedulerDown'}]},
  ],
};

local sanitizePrometheusRules = (import 'github.com/prometheus-operator/kube-prometheus/jsonnet/kube-prometheus/lib/rule-sanitizer.libsonnet')(rulePatches).sanitizePrometheusRules;

// PrometheusRule manifest
local newPrometheusRule(alerts) = {
  apiVersion: 'monitoring.coreos.com/v1',
  kind: 'PrometheusRule',
  metadata: {
    name: "kubernetes-alerts",
  },
  spec: alerts,
};

// Generate Alerts
sanitizePrometheusRules({'alerts.json': newPrometheusRule(alerts.prometheusAlerts)})
