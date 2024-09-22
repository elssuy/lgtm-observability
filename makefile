SHELL=/bin/bash

.PHONY: get-nginx-external-ip
get-nginx-external-ip:
	kubectl get -n ingress-nginx svc/ingress-nginx-controller -o=jsonpath='{.status.loadBalancer.ingress[0].ip}'

.PHONY: install-argocd
install-argocd:
	helm install argocd argo/argo-cd \
		--create-namespace \
		--namespace argocd \
		--version 7.4.5 \
		-f argocd/values.yaml

.PHONY: update-argocd
update-argocd:
	helm upgrade argocd argo/argo-cd \
		--namespace argocd \
		--version 7.4.5 \
		-f argocd/values.yaml

.PHONY: get-argocd-password
get-argocd-password:
	kubectl get -n argocd secret/argocd-initial-admin-secret -ojsonpath='{.data.password}'|base64 -d

.PHONY: deploy-requirements
deploy-requirements:
	find apps/requirements/* -type d -maxdepth 1 | xargs -I {} kubectl apply -f {}/application.yaml

.PHONY: deploy-lgtm
deploy-lgtm: update-config
	find apps/lgtm/* -type d -maxdepth 1 | xargs -I {} kubectl apply -f {}/application.yaml

.PHONY: deploy-security
deploy-security:
	find apps/security/* -type d -maxdepth 1 | xargs -I {} kubectl apply -f {}/application.yaml

.PHONY: destroy-lgtm
destroy-lgtm:
	find apps/lgtm/* -type d -maxdepth 1 | xargs -I {} kubectl delete -f {}/application.yaml

template-applications:
	sed " \
		s/{{\.TLD}}/${TLD}/; \
		s/{{\.MimirBlockBucketName}}/${MIMIR_BLOCK_BUCKET_NAME}/; \
		s/{{\.MimirRulerBucketName}}/${MIMIR_RULER_BUCKET_NAME}/; \
		s/{{\.MimirAlertManagerBucketName}}/${MIMIR_ALERTMANAGER_BUCKET_NAME}/; \
		s/{{\.LokiBucketName}}/${LOKI_BUCKET_NAME}/; \
		s/{{\.TempoBucketName}}/${TEMPO_BUCKET_NAME}/; \
		" \
		apps/lgtm/grafana/application-template.yaml > apps/lgtm/grafana/application.yaml
  
	sed " \
		s/{{\.TLD}}/${TLD}/; \
		s/{{\.MimirBlockBucketName}}/${MIMIR_BLOCK_BUCKET_NAME}/; \
		s/{{\.MimirRulerBucketName}}/${MIMIR_RULER_BUCKET_NAME}/; \
		s/{{\.MimirAlertManagerBucketName}}/${MIMIR_ALERTMANAGER_BUCKET_NAME}/; \
		s/{{\.LokiBucketName}}/${LOKI_BUCKET_NAME}/; \
		s/{{\.TempoBucketName}}/${TEMPO_BUCKET_NAME}/; \
		" \
		apps/lgtm/mimir/application-template.yaml > apps/lgtm/mimir/application.yaml

	sed " \
		s/{{\.TLD}}/${TLD}/; \
		s/{{\.MimirBlockBucketName}}/${MIMIR_BLOCK_BUCKET_NAME}/; \
		s/{{\.MimirRulerBucketName}}/${MIMIR_RULER_BUCKET_NAME}/; \
		s/{{\.MimirAlertManagerBucketName}}/${MIMIR_ALERTMANAGER_BUCKET_NAME}/; \
		s/{{\.LokiBucketName}}/${LOKI_BUCKET_NAME}/; \
		s/{{\.TempoBucketName}}/${TEMPO_BUCKET_NAME}/; \
		" \
		apps/lgtm/loki/application-template.yaml > apps/lgtm/loki/application.yaml
	sed " \
		s/{{\.TLD}}/${TLD}/; \
		s/{{\.MimirBlockBucketName}}/${MIMIR_BLOCK_BUCKET_NAME}/; \
		s/{{\.MimirRulerBucketName}}/${MIMIR_RULER_BUCKET_NAME}/; \
		s/{{\.MimirAlertManagerBucketName}}/${MIMIR_ALERTMANAGER_BUCKET_NAME}/; \
		s/{{\.LokiBucketName}}/${LOKI_BUCKET_NAME}/; \
		s/{{\.TempoBucketName}}/${TEMPO_BUCKET_NAME}/; \
		" \
		apps/lgtm/tempo/application-template.yaml > apps/lgtm/tempo/application.yaml
