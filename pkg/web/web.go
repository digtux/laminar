package web

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"github.com/digtux/laminar/pkg/cfg"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echopprof "github.com/sevenNt/echo-pprof"
	"go.uber.org/zap"
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
	logger        *zap.SugaredLogger
	PauseChan     chan time.Time
	BuildChan     chan DockerBuildJSON
	githubToken   string
	listenAddress string
	config        cfg.Config
}

func New(logger *zap.SugaredLogger, cfg cfg.Config) *Client {
	return &Client{
		logger:        logger,
		PauseChan:     make(chan time.Time),
		BuildChan:     make(chan DockerBuildJSON),
		githubToken:   cfg.Global.GitHubToken,
		listenAddress: cfg.Global.WebAddress,
		config:        cfg,
	}
}

func (client *Client) StartWeb() {
	e := echo.New()

	if client.config.Global.WebDebug {
		echopprof.Wrap(e)
	}
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			client.logger.Infow("laminar.http",
				"laminar.http.URI", v.URI,
				"laminar.http.status", v.Status,
				"laminar.http.RemoteAddr", c.Request().RemoteAddr,
			)
			return nil
		},
	}))
	e.HidePort = true
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
	client.logger.Infow("laminar web listener started",
		"laminar.address", client.listenAddress)

	if err := e.Start(client.listenAddress); err != http.ErrServerClosed {
		client.logger.Fatal(err)
	}
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
