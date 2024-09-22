package deployments

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	jb "github.com/jsonnet-bundler/jsonnet-bundler/pkg"
	jbfile "github.com/jsonnet-bundler/jsonnet-bundler/pkg/jsonnetfile"

	jsonnet "github.com/google/go-jsonnet"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

func DeployMixins(ctx context.Context, path string, kubeconfig string) error {

	vendorPath := filepath.Join(path, "vendor")

	err := InstallJsonnetDeps(path, vendorPath)
	if err != nil {
		return fmt.Errorf("could not install jsonnet dependencies: %v", err)
	}

	tmpdir, err := os.MkdirTemp("", "")
	if err != nil {
		return fmt.Errorf("could not create temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpdir)

	err = GenerateJsonnet(path, vendorPath, tmpdir)
	if err != nil {
		return fmt.Errorf("could not generate jsonnet file: %v", err)
	}

	err = ApplyKubernetesFiles(ctx, tmpdir, kubeconfig)
	if err != nil {
		return fmt.Errorf("could not apply jsonnet files: %v", err)
	}

	return nil
}

func InstallJsonnetDeps(path string, vendorPath string) error {
	// Build and deploy jsonnet files
	jf, err := jbfile.Load(filepath.Join(path, jbfile.File))
	if err != nil {
		return fmt.Errorf("failed to load jsonnet file: %v", err)
	}

	jfl, err := jbfile.Load(filepath.Join(path, jbfile.LockFile))
	if err != nil {
		return fmt.Errorf("failed to load lock jsonnet file: %v", err)
	}

	if err := os.MkdirAll(vendorPath, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create vendor path %s: %v", vendorPath, err)
	}
	if err := os.MkdirAll(filepath.Join(vendorPath, ".tmp"), os.ModePerm); err != nil {
		return fmt.Errorf("failed to create vendor path %s: %v", vendorPath, err)
	}

	_, err = jb.Ensure(jf, vendorPath, jfl.Dependencies)
	if err != nil {
		return fmt.Errorf("failed to load and install jsonnet dependencies: %v", err)
	}

	return nil
}

func GenerateJsonnet(path string, vendorpath string, outpath string) error {

	files := []string{
		"alerts.jsonnet",
		"dashboards.jsonnet",
		"rules.jsonnet",
	}

	vm := jsonnet.MakeVM()

	vm.Importer(&jsonnet.FileImporter{
		JPaths: []string{
			path,
			vendorpath,
		},
	})

	for _, f := range files {
		log.Printf("generating file from %s ...", f)

		m, err := vm.EvaluateFileMulti(f)
		if err != nil {
			return fmt.Errorf("could not evaluate file %s: %v", f, err)
		}

		for k, v := range m {
			log.Printf("writing file %s ...", k)
			if err := os.WriteFile(filepath.Join(outpath, k), []byte(v), 0644); err != nil {
				return fmt.Errorf("could not write file %s: %v", k, err)
			}
		}

	}

	return nil
}

func ApplyKubernetesFiles(ctx context.Context, path string, kubeconfig string) error {

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return fmt.Errorf("cloud not create kubernetes client config: %v", err)
	}

	kubeClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("cloud not create kubernetes client: %v", err)
	}

	discovery, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return fmt.Errorf("cloud not create discovery client: %v", err)
	}

	gr, err := restmapper.GetAPIGroupResources(discovery)
	if err != nil {
		return fmt.Errorf("could not create api group ressource mapper: %v", err)
	}

	mapper := restmapper.NewDiscoveryRESTMapper(gr)

	files, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("could not read directory entries: %v", err)
	}

	for _, fname := range files {

		p := filepath.Join(path, fname.Name())

		b, err := os.ReadFile(p)
		if err != nil {
			return fmt.Errorf("could not read file %s: %v", fname.Name(), err)
		}

		obj := &unstructured.Unstructured{}
		err = yaml.Unmarshal(b, obj)
		if err != nil {
			return fmt.Errorf("could not unmarshal file %s: %v", fname, err)
		}

		gvk := obj.GetObjectKind().GroupVersionKind()

		// Map object group version kind to api groupe
		mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return fmt.Errorf("no mapping found for kind and group version: %v", err)
		}

		o, err := kubeClient.Resource(mapping.Resource).Namespace("kube-system").Get(ctx, obj.GetName(), metav1.GetOptions{})
		if errors.IsNotFound(err) {
			log.Printf("create ressource: %s", p)

			_, err = kubeClient.Resource(mapping.Resource).Namespace("kube-system").Create(ctx, obj, metav1.CreateOptions{})
			if err != nil {
				return fmt.Errorf("could not create resource %s: %v", mapping.Resource, err)
			}
		} else {
			log.Printf("update ressource: %s", p)
			obj.SetResourceVersion(o.GetResourceVersion())

			_, err = kubeClient.Resource(mapping.Resource).Namespace("kube-system").Update(ctx, obj, metav1.UpdateOptions{})
			if err != nil {
				return fmt.Errorf("could not create resource %s: %v", mapping.Resource, err)
			}
		}

	}
	return nil
}
