package pq

import "github.com/elastic/go-txfile"

func getPage(tx *txfile.Tx, id txfile.PageID) ([]byte, error) {
	page, err := tx.Page(id)
	if err != nil {
		return nil, err
	}

	return page.Bytes()
}

func withPage(tx *txfile.Tx, id txfile.PageID, fn func([]byte) error) error {
	page, err := getPage(tx, id)
	if err != nil {
		return err
	}
	return fn(page)
}

func readPageByID(accessor *access, pool *pagePool, id txfile.PageID) (*page, error) {
	tx := accessor.BeginRead()
	defer tx.Close()

	var page *page
	return page, withPage(tx, id, func(buf []byte) error {
		page = pool.NewPageWith(id, buf)
		return nil
	})
}

func idLess(a, b uint64) bool {
	return int64(a-b) < 0
}

func idLessEq(a, b uint64) bool {
	return a == b || idLess(a, b)
}
