package help

import (
	"strings"

	generated "github.com/JohnKinyanjui/northframe/examples/commerce/.generated/routes/help"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func Page(ctx *web.Context) (generated.PageProps, error) {
	query := strings.TrimSpace(ctx.Query("q"))
	return generated.PageProps{Questions: filterQuestions(query), Query: query, Total: len(questionBank)}, nil
}
