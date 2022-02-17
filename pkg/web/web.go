package web

import (
	"bytes"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

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
		Body      string `json:"body"`
		//UpdatedAt string `json:"updated_at"`
		User      struct {
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

func StartWeb(log *zap.SugaredLogger, lastPause *time.Time, githubToken string){
	ghWebHookPath := fmt.Sprintf("/webhooks/github/%s", githubToken)
	e := echo.New()
	e.HideBanner = true

	e.GET("/healthz", func(c echo.Context) (err error){
		return c.String(http.StatusOK, "ok")
	})

	e.POST(ghWebHookPath, func(c echo.Context) (err error) {
		isComment := isIssueComment(c.Request().Header)
		if isComment {
			u := new(GitHubWebHookJSON)
			if err := c.Bind(u); err != nil {
				log.Warn("couldn't bind JSON.. are you sure github payload looks correct?")
			}
			log.Debugw("webhook",
				"status", "body_checked",
				"reason", "is a comment event",
				"data", u,
			)
			if stringContains(u.Comment.Body, "laminar pause") {
				// someone told laminar to be a good boy... update "lastPause"
				log.Infow("webhook",
					"status", "laminar instructed to pause",
				)
				*lastPause = time.Now()
				return c.String(http.StatusOK, "laminar paused")
			}
		} else {
			log.Infow("webhook",
				"status", "ignored",
				"reason", "not comment",
			)
		}
		return c.String(http.StatusOK, "ok")
	})

	e.Logger.Fatal(e.Start(":8080"))
}

func stringContains(comment, value string) bool {

	if bytes.Contains(
		[]byte(comment),
		[]byte(value),
	) {
		return true
	}
	return false
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
