local dashboards = (import 'github.com/grafana/tempo/operations/tempo-mixin/mixin.libsonnet') + {
  _config+:: {
    jobs+:: {
      gateway: 'nginx'
    }
  }

};


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
      "k8s-sidecar-target-directory": "/tmp/dashboards/Tempo Dashboards"
    }
  },
  data: {
    [filename]: std.manifestJson(dashboards.grafanaDashboards[filename])
  },
};


{
  [filename]: newConfigMap(filename)
  for filename in std.objectFields(dashboards.grafanaDashboards)
}
