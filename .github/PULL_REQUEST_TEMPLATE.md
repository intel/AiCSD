## PR Checklist

## Author Mandatory (to be filled by PR Author/Submitter)

### CODE MAINTAINABILITY

- [ ] **_Commit Message meets guidelines as indicated in the URL\*_**
    - <https://github.com/intel/AiCSD/blob/main/.github/CONTRIBUTING.md>
- [ ] Tests for the changes have been added (required for bug fixes / features). **If not, state why not**
- [ ] Added labels (these are git labels such as bug or enhancement)
- [ ] For any API changes or service changes, updates to the documentation were made (Swagger APIs & GitHub Docs). **If not, state why not**
- [ ] **_Every commit is a single defect fix and does not mix feature addition or changes\*_**

### What are the steps to test this PR contribution?

How does a reviewer know it works?

<!-- For example, "1. Run `make integration-test` ... observe no errors." -->
<!-- For example, "Open localhost:4200 and confirm view looks like this screenshot below" -->

Are there any special instructions beyond standard workflow?

<!-- For example, "This patch requires Linux or two systems to test." -->

## Maintainer Mandatory (to be filled by PR Reviewer/Approving Maintainer)

### QUALITY CHECKS

- [ ] **_Quality of code (At least one should be checked as applicable)\*_**
    - [ ] Commit Message meets guidelines
    - [ ] Code copyright is correct
    - [ ] PR changes adhere to industry practices and standards as well as domain or language specific anti-patterns
    - [ ] Adopted domain specific coding standards ([Go Coding Standards](https://go.dev/doc/effective_go))
    - [ ] Code is adequately commented and documented
    - [ ] Confusing logic is explained in comments
    - [ ] Error and exception code paths implemented correctly
    - [ ] Tracing output are minimized and logic
- [ ] **_Test coverage shows adequate coverage with required CI functional tests pass on all supported platforms\*_**
- [ ] **_Static code scan report shows zero critical issues\*_**

## Other information