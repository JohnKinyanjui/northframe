package help

import (
	"strings"

	generated "northframe.dev/northframe/examples/commerce/.generated/routes/help"
	"northframe.dev/northframe/pkg/web"
)

func Page(ctx *web.Context) (generated.PageProps, error) {
	query := strings.TrimSpace(ctx.Query("q"))
	return generated.PageProps{Questions: filterQuestions(query), Query: query, Total: len(questionBank)}, nil
}
