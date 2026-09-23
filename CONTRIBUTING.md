# Contributing to Stock-n-Stash

This guide explains how team members should make changes without accidentally affecting the working version of the project.

## Important Rule

Do not push changes directly to the `main` branch.

All work must be completed on a separate branch and submitted through a pull request. The `main` branch should always contain reviewed, working code.

## Getting the Repository

Clone the repository:

```bash
git clone https://github.com/jae-white/stock-n-stash.git
cd stock-n-stash
```

Create and activate the Python virtual environment:

```bash
python3 -m venv .venv
source .venv/bin/activate
python -m pip install -r requirements.txt
```

Backend contributors also need the Go version declared in `server/go.mod`.
Download the backend dependencies from the repository root with:

```bash
cd server
go mod download
cd ..
```

## Before Starting New Work

Always update your local `main` branch first:

```bash
git switch main
git pull
```

Then create a new branch:

```bash
git switch -c your-name/short-description
```

Examples:

```bash
git switch -c justin/camera-wiring
git switch -c luke/hardware-interface
git switch -c gavin/recognition-integration
git switch -c maxime/printing-documentation
git switch -c jackson/structure-design
```

Use a new branch for each separate task.

## Project Organization

Place files in the appropriate locations:

- `edge/` — edge-device Python code
- `edge/cameras/` — camera interfaces and capture code
- `server/` — Go HTTP API, SQLite persistence, and backend models
- `tests/` — automated tests
- `documentation/` — system explanations, diagrams, and design notes
- `.github/` — GitHub configuration files

Do not commit the following:

- `.venv/`
- `__pycache__/`
- Generated mock captures
- Large datasets
- Trained model files
- Personal editor or operating-system files

## Code Readability

Code should be understandable to other team members.

- Use descriptive names for variables, functions, and classes.
- Add a docstring to each class and important function.
- Keep functions focused on one responsibility.
- Add comments when explaining why something is being done.
- Do not add comments that simply repeat the code.
- Remove unused imports.
- Remove temporary debugging code before opening a pull request.
- Follow the style already used in the surrounding files.

Example:

```python
def handle_capture(self, capture_complete: bool):
    """Update the state machine after a camera capture attempt."""

    if self.state != State.CAPTURE:
        return

    if capture_complete:
        self.cap_valid = True
        self.state = State.RECOGNIZE
```

## Testing Changes

Run the Python tests for edge changes:

```bash
python3 -m unittest discover -s tests -v
```

Run the Go tests for backend changes:

```bash
cd server
go test ./...
```

Changes that affect the edge/backend interface should run both suites. All
applicable tests should pass before a pull request is opened.

When adding or changing behavior, add tests for:

- The expected successful behavior
- Failure behavior
- Calls made from the wrong state, when stateful behavior is involved
- Important edge cases

## Committing Changes

Check which files you changed:

```bash
git status
```

Stage only the files related to your task:

```bash
git add path/to/file
```

Create a short, descriptive commit:

```bash
git commit -m "Add camera connection interface"
```

Good commit messages describe what changed:

```text
Add three-camera capture component
Handle missing camera input
Document state machine recovery
Add capture failure tests
```

Avoid unclear commit messages such as:

```text
changes
update
stuff
fixed it
```

## Pushing a Branch

Push your branch to GitHub:

```bash
git push -u origin your-branch-name
```

After the first push, additional commits can be pushed with:

```bash
git push
```

## Opening a Pull Request

After pushing:

1. Open the repository on GitHub.
2. Click **Compare & pull request**.
3. Confirm that the base branch is `main`.
4. Give the pull request a clear title.
5. Explain what was changed and how it was tested.
6. Submit the pull request.
7. Wait for review before merging.

Do not merge another team member's pull request without reviewing the changes.

## Pull Request Description

Use this format:

```text
What changed:
- Describe the changes.

Why:
- Explain why the changes were needed.

Testing:
- Explain how the changes were tested.
```

## Pull Request Checklist

Before submitting a pull request, confirm that:

- The branch contains only related changes.
- The code is readable.
- Unnecessary files were not committed.
- All automated tests pass.
- New behavior has appropriate tests.
- Documentation was updated when necessary.
- The pull request clearly explains the change.

## Reporting Problems

If you discover a problem but are not ready to fix it, create a GitHub issue containing:

- A clear description of the problem
- Steps that reproduce it
- The expected behavior
- The actual behavior
- Any relevant error message

Do not include passwords, access tokens, private keys, or other sensitive information.
