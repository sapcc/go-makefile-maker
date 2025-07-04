// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package ghworkflow

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sapcc/go-bits/must"
)

func newWorkflow(name, defaultBranch string, ignorePaths []string) *workflow {
	return &workflow{
		Name: name,
		On:   pushAndPRTriggers(defaultBranch, ignorePaths),
		Permissions: permissions{
			Contents: tokenScopeRead, // for actions/checkout to fetch code
		},
	}
}

type workflow struct {
	Name string       `yaml:"name"`
	On   eventTrigger `yaml:"on"`
	// Permissions modify the default permissions granted to the GITHUB_TOKEN. If you
	// specify the access for any of the scopes, all of those that are not specified are
	// set to 'none'.
	Permissions permissions `yaml:"permissions"`
	// A map of <job_id> to their configuration(s).
	Jobs map[string]job `yaml:"jobs"`
}

func (w workflow) getPath() string {
	fileName := strings.ToLower(strings.ReplaceAll(w.Name, " ", "-"))
	return filepath.Join(workflowDir, fileName+".yaml")
}

func (w workflow) deleteIf(condition bool) bool {
	if !condition {
		must.Succeed(os.RemoveAll(w.getPath()))
		return true
	}

	return false
}

type githubTokenScope string

const (
	tokenScopeNone  = "none" //nolint:deadcode,varcheck // this exists for documentation purposes
	tokenScopeRead  = "read"
	tokenScopeWrite = "write"
)

type permissions struct {
	Actions        githubTokenScope `yaml:"actions,omitempty"`
	Checks         githubTokenScope `yaml:"checks,omitempty"`
	Contents       githubTokenScope `yaml:"contents,omitempty"`
	Packages       githubTokenScope `yaml:"packages,omitempty"`
	SecurityEvents githubTokenScope `yaml:"security-events,omitempty"`
}

// eventTriggers contains rules about the events that trigger a specific
// workflow.
// Ref: https://docs.github.com/en/actions/reference/events-that-trigger-workflows
type eventTrigger struct {
	Push             pushAndPRTriggerOpts `yaml:"push,omitempty"`
	PullRequest      pushAndPRTriggerOpts `yaml:"pull_request,omitempty"`
	Schedule         []cronExpr           `yaml:"schedule,omitempty"`
	WorkflowDispatch workflowDispatch     `yaml:"workflow_dispatch,omitempty"`
}

type cronExpr struct {
	// We use quotedString type here because '*' is a special character in YAML so we have
	// to quote the string.
	Cron quotedString `yaml:"cron"`
}

type pushAndPRTriggerOpts struct {
	Branches    []string `yaml:"branches,omitempty"`
	Paths       []string `yaml:"paths,omitempty"`
	PathsIgnore []string `yaml:"paths-ignore,omitempty"`
	Tags        []string `yaml:"tags,omitempty"`
}

type workflowDispatch struct {
	manualTrigger bool `yaml:"-"`
}

func (w workflowDispatch) IsZero() bool {
	return !w.manualTrigger
}

type job struct {
	// The name of the job displayed on GitHub.
	Name string `yaml:"name,omitempty"`

	// List of <job_id> that must complete successfully before this job will run.
	Needs []string `yaml:"needs,omitempty"`

	// The type of machine to run the job on.
	// Ref: https://docs.github.com/en/actions/using-github-hosted-runners/about-github-hosted-runners
	RunsOn any `yaml:"runs-on,flow"`

	// A map of environment variables that are available to all steps in the job.
	Env map[string]string `yaml:"env,omitempty"`

	// Steps can run commands, run setup tasks, or run an action. Not all steps
	// run actions, but all actions run as a step. Each step runs in its own
	// process in the runner environment and has access to the workspace and
	// filesystem.
	Steps []jobStep `yaml:"steps"`

	// Strategy creates a build matrix for the job and allows different
	// variations to run each job in.
	Strategy JobStrategy `yaml:"strategy,omitempty"`

	// A map of <service_id> to their configuration(s).
	Services map[string]jobService `yaml:"services,omitempty"`
}

type JobStrategy struct {
	Matrix struct {
		OS []string `yaml:"os"`
	} `yaml:"matrix"`
}

// jobService is used to host service containers for a job in a workflow. The
// runner automatically creates a Docker network and manages the life cycle of
// the service containers.
type jobService struct {
	// The Docker image to use as the service container to run the action. The
	// value can be the Docker Hub image name or a registry name.
	Image string `yaml:"image"`
	// Sets a map of environment variables in the service container.
	Env map[string]string `yaml:"env,omitempty"`
	// Sets an array of ports to expose on the service container.
	Ports []string `yaml:"ports,omitempty"`
	// Additional Docker container resource options.
	// For a list of options, see:
	//   https://docs.docker.com/engine/reference/commandline/create/#options
	Options string `yaml:"options,omitempty"`
}

func (j *job) addStep(s jobStep) {
	j.Steps = append(j.Steps, s)
}

// jobStep is a task that is run as part of a job.
type jobStep struct {
	// A name for your step to display on GitHub.
	Name string `yaml:"name"`

	// A unique identifier for the step. You can use the id to reference the
	// step in contexts.
	// Ref: https://docs.github.com/en/actions/reference/context-and-expression-syntax-for-github-actions
	ID string `yaml:"id,omitempty"`

	// You can use the if conditional to prevent a step from running unless a condition is met.
	If string `yaml:"if,omitempty"`

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
	With map[string]any `yaml:"with,omitempty"`

	// A map of environment variables for steps to use.
	Env map[string]string `yaml:"env,omitempty"`

	// Runs command-line programs using the operating system's shell.
	Run string `yaml:"run,omitempty"`
}
