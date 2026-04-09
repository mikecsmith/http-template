package handle

import (
	"context"
	"fmt"
	"net/http"

	"github.com/mikecsmith/httplab/internal/request"
	"github.com/mikecsmith/httplab/internal/respond"
)

type helloWorldReq struct {
	Name string `json:"name"`
}

func (v helloWorldReq) Valid(ctx context.Context) map[string]string {
	problems := make(map[string]string)
	if v.Name == "" {
		problems["name"] = "is required"
	}
	return problems
}

type helloWorldRes struct {
	Message string `json:"message"`
}

// HelloWorldGet is a demo handler for GET requests
func HelloWorldGet() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		respond.WithOK(w, r, helloWorldRes{Message: "Hello World!"})
	}
}

func HelloWorldPost() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, problems, err := request.DecodeValid[helloWorldReq](r)
		if err != nil {
			respond.With(w, r, http.StatusUnprocessableEntity, problems)
			return
		}
		if len(problems) > 0 {
			respond.WithError(w, r, respond.ErrBadRequest.WithDetails(problems))
			return
		}
		respond.WithOK(w, r, helloWorldRes{Message: fmt.Sprintf("Hello %s!", req.Name)})
	}
}
