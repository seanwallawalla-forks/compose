// +build kube

/*
   Copyright 2020 Docker Compose CLI authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package charts

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/compose-spec/compose-go/types"
	"github.com/docker/compose-cli/api/compose"
	"github.com/docker/compose-cli/api/context/store"
	"github.com/docker/compose-cli/kube/charts/helm"
	"github.com/docker/compose-cli/kube/charts/kubernetes"
	kubeutils "github.com/docker/compose-cli/kube/utils"
	chart "helm.sh/helm/v3/pkg/chart"
	util "helm.sh/helm/v3/pkg/chartutil"
	helmenv "helm.sh/helm/v3/pkg/cli"
)

// API defines management methods for helm charts
type API interface {
	GetDefaultEnv() *helmenv.EnvSettings
	Connect(ctx context.Context) error
	GenerateChart(project *types.Project, dirname string) error
	GetChartInMemory(project *types.Project) (*chart.Chart, error)
	SaveChart(project *types.Project, dest string) error

	Install(project *types.Project) error
	Uninstall(projectName string) error
	List(projectName string) ([]compose.Stack, error)
}

type sdk struct {
	h           *helm.HelmActions
	environment map[string]string
}

// sdk implement API
var _ API = sdk{}

func NewSDK(ctx store.KubeContext) (sdk, error) {
	return sdk{
		environment: kubeutils.Environment(),
		h:           helm.NewHelmActions(nil),
	}, nil
}

func (s sdk) Connect(ctx context.Context) error {
	return nil
}

// Install deploys a Compose stack
func (s sdk) Install(project *types.Project) error {
	chart, err := s.GetChartInMemory(project)
	if err != nil {
		return err
	}
	return s.h.InstallChart(project.Name, chart)
}

// Uninstall removes a runnign compose stack
func (s sdk) Uninstall(projectName string) error {
	return s.h.Uninstall(projectName)
}

// List returns a list of compose stacks
func (s sdk) List(projectName string) ([]compose.Stack, error) {
	return s.h.ListReleases()
}

// GetDefault initializes Helm EnvSettings
func (s sdk) GetDefaultEnv() *helmenv.EnvSettings {
	return helmenv.New()
}

func (s sdk) GetChartInMemory(project *types.Project) (*chart.Chart, error) {
	// replace _ with - in volume names
	for k, v := range project.Volumes {
		volumeName := strings.ReplaceAll(k, "_", "-")
		if volumeName != k {
			project.Volumes[volumeName] = v
			delete(project.Volumes, k)
		}
	}
	objects, err := kubernetes.MapToKubernetesObjects(project)
	if err != nil {
		return nil, err
	}
	//in memory files
	return helm.ConvertToChart(project.Name, objects)
}

func (s sdk) SaveChart(project *types.Project, dest string) error {
	chart, err := s.GetChartInMemory(project)
	if err != nil {
		return err
	}
	return util.SaveDir(chart, dest)
}

func (s sdk) GenerateChart(project *types.Project, dirname string) error {
	if strings.Contains(dirname, ".") {
		splits := strings.SplitN(dirname, ".", 2)
		dirname = splits[0]
	}

	dirname = filepath.Dir(dirname)
	return s.SaveChart(project, dirname)
}
