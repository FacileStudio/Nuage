package schemas

import "time"

// RegisteredApp is a suite app the server knows by its Registre OIDC client
// id. Rows are seeded from NUAGE_APPS at boot and never from a request or a
// token label. Active is false once the app leaves the configuration: its
// folders stay readable and deletable by their users, but it can no longer
// act for anyone.
type RegisteredApp struct {
	ID        int64     `json:"-" gorm:"primaryKey"`
	Key       string    `json:"key" gorm:"uniqueIndex;not null"`
	Name      string    `json:"name" gorm:"not null"`
	ClientID  string    `json:"-" gorm:"uniqueIndex;not null"`
	Active    bool      `json:"active" gorm:"not null;default:true"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

func (RegisteredApp) TableName() string { return "registered_apps" }

// AppFolder is one registered app's private storage for one user. The server
// creates it the first time the app stores something for that user. It is
// never a space and never appears in the file tree; its bytes count toward
// the user's personal plan.
type AppFolder struct {
	ID        int64         `json:"-" gorm:"primaryKey"`
	UserID    int64         `json:"-" gorm:"not null;uniqueIndex:idx_app_folder_user_app"`
	AppID     int64         `json:"-" gorm:"not null;uniqueIndex:idx_app_folder_user_app;index"`
	CreatedAt time.Time     `json:"created_at"`
	User      User          `json:"-" gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	App       RegisteredApp `json:"-" gorm:"foreignKey:AppID;constraint:OnDelete:RESTRICT"`
}

func (AppFolder) TableName() string { return "app_folders" }

// AppObject is one object an app stored in an app folder, addressed by the
// key the app chose. Its bytes live in the shared bucket under BucketKey.
// SourceFileID is the file a migration moved it from, unique so a rerun never
// moves the same file twice.
type AppObject struct {
	ID           int64     `json:"-" gorm:"primaryKey"`
	FolderID     int64     `json:"-" gorm:"not null;uniqueIndex:idx_app_object_folder_key"`
	Key          string    `json:"key" gorm:"not null;uniqueIndex:idx_app_object_folder_key"`
	Name         string    `json:"name" gorm:"not null"`
	MimeType     string    `json:"mime_type"`
	Size         int64     `json:"size"`
	Hash         string    `json:"hash"`
	BucketKey    string    `json:"-" gorm:"not null"`
	SourceFileID *int64    `json:"-" gorm:"uniqueIndex"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	Folder       AppFolder `json:"-" gorm:"foreignKey:FolderID;constraint:OnDelete:CASCADE"`
}

func (AppObject) TableName() string { return "app_objects" }
