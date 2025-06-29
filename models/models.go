package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// JSONB для PostgreSQL
type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONB)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into JSONB", value)
	}
	return json.Unmarshal(bytes, j)
}

// Admin модель для администраторов
type Admin struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    int64     `gorm:"uniqueIndex;not null" json:"user_id"`
	Username  *string   `json:"username"`
	FirstName *string   `json:"first_name"`
	LastName  *string   `json:"last_name"`
	CreatedAt time.Time `json:"created_at"`
}

func (Admin) TableName() string {
	return "admins"
}

// BannedUser модель для заблокированных пользователей
type BannedUser struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    int64     `gorm:"uniqueIndex;not null" json:"user_id"`
	Username  *string   `json:"username"`
	FirstName *string   `json:"first_name"`
	LastName  *string   `json:"last_name"`
	BannedAt  time.Time `json:"banned_at"`
	BannedBy  *int64    `json:"banned_by"`
	Reason    *string   `gorm:"size:500" json:"reason"`
}

func (BannedUser) TableName() string {
	return "banned_users"
}

// Meme модель для мемов
type Meme struct {
	ID                 uint          `gorm:"primaryKey" json:"id"`
	FileID             string        `gorm:"uniqueIndex;not null;size:255" json:"file_id"`
	FileType           string        `gorm:"not null;size:50" json:"file_type"`
	UserID             int64         `gorm:"not null" json:"user_id"`
	Username           *string       `gorm:"size:255" json:"username"`
	FirstName          *string       `gorm:"size:255" json:"first_name"`
	LastName           *string       `gorm:"size:255" json:"last_name"`
	CreatedAt          time.Time     `json:"created_at"`
	ForwardedToChannel bool          `gorm:"default:false" json:"forwarded_to_channel"`
	ChannelMessageID   *int          `json:"channel_message_id"`
	MediaGroupID       *string       `gorm:"size:255" json:"media_group_id"`
	Comments           []MemeComment `gorm:"foreignKey:MemeID;constraint:OnDelete:CASCADE" json:"comments"`
	Histories          []MemeHistory `gorm:"foreignKey:MemeID;constraint:OnDelete:CASCADE" json:"histories"`
}

func (Meme) TableName() string {
	return "memes"
}

// MemeComment модель для комментариев к мемам
type MemeComment struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	MemeID    uint      `gorm:"not null;index" json:"meme_id"`
	MessageID *int      `json:"message_id"`
	UserID    int64     `gorm:"not null" json:"user_id"`
	IsBot     bool      `gorm:"default:false" json:"is_bot"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time `json:"created_at"`
	Meme      Meme      `gorm:"foreignKey:MemeID" json:"meme"`
}

func (MemeComment) TableName() string {
	return "meme_comments"
}

// MemeHistory модель для истории взаимодействий с мемами
type MemeHistory struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ChatID    int64     `gorm:"not null;index" json:"chat_id"`
	MemeID    uint      `gorm:"not null;index" json:"meme_id"`
	Context   JSONB     `gorm:"type:jsonb" json:"context"`
	UpdatedAt time.Time `json:"updated_at"`
	Meme      Meme      `gorm:"foreignKey:MemeID" json:"meme"`
}

func (MemeHistory) TableName() string {
	return "meme_history"
}

// UserDialog для хранения диалогов с пользователями
type UserDialog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    int64     `gorm:"not null;index" json:"user_id"`
	Role      string    `gorm:"not null;size:20" json:"role"` // system, user, assistant
	Content   string    `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func (UserDialog) TableName() string {
	return "user_dialogs"
}

// MemeInteraction для хранения взаимодействий с мемами
type MemeInteraction struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    int64     `gorm:"not null;index" json:"user_id"`
	MemeID    string    `gorm:"not null;index" json:"meme_id"` // может быть file_id или group_id
	Role      string    `gorm:"not null;size:20" json:"role"`  // user, assistant
	Content   string    `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

func (MemeInteraction) TableName() string {
	return "meme_interactions"
}

// CommentMemeMapping для связи комментариев с мемами
type CommentMemeMapping struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	MessageID int       `gorm:"uniqueIndex;not null" json:"message_id"`
	MemeID    string    `gorm:"not null;index" json:"meme_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (CommentMemeMapping) TableName() string {
	return "comment_meme_mappings"
}
