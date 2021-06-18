// Copyright 2021 SAP SE
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ghworkflow

type workflow struct {
	Name string                  `yaml:"name"`
	On   map[string]eventTrigger `yaml:"on"`
	// A map of <job_id> and their configuration(s).
	Jobs map[string]job `yaml:"jobs"`
}

// eventTriggers contains rules about the events that that trigger a specific
// workflow.
// Ref: https://docs.github.com/en/actions/reference/events-that-trigger-workflows
type eventTrigger struct {
	Branches    []string `yaml:"branches"`
	PathsIgnore []string `yaml:"paths-ignore,omitempty"`
}

type job struct {
	// The name of the job displayed on GitHub.
	Name string `yaml:"name,omitempty"`

	// List of <job_id> that must complete successfully before this job will run.
	Needs []string `yaml:"needs,omitempty"`

	// The type of machine to run the job on.
	// Ref: https://docs.github.com/en/actions/using-github-hosted-runners/about-github-hosted-runners
	RunsOn string `yaml:"runs-on"`

	// A map of environment variables that are available to all steps in the job.
	Env map[string]string `yaml:"env,omitempty"`

	// You can use the if conditional to prevent a step from running unless a condition is met.
	If string `yaml:"if,omitempty"`

	// Steps can run commands, run setup tasks, or run an action. Not all steps
	// run actions, but all actions run as a step. Each step runs in its own
	// process in the runner environment and has access to the workspace and
	// filesystem.
	Steps []jobStep `yaml:"steps"`

	// Strategy creates a build matrix for the job and allows different
	// variations to run each job in.
	Strategy struct {
		Matrix struct {
			OS []string `yaml:"os"`
		} `yaml:"matrix"`
	} `yaml:"strategy,omitempty"`
}

func (j *job) addStep(s jobStep) {
	j.Steps = append(j.Steps, s)
}

// jobStep is a task that is run as part of a job.
type jobStep struct {
	// A unique identifier for the step. You can use the id to reference the
	// step in contexts.
	// Ref: https://docs.github.com/en/actions/reference/context-and-expression-syntax-for-github-actions
	ID string `yaml:"id,omitempty"`

	// A name for your step to display on GitHub.
	Name string `yaml:"name"`

	// Selects an action to run as part of a step in your job.
	//
	// It is strongly recommend that the version of the action be specified or
	// else it could break workflow when the action owner publishes an update.
	// Some actions require inputs that you must set using the with keyword.
	// Review the action's README file to determine the inputs required.
	Uses string `yaml:"uses,omitempty"`

	// A map of the input parameters defined by the action. Each input
	// parameter is a key/value pair. Input parameters are set as environment
	// variables. The variable is prefixed with INPUT_ and converted to upper
	// case.
	With map[string]interface{} `yaml:"with,omitempty"`

	// A map of environment variables for steps to use.
	Env map[string]string `yaml:"env,omitempty"`

	// Runs command-line programs using the operating system's shell.
	Run string `yaml:"run,omitempty"`
}
