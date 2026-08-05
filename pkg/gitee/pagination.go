package gitee

import "context"

const maxPages = 100

func paginateAll[T any](ctx context.Context, fetch func(ctx context.Context, page, perPage int) ([]T, error)) ([]T, error) {
	var all []T
	page := 1
	perPage := 100
	for page <= maxPages {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		results, err := fetch(ctx, page, perPage)
		if err != nil {
			return nil, err
		}
		all = append(all, results...)
		if len(results) < perPage {
			break
		}
		page++
	}
	return all, nil
}
