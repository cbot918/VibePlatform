package model

type User struct {
	ID        int64  `json:"id"`
	GithubID  int64  `json:"github_id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Email     string `json:"email"`
}
