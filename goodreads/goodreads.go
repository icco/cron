package goodreads

import (
	"context"
	"fmt"

	"github.com/KyleBanks/goodreads"
	"github.com/KyleBanks/goodreads/responses"
	"github.com/icco/cron/shared"
	gql "github.com/icco/graphql"
	"github.com/machinebox/graphql"
	"go.uber.org/zap"
)

// Goodreads contains the scope for doing work against the goodreads API.
type Goodreads struct {
	shared.Config

	Token        string
	GraphQLToken string
}

// GetBooks gets the 100 most recent reviews for Nat.
func (g *Goodreads) GetBooks(ctx context.Context) ([]responses.Review, error) {
	c := goodreads.NewClient(g.Token)
	return c.ReviewList("18143346", "read", "date_read", "", "d", 1, 200)
}

// UpsertBooks gets books and uploads them.
func (g *Goodreads) UpsertBooks(ctx context.Context) error {
	reviews, err := g.GetBooks(ctx)
	if err != nil {
		return fmt.Errorf("get books: %w", err)
	}

	for _, r := range reviews {
		err := g.UploadBook(ctx, r.Book)
		if err != nil {
			return fmt.Errorf("upload book: %w", err)
		}
	}

	g.Log.Infow("uploaded books", "reviews", len(reviews))

	return nil
}

// UploadBook uploads a single book.
func (g *Goodreads) UploadBook(ctx context.Context, b responses.AuthorBook) error {
	book := gql.EditBook{
		ID:    &b.ID,
		Title: &b.Title,
	}

	gqlClient := graphql.NewClient("https://graphql.natwelch.com/graphql")
	mut := `
  mutation ($b: EditBook!) {
      upsertBook(input: $b) {
        id
      }
    }
  `

	req := graphql.NewRequest(mut)
	req.Var("b", book)
	req.Header.Add("X-API-AUTH", g.GraphQLToken)
	req.Header.Add("User-Agent", "icco-cron/1.0")
	err := gqlClient.Run(ctx, req, nil)
	if err != nil {
		g.Log.Errorw("error talking to graphql", zap.Error(err))
		return err
	}

	return nil
}
