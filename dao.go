package xf

import (
	"time"
)

func regulateUpdates(updates map[string]any) {
	delete(updates, FieldCreatedAt)
	delete(updates, FieldIsDeleted)
	delete(updates, FieldID)

	updates[FieldUpdatedAt] = time.Now()
}

func configurePage(page *PageMeta) {
	if page == nil {
		return
	}

	// paging
	if page.Page == nil {
		page.Page = new(int64)
	}

	pageNo := *page.Page
	if pageNo <= 0 {
		pageNo = 1
		*page.Page = pageNo
	}

	if page.Size <= 0 {
		page.Size = 10
	} else if page.Size > 1000 {
		page.Size = 1000
	}
}

type DAO[T any, P CommonModel[T]] interface {
	MustGetList(page *PageMeta, fields map[string]any) []P
	MustGetPage(page *PageMeta, fields map[string]any) []P
	MustCount(page *PageMeta) int64
	MustGet(filter, fields map[string]any) P
	Exist(filter map[string]any) bool
	MustAdd(doc P)
	MustUpdate(filter, updates map[string]any, advancedUpdates any) int64 // restrict to update one record
	MustUpdateMany(filter, updates map[string]any, advancedUpdates any) int64
	MustSave(doc P)
	MustSoftDelete(filter map[string]any) int64
	MustSoftDeleteMany(filter map[string]any) int64
	MustHardDelete(filter map[string]any) int64
	MustHardDeleteMany(filter map[string]any) int64
}
