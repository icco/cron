package goodreads

import (
	"context"

	"github.com/KyleBanks/goodreads"
	gql "github.com/icco/graphql"
	"github.com/machinebox/graphql"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Goodreads contains the scope for doing work against the goodreads API.
type Goodreads struct {
	Token        string
	Log          *logrus.Logger
	GraphQLToken string
}

// GetBooks gets the 100 most recent reviews for Nat.
func (g *Goodreads) GetBooks(ctx context.Context) ([]goodreads.Review, error) {
	c := goodreads.NewClient(g.Token)
	return c.ReviewList("18143346", "read", "date_read", "", "d", 1, 200)
}

// UpsertBooks gets books and uploads them.
func (g *Goodreads) UpsertBooks(ctx context.Context) error {
	reviews, err := g.GetBooks(ctx)
	if err != nil {
		return errors.Wrap(err, "get books")
	}

	for _, r := range reviews {
		err := g.UploadBook(ctx, r.Book)
		if err != nil {
			return errors.Wrap(err, "upload book")
		}
	}

	return nil
}

// UploadBook uploads a single book.
func (g *Goodreads) UploadBook(ctx context.Context, b goodreads.Book) error {
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
		g.Log.WithError(err).Error("error talking to graphql")
		return err
	}

	return nil
}
