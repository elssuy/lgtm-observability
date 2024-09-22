package deployments

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"k8s.io/apimachinery/pkg/util/yaml"
)

func loadFromYamlFile[T any](path string) (*T, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read file: %v", err)
	}

	return loadFromByte[T](b)
}

func loadFromByte[T any](b []byte) (*T, error) {
	var app T

	err := yaml.Unmarshal(b, &app)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal yaml file: %v", err)
	}

	return &app, nil
}

func loadFromYamlFileAndTemplate[T any](path string, vars interface{}) (*T, error) {
	tpl, err := template.ParseFiles(path)
	if err != nil {
		return nil, fmt.Errorf("could not load template file %s: %v", path, err)
	}

	var content bytes.Buffer
	err = tpl.Execute(&content, vars)
	if err != nil {
		return nil, fmt.Errorf("could not template file %s: %v", path, err)
	}

	return loadFromByte[T](content.Bytes())
}
