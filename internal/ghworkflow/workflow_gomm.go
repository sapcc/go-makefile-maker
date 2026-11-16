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
	runJob.addStep(jobStep{
		Name: "Set up Go",
		Uses: core.SetupGoAction,
		With: map[string]any{
			"check-latest": true,
			"go-version":   cfg.GitHubWorkflow.Global.GoVersion.UnwrapOr(core.DefaultGoVersion),
		},
	})
	if isGoMakefileMakerRepository {
		w.Permissions.Contents = tokenScopeWrite
		runJob.Steps[0].With = map[string]any{
			"ref":        "${{github.event.pull_request.head.ref}}",
			"repository": "${{github.event.pull_request.head.repo.full_name}}",
		}
	} else {
		w.Permissions.PullRequests = tokenScopeWrite
		runJob.Steps[0].With["fetch-depth"] = 2
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
	}

	runJob.addStep(jobStep{
		Name: "Set up Node.js",
		Uses: core.SetupNodeAction,
		With: map[string]any{
			"node-version": "26",
		},
	})
	runJob.addStep(jobStep{
		Name: "Install Renovate config validator",
		Run:  "npm install --global renovate",
	})

	if isGoMakefileMakerRepository {
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
		runJob.addStep(jobStep{
			Name: "Run go-makefile-maker",
			Env: map[string]string{
				"BASE_SHA": "${{ github.event.pull_request.base.sha }}",
				"PR_SHA":   "${{ github.event.pull_request.head.sha }}",
			},
			Run: makeMultilineYAMLString([]string{
				"if [[ \"$GITHUB_EVENT_NAME\" == \"pull_request\" ]]; then",
				"  git checkout --force --detach \"$BASE_SHA\"",
				"  go run github.com/sapcc/go-makefile-maker@latest",
				"  git add -A",
				"  git diff --cached --no-ext-diff --no-color --binary > \"$RUNNER_TEMP/gomm-base.diff\"",
				"  git diff --cached --no-ext-diff --no-color --binary --unified=0 --no-renames > \"$RUNNER_TEMP/gomm-base-normalized.diff\"",
				"  git checkout --force --detach \"$PR_SHA\"",
				"  git clean -fd",
				"  go run github.com/sapcc/go-makefile-maker@latest",
				"  git add -A",
				"  git diff --cached --no-ext-diff --no-color --binary > \"$RUNNER_TEMP/gomm-pr.diff\"",
				"  git diff --cached --no-ext-diff --no-color --binary --unified=0 --no-renames > \"$RUNNER_TEMP/gomm-pr-normalized.diff\"",
				"else",
				"  go run github.com/sapcc/go-makefile-maker@latest",
				"fi",
			}),
		})
		runJob.addStep(jobStep{
			Name: "Report generated changes",
			Uses: core.GitHubScriptAction,
			With: map[string]any{
				"script": `const { execFileSync } = require('node:child_process')
const fs = require('node:fs')
const path = require('node:path')

const execOptions = { encoding: 'utf8', maxBuffer: 50 * 1024 * 1024 }
const marker = '<!-- go-makefile-maker-generated-diff -->'
let baseDiff = ''
let diff = ''
let status = ''
let hasChanges

if (context.eventName === 'pull_request') {
  const temp = process.env.RUNNER_TEMP
  baseDiff = fs.readFileSync(path.join(temp, 'gomm-base.diff'), 'utf8')
  diff = fs.readFileSync(path.join(temp, 'gomm-pr.diff'), 'utf8')
  const baseNormalized = fs.readFileSync(path.join(temp, 'gomm-base-normalized.diff'), 'utf8')
  const prNormalized = fs.readFileSync(path.join(temp, 'gomm-pr-normalized.diff'), 'utf8')

  function normalizeEdits(patch) {
    const edits = []
    let inHunk = false
    let inBinaryPatch = false

    for (const line of patch.split('\n')) {
      if (line.startsWith('diff --git ')) {
        edits.push(line)
        inHunk = false
        inBinaryPatch = false
        continue
      }

      if (line === 'GIT binary patch') {
        edits.push(line)
        inBinaryPatch = true
        continue
      }
      if (inBinaryPatch) {
        edits.push(line)
        continue
      }

      if (line.startsWith('@@')) {
        inHunk = true
        continue
      }
      if (line.startsWith('new file mode ') || line.startsWith('deleted file mode ') ||
          line.startsWith('old mode ') || line.startsWith('new mode ')) {
        edits.push(line)
        continue
      }
      if (!inHunk) continue

      if (line.startsWith('+') || line.startsWith('-') || line.startsWith('\\')) {
        edits.push(line)
      }
    }

    return edits.join('\n')
  }

  hasChanges = normalizeEdits(baseNormalized) !== normalizeEdits(prNormalized)
  status = execFileSync('git', ['status', '--porcelain'], execOptions)
} else {
  execFileSync('git', ['add', '-N', '--', '.'])
  status = execFileSync('git', ['status', '--porcelain'], execOptions)
  diff = execFileSync('git', ['diff', '--no-ext-diff', '--no-color'], execOptions)
  hasChanges = status.length > 0
}

if (hasChanges) {
  core.info(context.eventName === 'pull_request'
    ? 'go-makefile-maker produces different generated edits on the base and PR commits.'
    : 'go-makefile-maker would change these files:')
  if (status) core.info(status)
  if (context.eventName === 'pull_request') {
    core.info('Generated diff on the base commit:')
    core.info(baseDiff || '(no generated changes)')
  }
  core.info(context.eventName === 'pull_request' ? 'Generated diff on the PR commit:' : 'Generated diff:')
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
      const maxDiffLength = 28000
      const isPRDiffTruncated = diff.length > maxDiffLength
      const isBaseDiffTruncated = baseDiff.length > maxDiffLength
      const reportedDiff = isPRDiffTruncated ? diff.slice(0, maxDiffLength) : diff
      const reportedBaseDiff = isBaseDiffTruncated ? baseDiff.slice(0, maxDiffLength) : baseDiff
      const fence = String.fromCharCode(96).repeat(4)
      const body = [
        marker,
        '⚠️ go-makefile-maker produces different generated edits on the base and PR commits.',
        '',
        '<details><summary>Generated diff on the PR commit</summary>',
        '',
        fence + 'diff',
        reportedDiff + (isPRDiffTruncated ? '\n... diff truncated to fit in this comment.' : ''),
        fence,
        '</details>',
        '',
        '<details><summary>Generated diff on the base commit</summary>',
        '',
        fence + 'diff',
        (reportedBaseDiff || '(no generated changes)') + (isBaseDiffTruncated ? '\n... diff truncated to fit in this comment.' : ''),
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
    ? 'go-makefile-maker produces different generated edits on the base and PR commits; see the pull request comment for the diffs.'
    : 'go-makefile-maker produced changes.')
}`,
			},
		})
	}

	w.Jobs["run"] = runJob
	return Some(w)
}
