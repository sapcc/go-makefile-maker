// SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package ghworkflow

import (
	"strings"

	. "go.xyrillian.de/gg/option"

	"github.com/sapcc/go-makefile-maker/internal/core"
)

func goMakefileMakerWorkflow(cfg core.Configuration) Option[workflow] {
	const workflowName = "Run go-makefile-maker"

	isGoMakefileMakerRepository := strings.Contains(cfg.Metadata.URL, "github.com/sapcc/go-makefile-maker")
	w := workflow{
		Name: workflowName,
		On: eventTrigger{
			Push: pushAndPRTriggerOpts{
				Branches:    []string{"**"},
				PathsIgnore: []string{"**.md"},
			},
			PullRequest: pushAndPRTriggerOpts{
				Branches:    []string{"**"},
				PathsIgnore: []string{"**.md"},
			},
			WorkflowDispatch: workflowDispatch{manualTrigger: true},
		},
		Permissions: permissions{Contents: tokenScopeRead},
		Jobs:        make(map[string]job),
	}

	runJob := baseJob(workflowName, cfg.GitHubWorkflow)
	if isGoMakefileMakerRepository {
		w.Permissions.Contents = tokenScopeWrite
		runJob.Steps[0].With = map[string]any{
			"ref":        "${{github.event.pull_request.head.ref}}",
			"repository": "${{github.event.pull_request.head.repo.full_name}}",
		}
	}
	runJob.addStep(jobStep{
		Name: "Set up Go",
		Uses: core.SetupGoAction,
		With: map[string]any{
			"check-latest": true,
			"go-version":   cfg.GitHubWorkflow.Global.GoVersion.UnwrapOr(core.DefaultGoVersion),
		},
	})

	if isGoMakefileMakerRepository {
		runJob.addStep(jobStep{
			Name: "Install Renovate config validator",
			Run:  "npm install --global renovate",
		})
		runJob.addStep(jobStep{
			Name: "Build and run go-makefile-maker",
			Run: makeMultilineYAMLString([]string{
				"make",
				"make install-go-licence-detector",
				"./build/go-makefile-maker",
			}),
		})
		runJob.addStep(jobStep{
			Name: "Commit generated changes",
			Run: makeMultilineYAMLString([]string{
				"if [[ -n \"$(git status --porcelain)\" ]]; then",
				"  git config user.name \"github-actions[bot]\"",
				"  git config user.email \"41898282+github-actions[bot]@users.noreply.github.com\"",
				"  git add .",
				"  git commit -m \"Run go-makefile-maker\"",
				"  git push",
				"fi",
			}),
		})
	} else {
		w.Permissions.PullRequests = tokenScopeWrite
		if cfg.Golang.EnableVendoring && cfg.Reuse.IsEnabled() {
			runJob.addStep(jobStep{
				Name: "Install go-licence-detector",
				Run:  "go install go.elastic.co/go-licence-detector@latest",
			})
		}
		runJob.addStep(jobStep{
			Name: "Set default branch for go-makefile-maker",
			Env: map[string]string{
				"DEFAULT_BRANCH": "${{ github.event.repository.default_branch }}",
			},
			Run: `git symbolic-ref refs/remotes/origin/HEAD "refs/remotes/origin/$DEFAULT_BRANCH"`,
		})
		runJob.addStep(jobStep{
			Name: "Install Renovate config validator",
			Run:  "npm install --global renovate",
		})
		runJob.addStep(jobStep{
			Name: "Run go-makefile-maker",
			Run:  "go run github.com/sapcc/go-makefile-maker@latest",
		})
		runJob.addStep(jobStep{
			Name: "Report generated changes",
			Uses: core.GitHubScriptAction,
			With: map[string]any{
				"script": `const { execFileSync } = require('node:child_process')

execFileSync('git', ['add', '-N', '--', '.'])
const execOptions = { encoding: 'utf8', maxBuffer: 50 * 1024 * 1024 }
const status = execFileSync('git', ['status', '--porcelain'], execOptions)
const diff = execFileSync('git', ['diff', '--no-ext-diff', '--no-color'], execOptions)
const marker = '<!-- go-makefile-maker-generated-diff -->'
const hasChanges = status.length > 0

if (hasChanges) {
  core.info('go-makefile-maker would change these files:')
  core.info(status)
  core.info('Generated diff:')
  core.info(diff)
}

if (context.eventName === 'pull_request') {
  try {
    const { owner, repo } = context.repo
    const issue_number = context.payload.pull_request.number
    const comments = await github.paginate(github.rest.issues.listComments, {
      owner,
      repo,
      issue_number,
      per_page: 100,
    })
    const existingComment = comments.reverse().find(comment =>
      comment.user?.login === 'github-actions[bot]' && comment.body?.includes(marker)
    )

    if (hasChanges) {
      const maxDiffLength = 60000
      const isTruncated = diff.length > maxDiffLength
      const reportedDiff = isTruncated ? diff.slice(0, maxDiffLength) : diff
      const fence = String.fromCharCode(96).repeat(4)
      const body = [
        marker,
        '⚠️ go-makefile-maker would change generated files. Run it locally and commit the resulting changes.',
        '',
        '<details><summary>Generated diff</summary>',
        '',
        fence + 'diff',
        reportedDiff + (isTruncated ? '\n... diff truncated to fit in this comment.' : ''),
        fence,
        '</details>',
      ].join('\n')

      if (existingComment) {
        await github.rest.issues.updateComment({
          owner,
          repo,
          comment_id: existingComment.id,
          body,
        })
      } else {
        await github.rest.issues.createComment({ owner, repo, issue_number, body })
      }
    } else if (existingComment) {
      await github.rest.issues.deleteComment({
        owner,
        repo,
        comment_id: existingComment.id,
      })
    }
  } catch (error) {
    core.warning('Unable to update the go-makefile-maker PR comment: ' + (error?.message ?? String(error)))
  }
}

if (hasChanges) {
  core.setFailed(context.eventName === 'pull_request'
    ? 'go-makefile-maker produced changes; see the pull request comment for the diff.'
    : 'go-makefile-maker produced changes.')
}`,
			},
		})
	}

	w.Jobs["run"] = runJob
	return Some(w)
}
