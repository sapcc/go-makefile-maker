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
				"go install github.com/sapcc/go-makefile-maker@latest",
				"GOMM=\"$(go env GOPATH)/bin/go-makefile-maker\"",
				"if [[ \"$GITHUB_EVENT_NAME\" == \"pull_request\" ]]; then",
				"  git checkout --force --detach \"$BASE_SHA\"",
				"  \"$GOMM\"",
				"  git add -A",
				"  git diff --cached --no-ext-diff --no-color --binary > \"$RUNNER_TEMP/gomm-base.diff\"",
				"  git checkout --force --detach \"$PR_SHA\"",
				"  git clean -fd",
				"  \"$GOMM\"",
				"  git add -A",
				"  git status --porcelain > \"$RUNNER_TEMP/gomm-pr.status\"",
				"  git diff --cached --no-ext-diff --no-color --binary > \"$RUNNER_TEMP/gomm-pr.diff\"",
				"  git diff --cached --no-ext-diff --no-color --binary --unified=0 --no-renames > \"$RUNNER_TEMP/gomm-pr-normalized.diff\"",
				"  git reset --hard \"$PR_SHA\"",
				"  git clean -fd",
				"  if [[ ! -s \"$RUNNER_TEMP/gomm-pr.diff\" ]]; then",
				"    : > \"$RUNNER_TEMP/gomm-inherited.diff\"",
				"  else",
				"    # Carry base-generated edits forward; compare them at PR-tree locations.",
				"    if [[ ! -s \"$RUNNER_TEMP/gomm-base.diff\" ]] || git apply --3way --theirs --index \"$RUNNER_TEMP/gomm-base.diff\"; then",
				"      git diff --cached --no-ext-diff --no-color --unified=0 --no-renames \"$PR_SHA\" > \"$RUNNER_TEMP/gomm-inherited.diff\"",
				"    else",
				"      touch \"$RUNNER_TEMP/gomm-compare-apply-failed\"",
				"      : > \"$RUNNER_TEMP/gomm-inherited.diff\"",
				"    fi",
				"  fi",
				"else",
				"  \"$GOMM\"",
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
let comparisonApplyFailed = false

if (context.eventName === 'pull_request') {
  const temp = process.env.RUNNER_TEMP
  baseDiff = fs.readFileSync(path.join(temp, 'gomm-base.diff'), 'utf8')
  diff = fs.readFileSync(path.join(temp, 'gomm-pr.diff'), 'utf8')
  const inheritedDiff = fs.readFileSync(path.join(temp, 'gomm-inherited.diff'), 'utf8')
  const prNormalized = fs.readFileSync(path.join(temp, 'gomm-pr-normalized.diff'), 'utf8')
  comparisonApplyFailed = fs.existsSync(path.join(temp, 'gomm-compare-apply-failed'))
  try {
    hasChanges = comparisonApplyFailed || !editsAreSubset(prNormalized, inheritedDiff)
  } catch (error) {
    comparisonApplyFailed = true
    hasChanges = true
    core.warning('Unable to compare generated edits: ' + (error?.message ?? String(error)))
  }
  status = fs.readFileSync(path.join(temp, 'gomm-pr.status'), 'utf8')
} else {
  execFileSync('git', ['add', '-N', '--', '.'])
  status = execFileSync('git', ['status', '--porcelain'], execOptions)
  diff = execFileSync('git', ['diff', '--no-ext-diff', '--no-color'], execOptions)
  hasChanges = status.length > 0
}

function editsAreSubset(candidatePatch, inheritedPatch) {
  // Both diffs start at PR_SHA, so these file/line edit keys retain locations while allowing subsets.
  function editCounts(patch) {
    const counts = new Map()
    let file = ''
    let oldLine = 0
    let block
    let binaryPayload

    function add(key) {
      counts.set(key, (counts.get(key) ?? 0) + 1)
    }

    function flushBlock() {
      if (!block) return
      for (const deletion of block.deletions) {
        add(JSON.stringify([file, '-', deletion.line, deletion.text]))
      }
      for (let index = 0; index < block.additions.length; index++) {
        const line = block.deletions[index]?.line ??
          block.deletions.at(-1)?.line ?? block.startLine
        add(JSON.stringify([file, '+', line, block.additions[index]]))
      }
      block = undefined
    }

    function flushBinary() {
      if (!binaryPayload) return
      add(JSON.stringify([file, 'binary', binaryPayload.join('\n')]))
      binaryPayload = undefined
    }

    for (const line of patch.split('\n')) {
      if (line.startsWith('diff --git ')) {
        flushBlock()
        flushBinary()
        file = line
        continue
      }
      if (line.startsWith('@@')) {
        flushBlock()
        const match = line.match(/^@@ -(\d+)(?:,\d+)? \+\d+(?:,\d+)? @@/)
        if (!match) throw new Error('Invalid diff hunk header: ' + line)
        oldLine = Number(match[1])
        continue
      }
      if (line.startsWith('new file mode ') || line.startsWith('deleted file mode ') ||
          line.startsWith('old mode ') || line.startsWith('new mode ')) {
        add(JSON.stringify([file, 'mode', line]))
        continue
      }
      if (line.startsWith('GIT binary patch')) {
        flushBlock()
        binaryPayload = [line]
        continue
      }
      if (binaryPayload) {
        binaryPayload.push(line)
        continue
      }
      if (line.startsWith('\\')) continue
      if (line.startsWith('-') && !line.startsWith('---')) {
        if (!block) block = { startLine: oldLine, deletions: [], additions: [] }
        block.deletions.push({ line: oldLine, text: line.slice(1) })
        oldLine++
      } else if (line.startsWith('+') && !line.startsWith('+++')) {
        if (!block) block = { startLine: oldLine, deletions: [], additions: [] }
        block.additions.push(line.slice(1))
      } else if (line.startsWith(' ')) {
        flushBlock()
        oldLine++
      }
    }
    flushBlock()
    flushBinary()
    return counts
  }

  const inherited = editCounts(inheritedPatch)
  for (const [edit, count] of editCounts(candidatePatch)) {
    if ((inherited.get(edit) ?? 0) < count) return false
  }
  return true
}

if (hasChanges) {
  core.info(context.eventName === 'pull_request'
    ? comparisonApplyFailed
      ? 'Unable to compare generated changes using a three-way patch; reporting the PR generated diff conservatively.'
      : 'go-makefile-maker produces generated edits on the PR that are not covered by the base commit’s generated edits.'
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
        comparisonApplyFailed
          ? '⚠️ Unable to compare generated edits using a three-way patch, so the comparison is inconclusive. Reporting the PR generated diff conservatively.'
          : '⚠️ go-makefile-maker produces generated edits on the PR that are not covered by the base commit’s generated edits.',
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
    ? comparisonApplyFailed
      ? 'Unable to compare generated edits using a three-way patch; see the pull request comment for the diffs.'
      : 'go-makefile-maker produces generated edits on the PR that are not covered by the base commit’s generated edits; see the pull request comment for the diffs.'
    : 'go-makefile-maker produced changes.')
}`,
			},
		})
	}

	w.Jobs["run"] = runJob
	return Some(w)
}
