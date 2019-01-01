Repository Template Terraform Provider
======================================

The Repository Template Terraform Provider makes it easy to keep certain files in sync across several different repositories.

Setup
-----

```hcl
provider "repository-template" {
  github_token       = "<personal access token>"
  commit_message     = "chore: Update files to match template" // Replace with desired commit message
  commit_author_name = "Template Bot"                          // Replace with desired commit author name
}
```

Resources
---------

### repository-template_github

#### Example Usage:

```hcl
resource "repository-template_github" "example" {
  repository_owner = "example-user"
  repository_name  = "example"
  target_branch    = "master"
  working_branch   = "ci/template"

  files = {
    "CONTRIBUTING.md" = "Pull requests are welcome!"
  }
}
```

#### Argument Reference:

The following arguments are supported:

* repository_owner (Required) - The GitHub user or organization that owns the repository.
* repository_name (Required) - The name of the repository.
* target_branch (Required) - The branch that is desired to match the template (typically `master`).
* working_branch (Required) - The branch to make changes on before submitting a pull request.
* files (Required) - A map of file paths to contents for which the target_branch should contain files that match.

#### Attributes Reference

All of the attributes above are exported.

Building The Provider
---------------------

```shell
make build # `gnumake build` on macOS
```
