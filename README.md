# Demo Deployment - Scaleway

The demo deployment uses pulumi, golang, and scaleway cloud. You need to setup your pulumi backend first.
This backend can be local: `file:///path/to/your/local/backend`

Setup environment variables:

```bash
# Pulumi backend
export PULUMI_BACKEND_URL="file:///path/to/your/local/backend/folder"

# API Credentials
export SCW_ACCESS_KEY=""
export SCW_SECRET_KEY=""

# General project informations
export SCW_DEFAULT_ORGANIZATION_ID=""
export SCW_DEFAULT_PROJECT_ID=""
export SCW_DEFAULT_REGION=""
export SCW_DEFAULT_ZONE=""
```

Then once thoses variables are setup, run deployment script:

```bash
go run automation/cmd/up/main.go
```

You should see the output message:
```bash
# [...]
2024/09/16 16:19:00 ==== Outputs ====
Name                 Value
tld                  ...
argocd-url           ...
grafana-url          ...
mimir-block-bucket   ...
mimir-ruler-bucket   ...
mimir-alert-bucket   ...
loki-bucket          ...
tempo-bucket         ...
```

Grafana user and password is `admin`

This script will export the admin kubeconfig file and name it `kubeconfig-admin-cluster.yaml`. 
You can access it like so:
```bash
export KUBECONFIG="./kubeconfig-admin-cluster.yaml"

kubectl get no
```

Use a tool like [KubeCM](https://github.com/sunny0826/kubecm) to manage your kubeconfig.
```bash
kubecm add -f ./kubeconfig-admin-cluster.yaml --context-name admin
kubecm sw admin
kubectl get no
```

## Destroy cluster

To destroy all ressources, simply run the destroy script:

```bash
go run automation/cmd/down/main.go
```

# Manual Deployment

We use argocd as source of truth for application that are deployed in the cluster. 
So it is deployed first and updated later to expose it's ingress.

# 1. Install HA ArgoCD

To inspect what config values was modified use thoses commands:

For bash users
```bash
vimdiff <(helm show values argo/argo-cd) argocd/values.yaml
```

For fish users
```fish
vimdiff (helm show values argo/argo-cd | psub ) argocd/values.yaml
```

Search latest ArgoCD version:
```bash
helm repo add argo https://argoproj.github.io/argo-helm
helm repo update
helm search repo argo/argo-cd

NAME        	CHART VERSION	APP VERSION	DESCRIPTION
argo/argo-cd	7.4.5        	v2.12.2    	A Helm chart for Argo CD, a declarative, GitOps...
```

Update ArgoCD Version inside `makefile`: 
```makefile
install-argocd:
	helm install argocd argo/argo-cd \
		--create-namespace \
		--namespace argocd \
		--version 7.4.5 \       # <- Here
		-f argocd/values.yaml

update-argocd:
	helm upgrade argocd argo/argo-cd \
		--namespace argocd \
		--version 7.4.5 \       # <- Here
		-f argocd/values.yaml
```

And then install argocd with:
```bash
make install-argocd
```

# 2. Apply ArgoCD requirements applications

Deploy requirements application and get nginx external ip:
```bash
make deploy-requirements
make get-nginx-external-ip

xxx.xxx.xxx.xxx
```

# 3. Update ArgoCD with ingress external ip

Update ArgoCD `values.yaml` file to update the ingress value:
```yaml
# [...]
## Globally shared configuration
global:
  # -- Default domain used by all components
  ## Used for ingresses, certificates, SSO, notifications, etc.
  domain: argocd.xxx.xxx.xxx.xxx.nip.io
# [...]
```

> [!NOTE]
> We are using [nip.io](https://nip.io/) services to get a dns resolution for our ip.
> Feel free to use anything else.

Then update argocd deployment:
```bash
make update-argocd
```

# 4. Required: S3 Storage

## On scaleway

For the LGTM stack to work you will need S3 buckets and credentials:

- One bucket for each Mimir component
  - One bucket for Alertmanager
  - One bucket for Ruler services
  - One bucket for Data Block
- One bucket for Loki
- One bucket for Tempo
- One or multiple API keys for each bucket

> [!IMPORTANT]  
> Keep your bucket on the same region as your kubernetes cluster.

Create your buckets [doc here](https://www.scaleway.com/en/docs/storage/object/how-to/create-a-bucket/)

Create an application for your stack [doc here](https://www.scaleway.com/en/docs/identity-and-access-management/iam/how-to/create-application/).
Then generate an API Key for this application [doc here](https://www.scaleway.com/en/docs/identity-and-access-management/iam/how-to/create-api-keys/).

Create and attach the right policies to manage the bucket you've created [doc here](https://www.scaleway.com/en/docs/identity-and-access-management/iam/how-to/create-policy/)

You should end up with credentials, put them in the `secret-template.yaml` resource file
in each folder in `apps/lgtm/{loki, mimir, tempo}`

Here is the `secret.yaml` file:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: s3-credentials
type: Opaque
stringData:
  AWS_ACCESS_KEY_ID: SCWxxxxxxxxxxxxxxxxx
  AWS_SECRET_ACCESS_KEY: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
````

Deploy it on each namespace:
```bash
kubectl create ns loki
kubectl create ns mimir
kubectl create ns tempo

kubectl apply -f secret.yaml -n loki
kubectl apply -f secret.yaml -n mimir
kubectl apply -f secret.yaml -n tempo
```

# 5. Deploy the monitoring stack

> [!NOTE]
> A makefile command `make template-applications` is available to make this process more easy
> Export required environment variables `TLD` `MIMIR_BLOCK_BUCKET_NAME`
> `` `` `` `` ``and run the makefile command.

Export required environment variables:
```bash
export TLD=""
export MIMIR_BLOCK_BUCKET_NAME=""
export MIMIR_RULER_BUCKET_NAME=""
export MIMIR_ALERTMANAGER_BUCKET_NAME=""
export LOKI_BUCKET_NAME=""
export TEMPO_BUCKET_NAME=""
```

Run the template makefile command:
```bash
make tempalte-applications
```

You should endup with templated files for grafana, mimir, loki and tempo in `/apps/lgtm/`

Run the make file command to deploy the monitoring stack:
```bash
make deploy-lgtm
```

# 5. Access dashboards

You get access to grafana dashboards via the url: `grafana.<ip>.nip.io`. User and Password is: `admin`

You get access to argocd dashboards via the url: `argocd.<ip>.nip.io`. User is `admin` and password can be get
via `make get-argocd-password`


# Unstructured Notes

The current state of k8s-monitoring helm chart doen't support rules sync for Mimir and Loki:
https://github.com/grafana/k8s-monitoring-helm/pull/568
So we have to deploy our own config for it.

Issues for Secret Reference in case we whant to use external s3 storage for Loki
https://github.com/grafana/loki/pull/12652

Grafana Operator is a bad idea because they will not support organization. They suggest to use simple grafana instance with good CI/CD:
https://grafana.github.io/grafana-operator/docs/grafana/#organizations


Lots of examples for using k8s cli library:
https://github.com/iximiuz/client-go-examples
https://github.com/iximiuz/client-go-examples/blob/main/cli-runtime-printers/main.go


All mixins are listed here:
https://github.com/prometheus-operator/kube-prometheus/blob/main/jsonnet/kube-prometheus/jsonnetfile.json
Examples here:
https://github.com/prometheus-operator/kube-prometheus/tree/main/examples

We have to compile them by ourselvs

Tanka is needed for tns stack:
https://tanka.dev/install/#tanka
