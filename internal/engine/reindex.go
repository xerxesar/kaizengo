package engine

import (
	"context"

	"kaizengo/internal/platform/search"
)

func (s *modelService) reindexAll(ctx context.Context) (int, error) {
	if !search.ShouldIndex(s.spec.Name, s.model.Name) {
		return 0, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT `+s.selectList()+` FROM `+s.qtable()+` WHERE deleted = false`,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		rec, err := s.scan(rows)
		if err != nil {
			return count, err
		}
		orgID, _ := rec["orgId"].(string)
		doc, ok := search.BuildDocument(s.spec.Name, s.model.Name, orgID, rec)
		if !ok {
			continue
		}
		if err := search.Upsert(ctx, doc); err != nil {
			return count, err
		}
		count++
	}
	return count, rows.Err()
}
