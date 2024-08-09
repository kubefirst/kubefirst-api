/*
Copyright (C) 2021-2023, Kubefirst

This program is licensed under MIT.
See the LICENSE file for more details.
*/
package gitlab

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/xanzy/go-gitlab"
)

// NewGitLabClient instantiates a wrapper to communicate with GitLab
// It sets the path and ID of the group under which resources will be managed
func NewGitLabClient(token string, parentGroupName string) (*Wrapper, error) {
	git, err := gitlab.NewClient(token)
	if err != nil {
		return nil, fmt.Errorf("error instantiating gitlab client: %w", err)
	}

	// Get parent group ID
	minAccessLevel := gitlab.DeveloperPermissions
	container := make([]gitlab.Group, 0)
	for nextPage := 1; nextPage > 0; {
		groups, resp, err := git.Groups.ListGroups(&gitlab.ListGroupsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    nextPage,
				PerPage: 100,
			},
			MinAccessLevel: &minAccessLevel,
		})
		if err != nil {
			return nil, fmt.Errorf("could not get gitlab groups: %w", err)
		}
		for _, group := range groups {
			container = append(container, *group)
		}
		nextPage = resp.NextPage
	}

	var gid int
	for _, group := range container {
		if group.FullPath == parentGroupName {
			gid = group.ID
		} else {
			continue
		}
	}

	if gid == 0 {
		return nil, fmt.Errorf("could not find gitlab group %s", parentGroupName)
	}

	// Get parent group path
	group, _, err := git.Groups.GetGroup(gid, &gitlab.GetGroupOptions{})
	if err != nil {
		return nil, fmt.Errorf("could not get gitlab parent group path: %w", err)
	}

	return &Wrapper{
		Client:          git,
		ParentGroupID:   gid,
		ParentGroupPath: group.FullPath,
	}, nil
}

// CheckProjectExists within a parent group
func (gl *Wrapper) CheckProjectExists(projectName string) (bool, error) {
	allprojects, err := gl.GetProjects()
	if err != nil {
		return false, err
	}

	exists := false
	for _, project := range allprojects {
		if project.Name == projectName {
			exists = true
		}
	}

	return exists, nil
}

// GetProjectID returns a project's ID scoped to the parent group
func (gl *Wrapper) GetProjectID(projectName string) (int, error) {
	container := make([]gitlab.Project, 0)
	for nextPage := 1; nextPage > 0; {
		projects, resp, err := gl.Client.Groups.ListGroupProjects(gl.ParentGroupID, &gitlab.ListGroupProjectsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    nextPage,
				PerPage: 20,
			},
		})
		if err != nil {
			return 0, err
		}
		for _, project := range projects {
			container = append(container, *project)
		}
		nextPage = resp.NextPage
	}

	for _, project := range container {
		if !strings.Contains(project.Name, "deleted") &&
			strings.ToLower(project.Name) == projectName {
			return project.ID, nil
		}
	}

	return 0, fmt.Errorf("could not get project ID for project %s", projectName)
}

// GetProjects for a specific parent group by ID
func (gl *Wrapper) GetProjects() ([]gitlab.Project, error) {
	container := make([]gitlab.Project, 0)
	for nextPage := 1; nextPage > 0; {
		projects, resp, err := gl.Client.Groups.ListGroupProjects(gl.ParentGroupID, &gitlab.ListGroupProjectsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    nextPage,
				PerPage: 20,
			},
		})
		if err != nil {
			return []gitlab.Project{}, err
		}
		for _, project := range projects {
			// Skip deleted projects
			if !strings.Contains(project.Name, "deleted") {
				container = append(container, *project)
			}
		}
		nextPage = resp.NextPage
	}

	return container, nil
}

