# Contributing to Stock-n-Stash

This guide is written for contributors with any level of software experience.
It explains how to install Git, make changes without affecting the working
version of the project, and submit those changes for review.

App contributors should also read
[`documentation/pwa-app-development.md`](documentation/pwa-app-development.md).

## Git and GitHub in Plain Language

- **Git** runs on your computer and records the history of project files.
- **GitHub** stores a shared copy of the Git repository online.
- A **repository** is the project folder and its recorded history.
- A **branch** is an isolated line of work. Changes on your branch do not alter
  `main` until a pull request is reviewed and merged.
- A **commit** is a labeled snapshot of related changes.
- A **push** uploads local commits to GitHub.
- A **pull** downloads commits from GitHub.
- A **pull request**, or **PR**, asks the team to review and merge a branch.

## Important Rule

Do not commit or push changes directly to the `main` branch.

Complete each task on a separate branch and submit it through a pull request.
The `main` branch should always contain reviewed, working code.

## 1. Install Git

Use the instructions for your operating system. The official download page is
[git-scm.com/install](https://git-scm.com/install/).

### Windows

1. Download **Git for Windows** from
   [git-scm.com/download/win](https://git-scm.com/download/win).
2. Run the installer. The default choices are suitable for this project.
3. Open **Git Bash**, PowerShell, or the terminal in VS Code.
4. Verify the installation:

   ```bash
   git --version
   ```

Git Bash is often the easiest terminal for a beginner because its commands are
consistent with the macOS and Linux examples in this repository.

### macOS

Open Terminal and install Apple's Command Line Tools:

```bash
xcode-select --install
```

If the computer already uses Homebrew, an up-to-date Git version can instead be
installed with:

```bash
brew install git
```

Verify the installation:

```bash
git --version
```

### Ubuntu or Debian Linux

```bash
sudo apt update
sudo apt install git
git --version
```

### Fedora Linux

```bash
sudo dnf install git
git --version
```

If installation fails, send the team the operating-system version and the full
error message rather than repeatedly changing system settings.

## 2. Configure Git Once

Git places a name and email address on every commit. Use the name and email
associated with your GitHub account:

```bash
git config --global user.name "Your Name"
git config --global user.email "you@example.com"
git config --global init.defaultBranch main
```

Check the saved values:

```bash
git config --global --list
```

Create a GitHub account if needed and ask the repository owner to add that
account as a collaborator. This project uses an HTTPS repository address. When
GitHub asks you to sign in, use its browser or credential-manager flow. GitHub
does not accept an account password as a Git command-line password.

The official GitHub setup guide is
[Set up Git](https://docs.github.com/en/get-started/git-basics/set-up-git).

## 3. Download the Repository

Choose a parent folder where you keep projects, then run:

```bash
git clone https://github.com/jae-white/stock-n-stash.git
cd stock-n-stash
```

`git clone` is normally needed only once. Do not download a new ZIP file for
every task; the cloned repository can be updated with `git pull`.

Confirm that the repository is connected to GitHub:

```bash
git remote -v
git status
```

## 4. Set Up the Part of the Project You Are Editing

### Python edge software

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install -r requirements.txt
```

On Windows PowerShell, activate the environment with:

```powershell
.venv\Scripts\Activate.ps1
```

### Go backend

Install the Go version declared in `server/go.mod`, then download dependencies:

```bash
cd server
go mod download
cd ..
```

### React PWA

Follow [`documentation/pwa-app-development.md`](documentation/pwa-app-development.md).
Do not create a second app scaffold if an `app/` directory already exists.

## 5. Start Every Task from an Updated `main`

Save or commit any work already in progress before switching branches. Then:

```bash
git switch main
git pull --ff-only
git switch -c your-name/short-description
```

`--ff-only` stops instead of creating an unexpected merge commit on `main`.

Example branch names:

```text
jae/pwa-offline-status
justin/camera-wiring
luke/hardware-interface
gavin/recognition-integration
maxime/printing-documentation
jackson/structure-design
```

Use lowercase words separated by hyphens. Use a new branch for each unrelated
task so the pull request stays easy to review.

## 6. Make and Inspect Changes

Edit the files for the task. While working, check the repository often:

```bash
git status
git diff
```

- `git status` lists changed, staged, and untracked files.
- `git diff` shows the exact unstaged line changes.
- `git diff --staged` shows what the next commit will contain.

Before staging, remove temporary debugging output and confirm that no password,
token, private key, `.env` file, database, model, or personal file is included.

## 7. Stage and Commit Related Work

Stage specific files rather than blindly staging the whole repository:

```bash
git add path/to/file
git diff --staged
git commit -m "Add offline status indicator"
```

A commit should contain one understandable piece of work. Make another commit
when a task has a separate logical step.

Good commit messages start with an action and describe the result:

```text
Add three-camera capture component
Handle missing camera input
Document state machine recovery
Add capture failure tests
```

Avoid messages such as `changes`, `update`, `stuff`, or `fixed it`.

## 8. Push the Branch

The first push connects the local branch to a branch on GitHub:

```bash
git push -u origin your-name/short-description
```

After that, push additional commits with:

```bash
git push
```

If Git rejects the push, read the message before running another command. Do
not use `--force` unless the project owner explicitly asks you to.

## 9. Open a Pull Request

After pushing:

1. Open the repository on GitHub.
2. Click **Compare & pull request**, or open **Pull requests** and choose
   **New pull request**.
3. Confirm that the base branch is `main` and the compare branch is your branch.
4. Give the pull request a clear title.
5. Explain what changed, why it changed, and how it was tested.
6. Create a draft pull request if the work is not ready to merge.
7. Request review and wait for approval before merging.

Use this description format:

```text
What changed:
- Describe the changes.

Why:
- Explain why the changes were needed.

Testing:
- List the commands or physical checks that were run.
```

If a reviewer requests changes, keep using the same branch and pull request:

```bash
git add path/to/updated-file
git commit -m "Address review feedback"
git push
```

The pull request updates automatically. The official GitHub instructions are
[Creating a pull request](https://docs.github.com/en/pull-requests/how-tos/create-pull-requests/creating-a-pull-request).

## 10. Keep a Long-Running Branch Current

If other work reaches `main` before your pull request is finished:

```bash
git fetch origin
git merge origin/main
```

If Git reports a merge conflict, it will mark the conflicting sections in the
files. Do not guess which version is correct and do not delete the branch.
Coordinate with the person who changed the same area, edit the marked sections,
run the tests, and then finish the merge:

```bash
git add path/to/resolved-file
git commit
git push
```

## Safe Ways to Correct Common Mistakes

### A file was staged too early

Keep the file changes but remove the file from the next commit:

```bash
git restore --staged path/to/file
```

### A local edit should be discarded

First inspect it with `git diff`. The following command permanently discards
unstaged changes in that file:

```bash
git restore path/to/file
```

Use it only when you are certain the changes are not needed.

### Work started accidentally on `main`

Before committing, create a branch at the current position:

```bash
git switch -c your-name/short-description
```

The uncommitted changes move with you to the new branch.

### A secret was committed

Immediately tell the project owner. Removing the file in a later commit does
not remove the secret from history; the credential must be revoked and replaced.

Avoid `git reset --hard`, force pushes, and history-rewriting commands unless an
experienced maintainer is actively helping. Those operations can permanently
remove work.

## Project Organization

Place files in the appropriate locations:

- `edge/` — Python edge-device pipeline
- `edge/cameras/` — camera interfaces and capture code
- `edge/recognition/` — recognition implementations and result types
- `edge/association/` — multi-camera observation grouping
- `edge/positioning/` — camera-to-platform coordinate mapping
- `edge/reporting/` — result delivery interfaces
- `server/` — Go API and SQLite persistence
- `app/` — React/TypeScript PWA when its scaffold is added
- `tests/` — Python automated tests
- `documentation/` — architecture, setup, contracts, and design notes
- `.github/` — GitHub templates and automation

Do not commit:

- `.venv/`, `node_modules/`, `__pycache__/`, or build output
- `.env` files or credentials
- Local SQLite database files
- Generated mock captures
- Large datasets or trained model files
- Personal editor or operating-system files

## Code Readability

Code should be understandable to teammates outside the software specialty.

- Use descriptive names for variables, functions, classes, and components.
- Add docstrings or documentation comments to important public behavior.
- Keep functions and components focused on one responsibility.
- Comment why a non-obvious decision exists, not what an obvious line does.
- Remove unused imports and temporary debugging code.
- Follow the style already used in the surrounding files.
- Update the relevant Markdown document when behavior or architecture changes.
- Use `HARDWARE TODO:` for code that still needs a physical port, GPIO, driver,
  on-device calibration, or mock-to-hardware replacement. Include what must be
  verified; do not invent a port number.
- Keep the searchable hardware list in
  `documentation/hardware-integration-checklist.md` synchronized with the code.

## Testing Changes

Run the checks relevant to the code you touched.

Python edge software:

```bash
python3 -m unittest discover -s tests -v
```

Go backend:

```bash
cd server
go test ./...
```

PWA commands will be defined in the app's `package.json`. At minimum, app pull
requests should pass the configured formatter/linter, tests, type check, and
production build. See the PWA development guide for the expected test coverage.

When changing behavior, test:

- The expected successful behavior
- Failure and offline behavior
- Invalid input
- Important edge cases
- Any interface shared with another system component

## Pull Request Checklist

Before requesting review, confirm that:

- The branch contains only changes for one task.
- `git diff --staged` or the GitHub **Files changed** view looks correct.
- Code is readable and comments explain non-obvious decisions.
- Secrets, local databases, generated files, and large assets are absent.
- Applicable automated tests pass.
- New behavior has tests.
- Shared contracts and documentation were updated.
- The pull request explains the change and its verification.

## Git Command Cheat Sheet

| Goal | Command |
| --- | --- |
| See the current branch and changed files | `git status` |
| Download remote branch information | `git fetch origin` |
| Update local `main` | `git switch main` then `git pull --ff-only` |
| Create and enter a branch | `git switch -c name/task` |
| See unstaged changes | `git diff` |
| Stage one file | `git add path/to/file` |
| See staged changes | `git diff --staged` |
| Record staged changes | `git commit -m "Describe the change"` |
| Upload commits | `git push` |
| See recent commits | `git log --oneline -10` |
| See repository addresses | `git remote -v` |

## Reporting Problems

If you discover a problem but are not ready to fix it, create a GitHub issue
with:

- A clear description
- Steps that reproduce the problem
- Expected and actual behavior
- Relevant error messages, screenshots, or logs
- Hardware, operating system, and software versions when relevant

Never include passwords, access tokens, private keys, household join codes, or
other sensitive information.
