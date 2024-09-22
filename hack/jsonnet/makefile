vendor:
	jb install

dashboards: vendor
	rm -rf out_dashboards
	mkdir out_dashboards
	jsonnet -J vendor -m out_dashboards dashboards.jsonnet | xargs -I {} sh -c 'yq -oy {} > {}.yaml'
	rm -r out_dashboards/*.json

rules: vendor
	rm -rf out_rules
	mkdir out_rules
	jsonnet -J vendor -m out_rules rules.jsonnet | xargs -I {} sh -c 'yq -oy {} > {}.yaml'
	rm -r out_rules/*.json

alerts: vendor
	rm -rf out_alerts
	mkdir out_alerts
	jsonnet -J vendor -m out_alerts alerts.jsonnet | xargs -I {} sh -c 'yq -oy {} > {}.yaml'
	rm -r out_alerts/*.json

tempo_dashboards: vendor
	rm -rf out_tempo_dashboards
	mkdir out_tempo_dashboards
	jsonnet -J vendor -m out_tempo_dashboards tempo-dashboards.jsonnet | xargs -I {} sh -c 'yq -oy {} > {}.yaml'
	rm -r out_tempo_dashboards/*.json

.PHONY: generate
generate: dashboards rules alerts

	