// GetSubGroups for a specific parent group by ID
func (gl *Wrapper) GetSubGroups() ([]gitlab.Group, error) {
	container := make([]gitlab.Group, 0)
	for nextPage := 1; nextPage > 0; {
		subgroups, resp, err := gl.Client.Groups.ListSubGroups(gl.ParentGroupID, &gitlab.ListSubGroupsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    nextPage,
				PerPage: 20,
			},
		})
		if err != nil {
			return []gitlab.Group{}, err
		}
		for _, subgroup := range subgroups {
			container = append(container, *subgroup)
		}
		nextPage = resp.NextPage
	}

	return container, nil
}

// User Management

// AddUserSSHKey
func (gl *Wrapper) AddUserSSHKey(keyTitle string, keyValue string) error {
	_, _, err := gl.Client.Users.AddSSHKey(&gitlab.AddSSHKeyOptions{
		Title: &keyTitle,
		Key:   &keyValue,
	})
	if err != nil {
		return fmt.Errorf("could not add ssh key %q: %w", keyTitle, err)
	}

	return nil
}

// DeleteUserSSHKey
func (gl *Wrapper) DeleteUserSSHKey(keyTitle string) error {
	allkeys, err := gl.GetUserSSHKeys()
	if err != nil {
		return fmt.Errorf("could not get user ssh keys: %w", err)
	}

	var keyID int
	for _, key := range allkeys {
		if key.Title == keyTitle {
			keyID = key.ID
		}
	}

	if keyID == 0 {
		return fmt.Errorf("could not find ssh key %s so it will not be deleted - you may need to delete it manually", keyTitle)
	}

	_, err = gl.Client.Users.DeleteSSHKey(keyID)
	if err != nil {
		return fmt.Errorf("could not delete ssh key %q: %w", keyTitle, err)
	}

	log.Info().Msgf("deleted gitlab ssh key %s", keyTitle)
	return nil
}

// GetUserSSHKeys
func (gl *Wrapper) GetUserSSHKeys() ([]*gitlab.SSHKey, error) {
	keys, _, err := gl.Client.Users.ListSSHKeys()
	if err != nil {
		return nil, fmt.Errorf("could not get user ssh keys: %w", err)
	}

	return keys, nil
}

// Container Registry

// GetProjectContainerRegistryRepositories
func (gl *Wrapper) GetProjectContainerRegistryRepositories(projectName string) ([]gitlab.RegistryRepository, error) {
	projectID, err := gl.GetProjectID(projectName)
	if err != nil {
		return nil, fmt.Errorf("could not get project ID for project %s: %w", projectName, err)
	}

	container := make([]gitlab.RegistryRepository, 0)
	for nextPage := 1; nextPage > 0; {
		repositories, resp, err := gl.Client.ContainerRegistry.ListProjectRegistryRepositories(projectID, &gitlab.ListRegistryRepositoriesOptions{
			ListOptions: gitlab.ListOptions{
				Page:    nextPage,
				PerPage: 20,
			},
		})
		if err != nil {
			return nil, fmt.Errorf("could not get project container registry repositories for page %d: %w", nextPage, err)
		}

		for _, subgroup := range repositories {
			container = append(container, *subgroup)
		}
		nextPage = resp.NextPage
	}

	return container, nil
}

// DeleteProjectContainerRegistryRepository
func (gl *Wrapper) DeleteContainerRegistryRepository(projectName string, repositoryID int) error {
	projectID, err := gl.GetProjectID(projectName)
	if err != nil {
		return fmt.Errorf("could not get project ID for project %s: %w", projectName, err)
	}

	// Delete any tags
	nameRegEx := ".*"
	_, err = gl.Client.ContainerRegistry.DeleteRegistryRepositoryTags(projectID, repositoryID, &gitlab.DeleteRegistryRepositoryTagsOptions{
		NameRegexpDelete: &nameRegEx,
	})
	if err != nil {
		return fmt.Errorf("could not delete tags for container registry repository %d: %w", repositoryID, err)
	}

	log.Info().Msgf("removed all tags from container registry for project %s", projectName)

	// Delete repository
	_, err = gl.Client.ContainerRegistry.DeleteRegistryRepository(projectID, repositoryID)
	if err != nil {
		return fmt.Errorf("could not delete container registry repository %d: %w", repositoryID, err)
	}

	log.Info().Msgf("deleted container registry for project %s", projectName)
	return nil
}

