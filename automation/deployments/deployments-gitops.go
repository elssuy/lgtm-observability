package deployments

import (
	"context"
	"fmt"
	"os"
	"time"

	helmclient "github.com/mittwald/go-helm-client"
	helmclientvalues "github.com/mittwald/go-helm-client/values"

	"helm.sh/helm/v3/pkg/repo"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/scheme"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argoclientset "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
)

func deployArgoCDApplication(ctx context.Context, argoClient *argoclientset.Clientset, app *argoappv1.Application) error {

	cApp, err := argoClient.ArgoprojV1alpha1().Applications("argocd").Get(ctx, *app.GetMetadata().Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = argoClient.ArgoprojV1alpha1().Applications("argocd").Create(ctx, app, metav1.CreateOptions{})
		return err
	}

	app.ResourceVersion = cApp.ResourceVersion

	_, err = argoClient.ArgoprojV1alpha1().Applications("argocd").Update(ctx, app, metav1.UpdateOptions{})

	return err
}

func deployPrometheusOperatorCrdApplication(ctx context.Context, client *argoclientset.Clientset) error {

	app, err := loadFromYamlFile[argoappv1.Application]("./apps/requirements/prometheus-crds/application.yaml")
	if err != nil {
		return fmt.Errorf("could not load prometheus crd application file: %v", err)
	}

	err = deployArgoCDApplication(ctx, client, app)
	if err != nil {
		return fmt.Errorf("could not deploy prometheus crd application: %v", err)
	}

	return nil
}

func deployNginxApplication(ctx context.Context, kubeClient *kubernetes.Clientset, argoClient *argoclientset.Clientset) (string, error) {

	app, err := loadFromYamlFile[argoappv1.Application]("./apps/requirements/ingress-nginx/application.yaml")
	if err != nil {
		return "", fmt.Errorf("could not load prometheus crd application file: %v", err)
	}

	err = deployArgoCDApplication(ctx, argoClient, app)
	if err != nil {
		return "", fmt.Errorf("could not create or update nginx application: %v", err)
	}

	// Whait for ingress loadbalancer to have an ip
	var svc *corev1.Service
	waitNginxIpSvcFunc := func(ctx context.Context) (bool, error) {
		// List services
		svc, err := kubeClient.CoreV1().Services("ingress-nginx").Get(ctx, "ingress-nginx-controller", metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}

		if len(svc.Status.LoadBalancer.Ingress) == 0 {
			return false, nil
		}

		if svc.Status.LoadBalancer.Ingress[0].IP != "" {
			return true, nil
		}

		return false, nil
	}
	err = wait.PollUntilContextTimeout(ctx, time.Second, time.Second*360*9, true, waitNginxIpSvcFunc)
	if err != nil {
		return "", fmt.Errorf("could pool nginx service status: %v", err)
	}

	svc, err = kubeClient.CoreV1().Services("ingress-nginx").Get(ctx, "ingress-nginx-controller", metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("could not get ingress-nginx-controller service: %s", err)
	}

	return svc.Status.LoadBalancer.Ingress[0].IP, nil
}

func deployOrUpdateArgoCDHelmChart(ctx context.Context, helmClient helmclient.Client, tld string) error {

	//
	// Deploy Argocd
	//

	chartRepos := map[string]string{
		"argo": "https://argoproj.github.io/argo-helm",
	}

	for name, url := range chartRepos {
		chartRepo := repo.Entry{
			Name: name,
			URL:  url,
		}
		if err := helmClient.AddOrUpdateChartRepo(chartRepo); err != nil {
			return fmt.Errorf("could not add or update chart repo [%s]:%s : %v", name, url, err)
		}
	}

	values, err := os.ReadFile("./argocd/values.yaml")
	if err != nil {
		return fmt.Errorf("could not read argocd value file: %v", err)
	}

	stringValues := []string{}
	if tld != "" {
		stringValues = append(stringValues, fmt.Sprintf("global.domain=argocd.%s", tld))
	}

	chartSpec := helmclient.ChartSpec{
		ReleaseName:     "argocd",
		Namespace:       "argocd",
		CreateNamespace: true,
		ChartName:       "argo/argo-cd",
		Version:         "7.4.5",
		ValuesYaml:      string(values),
		ValuesOptions: helmclientvalues.Options{
			StringValues: stringValues,
		},
	}

	if _, err := helmClient.InstallOrUpgradeChart(ctx, &chartSpec, nil); err != nil {
		return fmt.Errorf("could not install or update chart: %v", err)
	}

	return nil
}

func DeployGitOpsStack(ctx context.Context, kubeconfig string) (string, error) {
	// Setup clients
	err := argoappv1.AddToScheme(scheme.Scheme)
	if err != nil {
		return "", fmt.Errorf("could not add scheme: %v", err)
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return "", fmt.Errorf("cloud not create kubernetes client config: %v", err)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "", fmt.Errorf("cloud not create kubernetes client: %v", err)
	}

	argoClient, err := argoclientset.NewForConfig(config)
	if err != nil {
		return "", fmt.Errorf("cloud not create argocd client: %v", err)
	}

	clientOptions := &helmclient.KubeConfClientOptions{
		Options: &helmclient.Options{
			Namespace: "argocd",
		},
		KubeConfig: []byte(kubeconfig),
	}

	helmClient, err := helmclient.NewClientFromKubeConf(clientOptions)
	if err != nil {
		return "", fmt.Errorf("could not create Helm client: %s", err)
	}

	// Deploy argocd helm chart
	err = deployOrUpdateArgoCDHelmChart(ctx, helmClient, "")
	if err != nil {
		return "", fmt.Errorf("could not deploy argocd helm chart: %v", err)
	}

	// Deploy Prometheus Operator CRD Application
	err = deployPrometheusOperatorCrdApplication(ctx, argoClient)
	if err != nil {
		return "", fmt.Errorf("could not deploy prometheus operator crd application: %v", err)
	}

	// Deploy Nginx Application
	ip, err := deployNginxApplication(ctx, kubeClient, argoClient)
	if err != nil {
		return "", fmt.Errorf("could not deploy prometheus operator crd application: %v", err)
	}

	// Update Argocd Helm chart
	tld := fmt.Sprintf("%s.nip.io", ip)
	err = deployOrUpdateArgoCDHelmChart(ctx, helmClient, tld)
	if err != nil {
		return "", fmt.Errorf("could not deploy argocd helm chart: %v", err)
	}

	return tld, nil
}
