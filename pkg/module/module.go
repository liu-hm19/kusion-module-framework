package module

import (
	"context"
	"errors"
	"fmt"

	yamlv2 "gopkg.in/yaml.v2"
	"gopkg.in/yaml.v3"
	v1 "kusionstack.io/kusion-api-go/api.kusion.io/v1"
	"kusionstack.io/kusion-module-framework/pkg/log"
	"kusionstack.io/kusion-module-framework/pkg/module/proto"
)

type FrameworkModule interface {
	Generate(ctx context.Context, req *GeneratorRequest) (*GeneratorResponse, error)
}

// FrameworkModuleWrapper is a module that implements the proto Module interface.
// It wraps a dev-centric FrameworkModule into a proto Module
type FrameworkModuleWrapper struct {
	// Module is the actual FrameworkModule implemented by platform engineers
	Module FrameworkModule
}

func (f *FrameworkModuleWrapper) Generate(ctx context.Context, req *proto.GeneratorRequest) (*proto.GeneratorResponse, error) {
	request, err := NewGeneratorRequest(req)
	if err != nil {
		return nil, err
	}
	response, err := f.Module.Generate(ctx, request)
	if err != nil {
		return nil, err
	}
	if response == nil {
		log.Info("no resources generated by request:%v", request)
		return EmptyResponse(), nil
	}

	// marshal resources and patcher
	var resources [][]byte
	for _, res := range response.Resources {
		out, err := yaml.Marshal(res)
		if err != nil {
			return nil, fmt.Errorf("marshal resource failed: %w. res:%v", err, res)
		}
		resources = append(resources, out)
	}

	var patcher []byte
	if response.Patcher != nil {
		patcher, err = yaml.Marshal(response.Patcher)
		if err != nil {
			return nil, fmt.Errorf("marshal patcher failed: %w. patcher:%v", err, patcher)
		}
	}

	return &proto.GeneratorResponse{
		Resources: resources,
		Patcher:   patcher,
	}, nil
}

type GeneratorRequest struct {
	// Project represents the project name
	Project string `json:"project" yaml:"project"`
	// Stack represents the stack name
	Stack string `json:"stack" yaml:"stack"`
	// App represents the application name, which is typically the same as the namespace of Kubernetes resources
	App string `json:"app" yaml:"app"`
	// Workload represents the workload configuration
	Workload v1.Accessory `json:"workload,omitempty" yaml:"workload,omitempty"`
	// DevConfig is the developer's inputs of this module
	DevConfig v1.Accessory `json:"devConfig,omitempty" yaml:"devConfig,omitempty"`
	// PlatformConfig is the platform engineer's inputs of this module
	PlatformConfig v1.GenericConfig `json:"platformConfig,omitempty" yaml:"platformConfig,omitempty"`
	// Context contains workspace-level configurations, such as topologies, server endpoints, metadata, etc.
	Context v1.GenericConfig `yaml:"context,omitempty" json:"context,omitempty"`
	// SecretStore represents a secure external location for storing secrets.
	SecretStore v1.SecretStore `yaml:"secretStore,omitempty" json:"secretStore,omitempty"`
}

type GeneratorResponse struct {
	// Resources represents the generated resources
	Resources []v1.Resource `json:"resources,omitempty" yaml:"resources,omitempty"`
	Patcher   *v1.Patcher   `json:"patcher,omitempty" yaml:"patcher,omitempty"`
}

func NewGeneratorRequest(req *proto.GeneratorRequest) (*GeneratorRequest, error) {
	log.Infof("module proto request received:%s", req.String())

	// validate generator request
	if req == nil {
		return nil, errors.New("empty generator request")
	}

	// validate workload
	var w v1.Accessory
	if req.Workload != nil {
		if err := yamlv2.Unmarshal(req.Workload, &w); err != nil {
			return nil, fmt.Errorf("unmarshal workload failed. %w", err)
		}
	}

	var dc v1.Accessory
	if req.DevConfig != nil {
		if err := yaml.Unmarshal(req.DevConfig, &dc); err != nil {
			return nil, fmt.Errorf("unmarshal dev config failed. %w", err)
		}
	}

	var pc v1.GenericConfig
	if req.PlatformConfig != nil {
		if err := yaml.Unmarshal(req.PlatformConfig, &pc); err != nil {
			return nil, fmt.Errorf("unmarshal platform module config failed. %w", err)
		}
	}

	var ctx v1.GenericConfig
	if req.Context != nil {
		if err := yaml.Unmarshal(req.Context, &ctx); err != nil {
			return nil, fmt.Errorf("unmarshal context failed. %w", err)
		}
	}

	var secretStore v1.SecretStore
	if req.SecretStore != nil {
		if err := yaml.Unmarshal(req.SecretStore, &secretStore); err != nil {
			return nil, fmt.Errorf("unmarshal secret store failed. %w", err)
		}
	}

	result := &GeneratorRequest{
		Project:        req.Project,
		Stack:          req.Stack,
		App:            req.App,
		Workload:       w,
		DevConfig:      dc,
		PlatformConfig: pc,
		Context:        ctx,
		SecretStore:    secretStore,
	}
	out, err := yaml.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("marshal new generator request failed. %w", err)
	}
	log.Infof("new generator request:%s", string(out))
	return result, nil
}

// EmptyResponse represents a legal but empty response. Interfaces should return an EmptyResponse instead of nil when the response is empty
func EmptyResponse() *proto.GeneratorResponse {
	return &proto.GeneratorResponse{}
}
