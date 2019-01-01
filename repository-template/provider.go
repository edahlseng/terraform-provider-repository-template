package repositoryTemplate

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"commit_author_name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Author name to use when signing commits",
			},
			"commit_message": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Commit message to use",
			},
			"github_token": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("GITHUB_TOKEN", nil),
				Description: "GitHub access token",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"repository-template_github": resourceRepositoryTemplateGitHub(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	config := Config{
		GitHubToken: d.Get("github_token").(string),
	}

	client := config.NewClient()

	client.CommitMessage = d.Get("commit_message").(string)
	client.CommitAuthorName = d.Get("commit_author_name").(string)

	return client, nil
}
