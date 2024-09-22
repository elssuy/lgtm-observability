package deployments

import (
	"context"
	"fmt"
	"log"

	argoappv1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	argoclientset "github.com/argoproj/argo-cd/v2/pkg/client/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/kubectl/pkg/scheme"
)

type DeployObservabilityStackArgs struct {
	TLD string

	MimirBlockBucketName        string
	MimirRulerBucketName        string
	MimirAlertManagerBucketName string
	LokiBucketName              string
	TempoBucketName             string

	AwsAccessKeyId     string
	AwsSecretAccessKey string
}

func deploySecret(ctx context.Context, client *kubernetes.Clientset, secret *corev1.Secret) error {

	namespace := secret.GetNamespace()

	_, err := client.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
	if errors.IsAlreadyExists(err) {
		_, err = client.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	}

	return err
}

func createNsIfNotExist(ctx context.Context, kubeClient *kubernetes.Clientset, name string) error {

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}

	_, err := kubeClient.CoreV1().Namespaces().Get(ctx, ns.ObjectMeta.Name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = kubeClient.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	}
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("could not create namespace %s: %v", ns.ObjectMeta.Name, err)
	}
	return nil
}

func DeployObservabilityStack(ctx context.Context, kubeconfig string, args DeployObservabilityStackArgs) error {

	// Setup kubernetes clients
	err := argoappv1.AddToScheme(scheme.Scheme)
	if err != nil {
		return fmt.Errorf("could not add scheme: %v", err)
	}

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return fmt.Errorf("cloud not create kubernetes client config: %v", err)
	}

	argoClient, err := argoclientset.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("cloud not create argocd client: %v", err)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("cloud not create kubernetes client: %v", err)
	}

	// Create Namespaces: Loki Mimir Tempo for secrets

	err = createNsIfNotExist(ctx, kubeClient, "loki")
	if err != nil {
		return err
	}

	err = createNsIfNotExist(ctx, kubeClient, "tempo")
	if err != nil {
		return err
	}

	err = createNsIfNotExist(ctx, kubeClient, "mimir")
	if err != nil {
		return err
	}

	// Deploy secrets resources
	secretList := []string{
		"./apps/lgtm/mimir/secrets-template.yaml",
		"./apps/lgtm/loki/secrets-template.yaml",
		"./apps/lgtm/tempo/secrets-template.yaml",
	}

	for _, path := range secretList {
		log.Printf("Deploying: %s\n", path)

		secret, err := loadFromYamlFileAndTemplate[corev1.Secret](path, args)
		if err != nil {
			return fmt.Errorf("could not load and template application file %s: %v", path, err)
		}

		err = deploySecret(ctx, kubeClient, secret)
		if err != nil {
			return fmt.Errorf("could not deploy secret %s: %v", secret.GetObjectMeta().GetName(), err)
		}
	}

	// Deploy application resources
	appList := []string{
		"./apps/lgtm/grafana/application-template.yaml",
		"./apps/lgtm/mimir/application-template.yaml",
		"./apps/lgtm/loki/application-template.yaml",
		"./apps/lgtm/tempo/application-template.yaml",
		"./apps/lgtm/k8s-monitoring/application-template.yaml",
		"./apps/lgtm/alloy-ruler/application-template.yaml",
	}

	for _, path := range appList {
		log.Printf("Deploying: %s\n", path)

		app, err := loadFromYamlFileAndTemplate[argoappv1.Application](path, args)
		if err != nil {
			return fmt.Errorf("could not load and template application file %s: %v", path, err)
		}

		err = deployArgoCDApplication(ctx, argoClient, app)
		if err != nil {
			return fmt.Errorf("could not deploy application %s: %v", app.GetObjectMeta().GetName(), err)
		}
	}

	return nil

}
