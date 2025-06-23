# Krateo Github Plugin for `rest-dynamic-controller`

This web service addresses some inconsistencies in the GitHub API's. 
This web service is written for [`rest-dynamic-controller`](https://github.com/krateoplatformops/rest-dynamic-controller/), the dynamic controller instaciated by [`oasgen-provider`](https://github.com/krateoplatformops/oasgen-provider).
In particular, this plugin is design to work alongside the [`github-provider-kog`](https://github.com/krateoplatformops/github-provider-kog-chart).

## Summary

- [Summary](#summary)
- [API Endpoints](#api-endpoints)
  - [Collaborator](#collaborator)
    - [GET](#get)
    - [POST](#post)
    - [PATCH](#patch)
    - [DELETE](#delete)
  - [TeamRepo](#teamrepo)
- [Swagger Documentation](#swagger-documentation)
- [Authentication](#authentication)

## API Endpoints

### Collaborator

#### GET 

- **Endpoint:** 
`GET /repository/{owner}/{repo}/collaborators/{username}/permission`

- **Description:** 
Retrieves the permission level of a specified user in a given repository. 
The endpoint extracts the `owner`, `repo`, and `username` from the request path, makes a first API call to the GitHub API to check if the user is a collaborator with the endpoint `/repos/{owner}/{repo}/collaborators/{username}`, and then makes a second API call to retrieve the permission level of the user with the endpoint `/repos/{owner}/{repo}/collaborators/{username}/permission`. 
In addition, the response is adjusted to fix inconsistencies in the GitHub API response (legacy issues), where the `permission` field is returned as `admin`, `write`,`read`, `triage` or `maintain`, but the plugin converts it to `admin`, `push`, `pull`, `triage` or `maintain`. 
The `permissions`, `html_url` fields are also included at the root level of the response.

- **Why is needed?**  
After removing a collaborator from a repository, attempting to verify the status of the collaborator using the GitHub API endpoint `/repos/{owner}/{repo}/collaborators/{username}/permission` results in a 200 OK response instead of the expected 404 Not Found.

- **Sample response**:
```json
{
  "html_url":"<REDACTED>",  // Adjusted field
  "id": 12345678,           // Adjusted field
  "permission":"admin",     // Adjusted field
  "permissions":{           // Adjusted field
    "admin":true,
    "maintain":true,
    "pull":true,
    "push":true,
    "triage":true
  },
  "role_name":"admin",
  "user":{
    "avatar_url":"<REDACTED>",
    "events_url":"<REDACTED>",
    "followers_url":"<REDACTED>",
    "following_url":"<REDACTED>",
    "gists_url":"<REDACTED>",
    "gravatar_id":"<REDACTED>",
    "html_url":"<REDACTED>",
    "id":12345678,
    "login":"<REDACTED>",
    "node_id":"<REDACTED>",
    "organizations_url":"<REDACTED>",
    "permissions":{
      "admin":true,
      "maintain":true,
      "pull":true,
      "push":true,
      "triage":true
    },
    "received_events_url":"<REDACTED>",
    "repos_url":"<REDACTED>",
    "role_name":"admin",
    "site_admin":false,
    "starred_url":"<REDACTED>",
    "subscriptions_url":"<REDACTED>",
    "type":"User",
    "url":"<REDACTED>",
    "user_view_type":"public"
  }
}
```

#### POST

- **Endpoint:** 
`POST /repository/{owner}/{repo}/collaborators/{username}`

- **Description:** 
Add a repository collaborator with a specified permission level.
If the user is not a member of the organization, an invitatation to be a repository collaborator is sent ("external collaborator").


- **Why is needed?**
This endpoint is needed to differentiate between adding a collaborator to a repository and inviting a user not in the organization to become a collaborator of a repository.

#### PATCH

- **Endpoint:** 
`PATCH /repository/{owner}/{repo}/collaborators/{username}`

- **Description:** 
Add a repository collaborator with a specified permission level.
If the user is not a member of the organization, an invitatation to be a repository collaborator is sent ("external collaborator").


- **Why is needed?**
This endpoint is needed to differentiate between updateing a collaborator's permission level in a repository and updating the invitation's permission level for a user not in the organization.

#### DELETE
- **Endpoint:**
`DELETE /repository/{owner}/{repo}/collaborators/{username}`

- **Description:**
Removes a repository collaborator.
If the user is not a member of the organization, the invitation to be a repository collaborator is revoked ("external collaborator").

- **Why is needed?**
This endpoint is needed to differentiate between removing a collaborator from a repository and revoking the invitation for a user not in the organization to become a collaborator of a repository.

### TeamRepo

#### 2) GET

- **Endpoint:** 
`/teamrepository/orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}`

- **Description:** 
Retrieves the permission level of a specified team in a given repository. The endpoint extracts the `organization`, `team_slug`, `owner`, and `repo` from the request path, logs the API call, and forwards the request to the GitHub API with the necessary headers. The response from GitHub is processed to adjust the repository permissions before being returned to the client.

- **Sample response**:

```json
{
  "allow_auto_merge":false,
  "allow_forking":false,
  "allow_merge_commit":true,
  "allow_rebase_merge":true,
  "allow_squash_merge":true,
  "archive_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/{archive_format}{/ref}",
  "archived":false,
  "assignees_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/assignees{/user}",
  "blobs_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/git/blobs{/sha}",
  "branches_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/branches{/branch}",
  "clone_url":"https://github.com/krateoplatformops-test/test-teamrepo.git",
  "collaborators_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/collaborators{/collaborator}",
  "comments_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/comments{/number}",
  "commits_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/commits{/sha}",
  "compare_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/compare/{base}...{head}",
  "contents_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/contents/{+path}",
  "contributors_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/contributors",
  "created_at":"2025-06-10T17:15:43Z",
  "default_branch":"main",
  "delete_branch_on_merge":false,
  "deployments_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/deployments",
  "description":null,
  "disabled":false,
  "downloads_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/downloads",
  "events_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/events",
  "fork":false,
  "forks":0,
  "forks_count":0,
  "forks_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/forks",
  "full_name":"krateoplatformops-test/test-teamrepo",
  "git_commits_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/git/commits{/sha}",
  "git_refs_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/git/refs{/sha}",
  "git_tags_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/git/tags{/sha}",
  "git_url":"git://github.com/krateoplatformops-test/test-teamrepo.git",
  "has_downloads":true,
  "has_issues":true,
  "has_pages":false,
  "has_projects":true,
  "has_wiki":false,
  "homepage":null,
  "hooks_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/hooks",
  "html_url":"https://github.com/krateoplatformops-test/test-teamrepo",
  "id":999717066,
  "issue_comment_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/issues/comments{/number}",
  "issue_events_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/issues/events{/number}",
  "issues_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/issues{/number}",
  "keys_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/keys{/key_id}",
  "labels_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/labels{/name}",
  "language":null,
  "languages_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/languages",
  "license":null,
  "merges_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/merges",
  "milestones_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/milestones{/number}",
  "mirror_url":null,
  "name":"test-teamrepo",
  "node_id":"R_kgDOO5Z4yg",
  "notifications_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/notifications{?since,all,participating}",
  "open_issues":0,
  "open_issues_count":0,
  "owner":"krateoplatformops-test", // Adjusted field
  "permission":"pull",              // Adjusted field
  "private":true,
  "pulls_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/pulls{/number}",
  "pushed_at":"2025-06-10T17:15:43Z",
  "releases_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/releases{/id}",
  "role_name":"read",
  "size":0,
  "ssh_url":"git@github.com:krateoplatformops-test/test-teamrepo.git",
  "stargazers_count":0,
  "stargazers_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/stargazers",
  "statuses_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/statuses/{sha}",
  "subscribers_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/subscribers",
  "subscription_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/subscription",
  "svn_url":"https://github.com/krateoplatformops-test/test-teamrepo",
  "tags_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/tags",
  "teams_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/teams",
  "topics":[],
  "trees_url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo/git/trees{/sha}",
  "updated_at":"2025-06-10T17:15:43Z",
  "url":"https://api.github.com/repos/krateoplatformops-test/test-teamrepo",
  "visibility":"private",
  "watchers":0,
  "watchers_count":0
}   
```

> [!NOTE]  
> Since the root level field `permission` in the GitHub API response can be {`admin`, `write`, `read`}, the plugin will convert it using the `role_name` field which instead can be {`admin`, `maintain`, `push`, `triage`, `pull`}. The `owner` field is also adjusted to be just a string instead of an object. The `permissions` field (object) is not included in this response.

## Swagger Documentation

For more detailed information about the API endpoints, please refer to the Swagger documentation available at `/swagger/index.html` endpoint of the service.

## Authentication

The plugin will forward the `Authorization` header passed in the request to this plugin to the GitHub API.
