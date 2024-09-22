package plm

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

type StackDependency struct {
	Name    string
	Version string
	URL     string
}

type StackConfig struct {
	StackName    string
	ProjectName  string
	Config       map[string]string
	Dependencies []StackDependency
}

type Layer[R any] struct {
	Program     func(ctx *pulumi.Context) error
	StackConfig StackConfig

	stack auto.Stack
}

func NewLayer[R any](ctx context.Context, c StackConfig, p func(ctx *pulumi.Context) error) (Layer[R], error) {

	l := Layer[R]{
		Program:     p,
		StackConfig: c,
	}

	err := l.Init(ctx)

	return l, err
}

func (l *Layer[R]) Init(ctx context.Context) error {
	log.Printf("initializing layer %s/%s", l.StackConfig.StackName, l.StackConfig.ProjectName)

	// Upsert stack
	s, err := auto.UpsertStackInlineSource(ctx, l.StackConfig.StackName, l.StackConfig.ProjectName, l.Program)
	if err != nil {
		return fmt.Errorf("Error while upserting stack: %v", err)
	}
	l.stack = s

	// Set configs values
	if err := l.SetConfig(ctx); err != nil {
		return fmt.Errorf("error while setting stack config values: %v", err)
	}

	// Install plugins
	if err := l.InstallPlugins(ctx); err != nil {
		return fmt.Errorf("error while installing plugins: %v", err)
	}

	return err
}

func (l *Layer[R]) GetOutputs(ctx context.Context) (*R, error) {
	output, err := l.stack.Outputs(ctx)
	if err != nil {
		return nil, err
	}

	// Unwrap values
	uo := make(map[string]interface{})
	for k, v := range output {
		uo[k] = v.Value
	}

	// Marshal to json for destructuring
	var r R
	jsonData, err := json.Marshal(uo)
	if err != nil {
		return nil, fmt.Errorf("could not marshal outputs: %v", err)
	}

	// Unmarshal to typesafe structure
	err = json.Unmarshal(jsonData, &r)
	if err != nil {
		return nil, fmt.Errorf("could unmarshal outputs: %v", err)
	}

	return &r, nil
}

func (l *Layer[R]) SetConfig(ctx context.Context) error {
	for k, v := range l.StackConfig.Config {
		err := l.stack.SetConfig(ctx, k, auto.ConfigValue{Value: v})
		if err != nil {
			return fmt.Errorf("could not set value %s: %v", k, err)
		}
	}

	return nil
}

func (l *Layer[R]) Up(ctx context.Context) error {
	if err := l.Refresh(ctx); err != nil {
		return err
	}

	log.Printf("upping layer %s/%s", l.StackConfig.StackName, l.StackConfig.ProjectName)

	stdoutStreamer := optup.ProgressStreams(os.Stdout)
	_, err := l.stack.Up(ctx, stdoutStreamer)
	if err != nil {
		return fmt.Errorf("Error while uping stack: %v", err)
	}

	return nil
}

func (l *Layer[R]) Down(ctx context.Context) error {
	if err := l.Refresh(ctx); err != nil {
		return err
	}

	log.Printf("downing layer %s/%s", l.StackConfig.StackName, l.StackConfig.ProjectName)

	stdoutStreamer := optdestroy.ProgressStreams(os.Stdout)
	_, err := l.stack.Destroy(ctx, stdoutStreamer)
	if err != nil {
		return fmt.Errorf("Error while downing stack: %v", err)
	}

	return nil
}

func (l *Layer[R]) InstallPlugins(ctx context.Context) error {
	log.Printf("installing plugins for layer %s/%s", l.StackConfig.StackName, l.StackConfig.ProjectName)

	w := l.stack.Workspace()
	for _, d := range l.StackConfig.Dependencies {
		log.Printf("installing %s@%s from %s...\n", d.Name, d.Version, d.URL)
		err := w.InstallPluginFromServer(ctx, d.Name, d.Version, d.URL)
		if err != nil {
			return fmt.Errorf("could not install plugin %s@%s from %s: %v", d.Name, d.Version, d.URL, err)
		}
	}

	return nil
}

func (l *Layer[R]) Refresh(ctx context.Context) error {
	log.Printf("refreshing layer %s/%s", l.StackConfig.StackName, l.StackConfig.ProjectName)
	if _, err := l.stack.Refresh(ctx); err != nil {
		return fmt.Errorf("Error while refreshing stack: %v", err)
	}

	return nil
}
