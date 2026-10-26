package data

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/taoworklabs/mmerp/internal/platform"
)

// Page is one page of a dynamic list: its rows and the true total matching the
// query, independent of paging.
type Page[R any] struct {
	Items []R
	Total int64
}

// Run executes q against l. The page and its count run together in one round
// trip, except on page 1: there the page runs alone first, and the count
// only follows if the page came back full — a page shorter than the limit
// already is the true total.
func (l List[R]) Run(ctx context.Context, q Query) (Page[R], error) {
	v, hasViewer := viewerFrom(ctx)
	page, count, empty, err := l.Compile(q, v, hasViewer)
	if err != nil {
		return Page[R]{}, err
	}
	if empty {
		return Page[R]{Items: []R{}}, nil
	}

	conn := platform.DBFrom(ctx)

	if q.Paging.Page == 1 {
		items, err := l.runPage(ctx, conn, page)
		if err != nil {
			return Page[R]{}, err
		}
		if int32(len(items)) < q.Paging.Limit() {
			return Page[R]{Items: items, Total: int64(len(items))}, nil
		}
		total, err := l.runCount(ctx, conn, count)
		if err != nil {
			return Page[R]{}, err
		}
		return Page[R]{Items: items, Total: total}, nil
	}

	return l.runBatch(ctx, conn, page, count)
}

func (l List[R]) runPage(ctx context.Context, conn platform.Conn, page Compiled) ([]R, error) {
	rows, err := conn.Query(platform.WithQueryName(ctx, page.Name, page.Shape), page.SQL, page.Args...)
	if err != nil {
		return nil, Normalize(err)
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[R])
	return items, Normalize(err)
}

func (l List[R]) runCount(ctx context.Context, conn platform.Conn, count Compiled) (int64, error) {
	var total int64
	err := conn.QueryRow(platform.WithQueryName(ctx, count.Name, count.Shape), count.SQL, count.Args...).Scan(&total)
	return total, Normalize(err)
}

// runBatch sends the page and its count together: pages past 1 cannot skip the
// count just because the page they land on happens to be short.
func (l List[R]) runBatch(ctx context.Context, conn platform.Conn, page, count Compiled) (Page[R], error) {
	batch := &pgx.Batch{}
	batch.Queue(page.SQL, page.Args...)
	batch.Queue(count.SQL, count.Args...)

	br := conn.SendBatch(platform.WithQueryName(ctx, page.Name, page.Shape), batch)
	defer func() { _ = br.Close() }()

	rows, err := br.Query()
	if err != nil {
		return Page[R]{}, Normalize(err)
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByName[R])
	if err != nil {
		return Page[R]{}, Normalize(err)
	}

	var total int64
	if err := br.QueryRow().Scan(&total); err != nil {
		return Page[R]{}, Normalize(err)
	}
	return Page[R]{Items: items, Total: total}, nil
}