// Token & Key Management

// CreateGroupDeployToken creates a deploy token for a group
// If no groupID (0 by default) argument is provided, the parent group ID is used
// If a group deploy token already exists, it will be deleted and recreated
func (gl *Wrapper) CreateGroupDeployToken(groupID int, p *DeployTokenCreateParameters) (string, error) {
	// Check to see if the token already exists
	allTokens, err := gl.ListGroupDeployTokens(gl.ParentGroupID)
	if err != nil {
		return "", fmt.Errorf("could not list group deploy tokens for group %d: %w", groupID, err)
	}

	exists := false
	for _, token := range allTokens {
		if token.Name == p.Name {
			exists = true
		}
	}

	gid := gl.ParentGroupID
	if groupID != 0 {
		gid = groupID
	}

	// Remove an existing deploy token
	if exists {
		var existingTokenID int
		for _, t := range allTokens {
			if t.Name == p.Name {
				existingTokenID = t.ID
			}
		}

		_, err := gl.Client.DeployTokens.DeleteGroupDeployToken(gid, existingTokenID)
		if err != nil {
			return "", fmt.Errorf("could not delete existing group deploy token %s: %w", p.Name, err)
		}
	}

	// Create the token
	token, _, err := gl.Client.DeployTokens.CreateGroupDeployToken(gid, &gitlab.CreateGroupDeployTokenOptions{
		Name:     &p.Name,
		Username: &p.Username,
		Scopes:   &p.Scopes,
	})
	if err != nil {
		return "", fmt.Errorf("could not create group deploy token %s: %w", p.Name, err)
	}

	log.Info().Msgf("created group deploy token %s", token.Name)
	return token.Token, nil
}

// CreateProjectDeployToken
func (gl *Wrapper) CreateProjectDeployToken(projectName string, p *DeployTokenCreateParameters) (string, error) {
	projectID, err := gl.GetProjectID(projectName)
	if err != nil {
		return "", fmt.Errorf("could not get project ID for project %s: %w", projectName, err)
	}

	// Check to see if the token already exists
	allTokens, err := gl.ListProjectDeployTokens(projectName)
	if err != nil {
		return "", fmt.Errorf("could not list project deploy tokens for project %s: %w", projectName, err)
	}

	exists := false
	for _, token := range allTokens {
		if token.Name == p.Name {
			exists = true
		}
	}

	if !exists {
		token, _, err := gl.Client.DeployTokens.CreateProjectDeployToken(projectID, &gitlab.CreateProjectDeployTokenOptions{
			Name:     &p.Name,
			Username: &p.Username,
			Scopes:   &p.Scopes,
		})
		if err != nil {
			return "", fmt.Errorf("could not create project deploy token %s: %w", p.Name, err)
		}

		log.Info().Msgf("created project deploy token %s", token.Name)
		return token.Token, nil
	}

	log.Info().Msgf("project deploy token %s already exists - skipping", p.Name)
	return "", nil
}

// ListGroupDeployTokens
func (gl *Wrapper) ListGroupDeployTokens(groupID int) ([]gitlab.DeployToken, error) {
	container := make([]gitlab.DeployToken, 0)
	for nextPage := 1; nextPage > 0; {
		tokens, resp, err := gl.Client.DeployTokens.ListGroupDeployTokens(groupID, &gitlab.ListGroupDeployTokensOptions{
			Page:    nextPage,
			PerPage: 20,
		})
		if err != nil {
			return nil, fmt.Errorf("could not list group deploy tokens for group %d: %w", groupID, err)
		}
		for _, token := range tokens {
			container = append(container, *token)
		}
		nextPage = resp.NextPage
	}

	return container, nil
}

