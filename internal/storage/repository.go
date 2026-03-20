package storage

// Repository 定义所有数据库操作接口
// 具体实现可以是 SQLite、PostgreSQL 等，业务层只依赖此接口
type Repository interface {
	// --- 图片 ---

	// SavePhoto 保存图片记录，成功后填充 photo.ID
	SavePhoto(photo *Photo) error

	// GetPhotoByID 按 ID 查询图片（不返回已删除的）
	GetPhotoByID(id int64, userID int64) (*Photo, error)

	// GetPhotoByUUID 按 UUID 查询图片（不返回已删除的）
	GetPhotoByUUID(uuid string, userID int64) (*Photo, error)

	// GetPhotoByUUIDAny 按 UUID 查询图片（包含已软删除，用于文件服务）
	GetPhotoByUUIDAny(uuid string, userID int64) (*Photo, error)

	// ListPhotos 查询用户图片（时间线，游标分页，不包含已删除）
	ListPhotos(params ListPhotosParams) (*PhotoPage, error)

	// ListTrashedPhotos 查询回收站图片（游标分页）
	ListTrashedPhotos(params ListPhotosParams) (*PhotoPage, error)

	// SoftDeletePhoto 软删除图片
	SoftDeletePhoto(id int64, userID int64, deletedBy int64) error

	// RestorePhoto 从回收站恢复图片
	RestorePhoto(id int64, userID int64) error

	// HardDeletePhoto 彻底删除图片记录（清空回收站时使用）
	HardDeletePhoto(id int64, userID int64) error

	// HardDeleteTrashedPhotos 清空指定用户的整个回收站
	HardDeleteTrashedPhotos(userID int64) ([]string, error) // 返回需要删除的 UUID 列表

	// --- 相册 ---

	// CreateAlbum 创建相册，成功后填充 album.ID
	CreateAlbum(album *Album) error

	// GetAlbumByID 按 ID 查询相册
	GetAlbumByID(id int64, userID int64) (*Album, error)

	// ListAlbums 查询用户的所有相册
	ListAlbums(userID int64) ([]*Album, error)

	// UpdateAlbum 更新相册信息（名称、描述、封面）
	UpdateAlbum(album *Album) error

	// DeleteAlbum 删除相册（不删除图片本身）
	DeleteAlbum(id int64, userID int64) error

	// --- 相册图片关联 ---

	// AddPhotoToAlbum 将图片添加到相册
	AddPhotoToAlbum(albumID int64, photoID int64, userID int64) error

	// RemovePhotoFromAlbum 从相册移除图片
	RemovePhotoFromAlbum(albumID int64, photoID int64, userID int64) error

	// ListAlbumPhotos 查询相册内的图片（游标分页，按拍摄时间排序）
	ListAlbumPhotos(params ListAlbumPhotosParams) (*PhotoPage, error)

	// IsPhotoInAlbum 检查图片是否在相册中
	IsPhotoInAlbum(albumID int64, photoID int64) (bool, error)

	// --- 分享链接 ---

	// CreateShareLink 创建分享链接，成功后填充 link.ID
	CreateShareLink(link *ShareLink) error

	// GetShareLinkByToken 按 token 查询分享链接（自动过滤已过期）
	GetShareLinkByToken(token string) (*ShareLink, error)

	// ListShareLinks 查询用户创建的所有分享链接
	ListShareLinks(userID int64) ([]*ShareLink, error)

	// DeleteShareLink 删除分享链接
	DeleteShareLink(id int64, userID int64) error

	// --- 生命周期 ---

	// Close 关闭数据库连接
	Close() error
}
