package web

import (
	"bytes"
	"fmt"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

type HookData struct {
	Events []Event
}

type Event struct {
	Name string
}

type GitHubWebHookJSON struct {
	Action  string `json:"action"`
	Comment struct {
		Body string `json:"body"`
		// UpdatedAt string `json:"updated_at"`
		User struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"comment"`
	Issue struct {
		HTMLURL string `json:"html_url"`
		Number  int64  `json:"number"`
	} `json:"issue"`
	Organization struct {
		Login string `json:"login"`
	} `json:"organization"`
	Repository struct {
		FullName string `json:"full_name"`
		Name     string `json:"name"`
	} `json:"repository"`
	Sender struct {
		AvatarURL string `json:"avatar_url"`
		Login     string `json:"login"`
	} `json:"sender"`
}

type DockerBuildJSON struct {
	URL               string `json:"url"`
	DockerRegistryURL string `json:"docker_registry_url"`
}

func stringContains(comment, value string) bool {
	return bytes.Contains(
		[]byte(comment),
		[]byte(value),
	)
}

type Client struct {
	logger      *zap.SugaredLogger
	PauseChan   chan time.Time
	BuildChan   chan DockerBuildJSON
	githubToken string
}

func New(logger *zap.SugaredLogger, githubToken string) *Client {
	return &Client{
		logger:      logger,
		PauseChan:   make(chan time.Time),
		BuildChan:   make(chan DockerBuildJSON),
		githubToken: githubToken,
	}
}

func (client *Client) StartWeb() {
	e := echo.New()

	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			client.logger.Infow("request",
				"URI", v.URI,
				"status", v.Status,
				"RemoteAddr", c.Request().RemoteAddr,
			)
			return nil
		},
	}))
	e.HideBanner = true
	e.GET("/healthz", func(c echo.Context) (err error) {
		return c.String(http.StatusOK, "ok")
	})
	e.POST(
		fmt.Sprintf("/webhooks/github/%s", client.githubToken),
		client.handleGithubWebhook,
	)
	e.POST(
		"/webhooks/build/docker",
		client.handleDockerBuildWebhook,
	)
	e.Logger.Fatal(e.Start(":8080"))
}

func (client *Client) handleGithubWebhook(ctx echo.Context) (err error) {
	isComment := isIssueComment(ctx.Request().Header)
	if isComment {
		u := new(GitHubWebHookJSON)
		if err := ctx.Bind(u); err != nil {
			client.logger.Warn("couldn't bind JSON.. are you sure github payload looks correct?")
		}
		client.logger.Debugw("webhook",
			"status", "body_checked",
			"reason", "is a comment event",
			"data", u,
		)
		if stringContains(u.Comment.Body, "[laminar pause]") {
			// someone told laminar to be a good boy... update "lastPause"
			client.logger.Infow("webhook",
				"status", "laminar instructed to pause",
			)
			client.PauseChan <- time.Now()
			return ctx.String(http.StatusOK, "laminar paused")
		}
	} else {
		client.logger.Infow("webhook",
			"status", "ignored",
			"reason", "not comment",
		)
	}
	return ctx.String(http.StatusOK, "ok")
}

func (client *Client) handleDockerBuildWebhook(ctx echo.Context) (err error) {
	client.logger.Infow("webhook",
		"status", "laminar told there is a new build",
	)
	u := new(DockerBuildJSON)
	if err = ctx.Bind(u); err != nil {
		client.logger.Warn("couldn't bind JSON.. are you sure github payload looks correct?")
	} else {
		client.BuildChan <- *u
		return ctx.String(http.StatusOK, "build webhook received")
	}
	return err
}

func isIssueComment(input http.Header) bool {
	// returns true if any of the event type is "issue_comment"
	httpHeader := "X-Github-Event"
	// range over the k/v map of all input headers
	for k, v := range input {
		// we only care about one of these keys
		if k == httpHeader {
			// the headers values are a slice, lets see if any match
			for _, headerValue := range v {
				if headerValue == "issue_comment" {
					// this event is a comment on a PR
					return true
				}
			}
		}
	}
	return false
}
