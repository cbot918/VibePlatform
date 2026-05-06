package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
	oauthgithub "golang.org/x/oauth2/github"

	"github.com/yjtech/vibeplatform/internal/model"
)

type GithubClient struct {
	oauthCfg *oauth2.Config
	apiBase  string
}

func NewGithubClient(clientID, clientSecret, redirectURL string) *GithubClient {
	return &GithubClient{
		oauthCfg: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"user:email"},
			Endpoint:     oauthgithub.Endpoint,
		},
		apiBase: "https://api.github.com",
	}
}

func (g *GithubClient) AuthCodeURL(state string) string {
	return g.oauthCfg.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

func (g *GithubClient) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.oauthCfg.Exchange(ctx, code)
}

type githubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Email     string `json:"email"`
}

func (g *GithubClient) FetchUser(ctx context.Context, token *oauth2.Token) (*model.User, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", g.apiBase+"/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch github user: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var gu githubUser
	if err := json.NewDecoder(resp.Body).Decode(&gu); err != nil {
		return nil, fmt.Errorf("decode github user: %w", err)
	}

	return &model.User{
		GithubID:  gu.ID,
		Login:     gu.Login,
		Name:      gu.Name,
		AvatarURL: gu.AvatarURL,
		Email:     gu.Email,
	}, nil
}
