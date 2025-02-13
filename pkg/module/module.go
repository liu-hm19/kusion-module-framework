package module

import (
	"context"
	"fmt"

	"gopkg.in/yaml.v2"
	"kusionstack.io/kusion/pkg/apis/core/v1"
	"kusionstack.io/kusion/pkg/apis/core/v1/workload"
	"kusionstack.io/kusion/pkg/log"
	"kusionstack.io/kusion/pkg/modules/proto"
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
	fwResources, err := f.Module.Generate(ctx, request)
	if err != nil {
		return nil, err
	}
	if fwResources == nil || fwResources.Resources == nil {
		log.Info("no resources generated by request:%v", request)
		return EmptyResponse(), nil
	}

	var resources [][]byte
	for _, res := range fwResources.Resources {
		out, err := yaml.Marshal(res)
		if err != nil {
			return nil, fmt.Errorf("marshal resource failed: %w. res:%v", err, res)
		}
		resources = append(resources, out)
	}
	return &proto.GeneratorResponse{
		Resources: resources,
	}, nil
}

type GeneratorRequest struct {
	// Project represents the project name
	Project string `json:"project,omitempty" yaml:"project"`
	// Stack represents the stack name
	Stack string `json:"stack,omitempty" yaml:"stack"`
	// App represents the application name, which is typically the same as the namespace of Kubernetes resources
	App string `json:"app,omitempty" yaml:"app"`
	// Workload represents the workload configuration
	Workload *workload.Workload `json:"workload,omitempty" yaml:"workload"`
	// DevModuleConfig is the developer's inputs of this module
	DevModuleConfig v1.Accessory `json:"dev_module_config,omitempty" yaml:"devModuleConfig"`
	// PlatformModuleConfig is the platform engineer's inputs of this module
	PlatformModuleConfig v1.GenericConfig `json:"platform_module_config,omitempty" yaml:"platformModuleConfig"`
	// RuntimeConfig is the runtime configurations defined in the workspace config
	RuntimeConfig *v1.RuntimeConfigs `json:"runtime_config,omitempty" yaml:"runtimeConfig"`
}

type GeneratorResponse struct {
	// Resources represents the generated resources
	Resources []v1.Resource `json:"resources,omitempty" yaml:"resources"`
}

func NewGeneratorRequest(req *proto.GeneratorRequest) (*GeneratorRequest, error) {

	log.Infof("module proto request received:%s", req.String())

	// validate workload
	if req.Workload == nil {
		return nil, fmt.Errorf("workload in the request is nil")
	}
	w := &workload.Workload{}
	if err := yaml.Unmarshal(req.Workload, w); err != nil {
		return nil, fmt.Errorf("unmarshal workload failed. %w", err)
	}

	var dc v1.Accessory
	if req.DevModuleConfig != nil {
		if err := yaml.Unmarshal(req.DevModuleConfig, &dc); err != nil {
			return nil, fmt.Errorf("unmarshal dev module config failed. %w", err)
		}
	}

	var pc v1.GenericConfig
	if req.PlatformModuleConfig != nil {
		if err := yaml.Unmarshal(req.PlatformModuleConfig, &pc); err != nil {
			return nil, fmt.Errorf("unmarshal platform module config failed. %w", err)
		}
	}

	var rc *v1.RuntimeConfigs
	if req.RuntimeConfig != nil {
		if err := yaml.Unmarshal(req.RuntimeConfig, rc); err != nil {
			return nil, fmt.Errorf("unmarshal runtime config failed. %w", err)
		}
	}

	result := &GeneratorRequest{
		Project:              req.Project,
		Stack:                req.Stack,
		App:                  req.App,
		Workload:             w,
		DevModuleConfig:      dc,
		PlatformModuleConfig: pc,
		RuntimeConfig:        rc,
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
