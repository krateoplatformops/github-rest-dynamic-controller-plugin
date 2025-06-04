# Krateo Github Plugin for `rest-dynamic-controller`

This web service addresses inconsistencies in the GitHub API's. This Webservice is written for [`rest-dynamic-controller`](https://github.com/krateoplatformops/rest-dynamic-controller/), the dynamic controller instaciated by [`oasgen-provider`](https://github.com/krateoplatformops/oasgen-provider).

## Summary

- [Krateo Github Plugin for `rest-dynamic-controller`](#krateo-github-plugin-for-rest-dynamic-controller)
  - [Summary](#summary)
  - [Overview](#overview)
  - [API](#api)
  - [Examples](#examples)
    - [Receiving notifications](#receiving-notifications)
    - [Listing last events](#listing-last-events)
  - [Configuration](#configuration)

## API

### 1) Get User Permission in a Repository

- **Endpoint:** `/repository/{owner}/{repo}/collaborators/{username}/permission`
- **Description:** Retrieves the permission level of a specified user in a given repository. The endpoint extracts the `owner`, `repo`, and `username` from the request path, checks if the user is a collaborator, and then fetches the user's permission level from the GitHub API. The result is returned in the response body.

Sample response:
```json
{
  "html_url":"<REDACTED>",
  "id": 12345678,
  "permission":"admin",
  "permissions":{
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

Note: since the root level field `permission` in the GitHub API response can be {`admin`, `write`, `read`}, the plugin will convert it using the `role_name` field which instead can be {`admin`, `maintain`, `push`, `triage`, `pull`}. The `permissions` field is also included to provide detailed permission levels.

### 2) Get Team Permission in a Repository

- **Endpoint:** `/teamrepository/orgs/{org}/teams/{team_slug}/repos/{owner}/{repo}`
- **Description:** Retrieves the permission level of a specified team in a given repository. The endpoint extracts the `organization`, `team_slug`, `owner`, and `repo` from the request path, logs the API call, and forwards the request to the GitHub API with the necessary headers. The response from GitHub is processed to adjust the repository permissions before being returned to the client.

For more detailed information about all the API endpoints, please refer to the Swagger documentation available at `/swagger/index.html`.

## Authentication
Since it's a wrapper for GitHub API, it supports the same authentication methods provided by GitHub to interact with GitHub resources.