# Contributing to GitHub Actions Workflow Checker

We love your input! We want to make contributing to GitHub Actions Workflow Checker as easy and transparent as possible, whether it's:

- Reporting a bug
- Discussing the current state of the code
- Submitting a fix
- Proposing new features
- Becoming a maintainer

## Development Process

We use GitHub to host code, to track issues and feature requests, as well as accept pull requests.

1. Fork the repo and create your branch from `main`
2. If you've added code that should be tested, add tests
3. If you've changed APIs, update the documentation
4. Ensure the test suite passes
5. Make sure your code lints
6. Issue that pull request!

## Pull Request Process

1. Update the README.md with details of changes to the interface, if applicable
2. Update the docs/api.md with any API changes
3. The PR will be merged once you have the sign-off of at least one maintainer

## Any Contributions You Make Will Be Under the MIT License

In short, when you submit code changes, your submissions are understood to be under the same [MIT License](LICENSE) that covers the project. Feel free to contact the maintainers if that's a concern.

## Report Bugs Using GitHub's [Issue Tracker](../../issues)

We use GitHub issues to track public bugs. Report a bug by [opening a new issue](../../issues/new); it's that easy!

## Write Bug Reports with Detail, Background, and Sample Code

**Great Bug Reports** tend to have:

- A quick summary and/or background
- Steps to reproduce
  - Be specific!
  - Give sample code if you can
- What you expected would happen
- What actually happens
- Notes (possibly including why you think this might be happening, or stuff you tried that didn't work)

## Use a Consistent Coding Style

* Use `go fmt` for formatting
* Document exported functions and types
* Write tests for new functionality
* Follow Go best practices and idioms
* Preserve version tag comments in workflow files
* Use commit hashes for secure action references
* Follow comment preservation guidelines

### Version Tag Handling

When working with GitHub Action references:
* Always preserve version information in comments
* Use commit hashes for secure referencing
* Follow this format for version comments:
  ```yaml
  # Using actions/checkout for repository access
  # Original version: v2
  uses: actions/checkout@a81bbbf8298c0fa03ea29cdc473d45769f953675  # v3
  ```

### Comment Preservation

When updating workflow files:
* Preserve all existing comments
* Add version tracking comments
* Maintain comment formatting and style
* Document version changes clearly
* Keep comments close to their relevant actions

### Commit Hash Usage

When working with action references:
* Always use full commit hashes (40 characters)
* Verify hash validity with GitHub API
* Include version information in comments
* Update hashes when versions change
* Document hash changes in PRs

## Code Review Process

The core team looks at Pull Requests on a regular basis. After feedback has been given we expect responses within two weeks. After two weeks we may close the PR if it isn't showing any activity.

### Pull Request Guidelines

When submitting changes that affect action updates:
1. Include before/after examples of workflow changes
2. Document version and hash changes clearly
3. Explain any comment preservation decisions
4. Verify commit hash validity
5. Include test coverage for new functionality
6. Update relevant documentation

## Community

Discussions about the project take place on this repository's [Issues](../../issues) and [Pull Requests](../../pulls) sections. Anybody is welcome to join these conversations.

## References

This document was adapted from the open-source contribution guidelines for [Facebook's Draft](https://github.com/facebook/draft-js/blob/a9316a723f9e918afde44dea68b5f9f39b7d9b00/CONTRIBUTING.md).
