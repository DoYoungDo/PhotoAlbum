package storage

import "time"

// Photo 图片模型
type Photo struct {
	ID           int64      `json:"id"`
	UUID         string     `json:"uuid"`          // 对应磁盘文件名（不含扩展名）
	OriginalName string     `json:"original_name"` // 用户上传时的原始文件名
	MimeType     string     `json:"mime_type"`     // image/jpeg 等
	Size         int64      `json:"size"`          // 文件大小（字节）
	Width        int        `json:"width"`
	Height       int        `json:"height"`
	TakenAt      time.Time  `json:"taken_at"` // ���摄时间（EXIF 或文件创建时间）
	UploadedAt   time.Time  `json:"uploaded_at"`
	UploadedBy   int64      `json:"uploaded_by"` // 关联 users.id
	DeletedAt    *time.Time `json:"deleted_at"`  // nil 表示未删除
	DeletedBy    *int64     `json:"deleted_by"`  // nil 表示未删除
}

// Album 相册模型
type Album struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	CoverPhotoID *int64    `json:"cover_photo_id"` // nil 时自动取最新图片
	CreatedBy    int64     `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
	PhotoCount   int       `json:"photo_count"` // 非数据库字段，查询时聚合
}

// AlbumPhoto 相册与图片的关联（多对多）
type AlbumPhoto struct {
	AlbumID int64     `json:"album_id"`
	PhotoID int64     `json:"photo_id"`
	AddedAt time.Time `json:"added_at"`
}

// ShareLink 分享链接模型
type ShareLink struct {
	ID        int64      `json:"id"`
	Token     string     `json:"token"`     // 随机生成的访问 token
	Type      string     `json:"type"`      // "photo" 或 "album"
	TargetID  int64      `json:"target_id"` // 图片ID 或 相册ID
	CreatedBy int64      `json:"created_by"`
	ExpiresAt *time.Time `json:"expires_at"` // nil 表示永不过期
	CreatedAt time.Time  `json:"created_at"`
}

// ShareType 分享类型常量
const (
	ShareTypePhoto = "photo"
	ShareTypeAlbum = "album"
)

// PhotoPage 图片分页结果（游标分页）
type PhotoPage struct {
	Photos     []*Photo `json:"photos"`
	NextCursor string   `json:"next_cursor"` // 空字符串表示没有更多
	HasMore    bool     `json:"has_more"`
}

// ListPhotosParams 查询图片参数
type ListPhotosParams struct {
	UserID      int64  // 必填，用户隔离
	Cursor      string // 游标，空表示从头开始
	Limit       int    // 每页数量，默认30
	OnlyTrashed bool   // true 时查询回收站
}

// ListAlbumPhotosParams 查询相册内图片参数
type ListAlbumPhotosParams struct {
	AlbumID int64
	UserID  int64
	Cursor  string
	Limit   int
}
