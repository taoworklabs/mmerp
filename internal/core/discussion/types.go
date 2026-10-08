package discussion

import (
	"net/http"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// maxBody bounds one comment, in characters.
const maxBody = 4000

// actionComment is the allowed action to add a comment.
const actionComment = "comment"

// Discussion is the comments of a record, oldest first; allowed_actions holds comment
// when the actor may add one.
type Discussion struct {
	AllowedActions []string  `json:"allowed_actions" nullable:"false"`
	Items          []Comment `json:"items" nullable:"false"`
	MaxLength      int       `json:"max_length" doc:"Characters one comment may hold"`
}

type Comment struct {
	ID         int64  `json:"id"`
	AuthorName string `json:"author_name"`
	Body       string `json:"body" doc:"Plain text; line breaks kept"`
	CreatedAt  string `json:"created_at" format:"date-time"`
}

// Mentionable is a user who may be mentioned on a record: one who may view it.
type Mentionable struct {
	Login string `json:"login"`
	Name  string `json:"name"`
}

var (
	ErrCommentTooLong = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "comment_too_long", Params: map[string]any{"max": maxBody}}
	errBlank          = &platform.Error{Status: http.StatusUnprocessableEntity, Code: "invalid_request", Params: map[string]any{"fields": []string{"body"}}}
)
