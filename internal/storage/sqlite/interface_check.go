package sqlite

import "photoalbum/internal/storage"

// 编译期验证 DB 实现了 Repository 接口
var _ storage.Repository = (*DB)(nil)