// ListProjectDeployTokens
func (gl *Wrapper) ListProjectDeployTokens(projectName string) ([]gitlab.DeployToken, error) {
	projectID, err := gl.GetProjectID(projectName)
	if err != nil {
		return nil, fmt.Errorf("could not get project ID for project %s: %w", projectName, err)
	}

	container := make([]gitlab.DeployToken, 0)
	for nextPage := 1; nextPage > 0; {
		tokens, resp, err := gl.Client.DeployTokens.ListProjectDeployTokens(projectID, &gitlab.ListProjectDeployTokensOptions{
			Page:    nextPage,
			PerPage: 20,
		})
		if err != nil {
			return nil, fmt.Errorf("could not list project deploy tokens for project %s: %w", projectName, err)
		}
		for _, token := range tokens {
			container = append(container, *token)
		}
		nextPage = resp.NextPage
	}

	return container, nil
}

// DeleteProjectWebhook
func (gl *Wrapper) DeleteProjectWebhook(projectName string, url string) error {
	projectID, err := gl.GetProjectID(projectName)
	if err != nil {
		return fmt.Errorf("could not get project ID for project %s: %w", projectName, err)
	}

	webhooks, err := gl.ListProjectWebhooks(projectID)
	if err != nil {
		return fmt.Errorf("could not list project webhooks for project %s: %w", projectName, err)
	}

	var hookID int
	for _, hook := range webhooks {
		if hook.ProjectID == projectID && hook.URL == url {
			hookID = hook.ID
		}
	}

	if hookID == 0 {
		return fmt.Errorf("no webhooks were found for project %s given search parameters", projectName)
	}

	_, err = gl.Client.Projects.DeleteProjectHook(projectID, hookID)
	if err != nil {
		return fmt.Errorf("could not delete project webhook %s: %w", url, err)
	}

	log.Info().Msgf("deleted hook %s/%s", projectName, url)
	return nil
}

// ListProjectWebhooks returns all webhooks for a project
func (gl *Wrapper) ListProjectWebhooks(projectID int) ([]gitlab.ProjectHook, error) {
	container := make([]gitlab.ProjectHook, 0)
	for nextPage := 1; nextPage > 0; {
		hooks, resp, err := gl.Client.Projects.ListProjectHooks(projectID, &gitlab.ListProjectHooksOptions{
			Page:    nextPage,
			PerPage: 10,
		})
		if err != nil {
			return nil, fmt.Errorf("could not list project webhooks for project %d: %w", projectID, err)
		}

		for _, hook := range hooks {
			container = append(container, *hook)
		}
		nextPage = resp.NextPage
	}
	return container, nil
}

// Runners

// ListGroupRunners returns all registered runners for a parent group
func (gl *Wrapper) ListGroupRunners() ([]gitlab.Runner, error) {
	container := make([]gitlab.Runner, 0)
	for nextPage := 1; nextPage > 0; {
		runners, resp, err := gl.Client.Runners.ListGroupsRunners(gl.ParentGroupID, &gitlab.ListGroupsRunnersOptions{
			ListOptions: gitlab.ListOptions{Page: nextPage, PerPage: 20},
			Type:        gitlab.String("group_type"),
		})
		if err != nil {
			return nil, fmt.Errorf("could not list group runners for group %d: %w", gl.ParentGroupID, err)
		}

		for _, runner := range runners {
			container = append(container, *runner)
		}
		nextPage = resp.NextPage
	}

	return container, nil
}

// DeleteGroupRunners deletes provided runners for a parent group
func (gl *Wrapper) DeleteGroupRunners(runners []gitlab.Runner) error {
	for _, runner := range runners {
		_, err := gl.Client.Runners.DeleteRegisteredRunnerByID(runner.ID)
		if err != nil {
			return fmt.Errorf("could not delete runner %s / %s / %v / %s: %w", runner.Name, runner.IPAddress, runner.ID, runner.Description, err)
		}

		log.Info().Msgf("deleted runner %s / %s / %v / %s\n", runner.Name, runner.IPAddress, runner.ID, runner.Description)
	}

	return nil
}
