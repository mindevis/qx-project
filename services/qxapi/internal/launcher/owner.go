package launcher

import "gorm.io/gorm"

type Owner struct {
	UserID string
}

func scopeOwner(q *gorm.DB, owner Owner) *gorm.DB {
	return q.Where("user_id = ?", owner.UserID)
}
