package docs

import (
	"net/http"

	"github.com/FacileStudio/tronc/apiref"
	"github.com/go-chi/chi/v5"
)

type (
	Registry = apiref.Registry
	Module   = apiref.Module
	Route    = apiref.Route
	Field    = apiref.Field
	Error    = apiref.Error
)

type RegisterRequest struct {
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required"`
	Name     string `json:"name" validate:"required"`
}

type AuthResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required"`
	Password string `json:"password" validate:"required"`
}

type UserResponse struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url,omitempty"`
	IsAdmin   bool   `json:"is_admin"`
	CreatedAt string `json:"created_at"`
}

type UserListResponse struct {
	Users []UserResponse `json:"users"`
}

type UpdateUserRequest struct {
	Name string `json:"name,omitempty"`
}

type AvatarResponse struct {
	AvatarURL string `json:"avatar_url"`
}

type APITokenResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Token     string `json:"token,omitempty"`
	CreatedAt string `json:"created_at"`
}

type CreateAPITokenRequest struct {
	Name string `json:"name" validate:"required"`
}

type FileResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	MimeType    string `json:"mime_type"`
	FolderID    string `json:"folder_id,omitempty"`
	SpaceID     string `json:"space_id,omitempty"`
	Version     int    `json:"version"`
	DownloadURL string `json:"download_url,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type FileListResponse struct {
	Files []FileResponse `json:"files"`
}

type UpdateFileRequest struct {
	Name     string `json:"name,omitempty"`
	FolderID string `json:"folder_id,omitempty"`
}

type LinkFileRequest struct {
	URL  string `json:"url" validate:"required"`
	Name string `json:"name" validate:"required"`
}

type PresignRequest struct {
	ExpiresIn int64 `json:"expires_in,omitempty"`
}

type PresignResponse struct {
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at"`
}

type FileVersionResponse struct {
	ID        string `json:"id"`
	FileID    string `json:"file_id"`
	Version   int    `json:"version"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
}

type VersionListResponse struct {
	Versions []FileVersionResponse `json:"versions"`
}

type InitChunkedUploadRequest struct {
	Name     string `json:"name" validate:"required"`
	Size     int64  `json:"size" validate:"required"`
	MimeType string `json:"mime_type,omitempty"`
	FolderID string `json:"folder_id,omitempty"`
	SpaceID  string `json:"space_id,omitempty"`
}

type InitChunkedUploadResponse struct {
	SessionID string `json:"session_id"`
	ChunkSize int64  `json:"chunk_size"`
}

type UploadPartResponse struct {
	PartNumber int    `json:"part_number"`
	ETag       string `json:"etag"`
}

type UploadStatusResponse struct {
	SessionID      string `json:"session_id"`
	UploadedParts  []int  `json:"uploaded_parts"`
	CompletedBytes int64  `json:"completed_bytes"`
	TotalBytes     int64  `json:"total_bytes"`
}

type FolderResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ParentID  string `json:"parent_id,omitempty"`
	SpaceID   string `json:"space_id,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type FolderListResponse struct {
	Folders []FolderResponse `json:"folders"`
}

type CreateFolderRequest struct {
	Name     string `json:"name" validate:"required"`
	ParentID string `json:"parent_id,omitempty"`
	SpaceID  string `json:"space_id,omitempty"`
}

type UpdateFolderRequest struct {
	Name     string `json:"name,omitempty"`
	ParentID string `json:"parent_id,omitempty"`
}

type ShareResponse struct {
	ID         int64         `json:"id"`
	Token      string        `json:"token"`
	FileID     *int64        `json:"file_id,omitempty"`
	FolderID   *int64        `json:"folder_id,omitempty"`
	SharedBy   int64         `json:"shared_by"`
	Permission string        `json:"permission"`
	ExpiresAt  *string       `json:"expires_at,omitempty"`
	CreatedAt  string        `json:"created_at"`
	File       *PublicFile   `json:"file,omitempty"`
	Folder     *PublicFolder `json:"folder,omitempty"`
}

type CreateShareRequest struct {
	FileID     *int64  `json:"file_id,omitempty"`
	FolderID   *int64  `json:"folder_id,omitempty"`
	Permission string  `json:"permission,omitempty"`
	ExpiresAt  *string `json:"expires_at,omitempty"`
	SpaceID    *int64  `json:"space_id,omitempty"`
}

type ShareListResponse struct {
	Shares []ShareResponse `json:"shares"`
}

type PublicFile struct {
	ID       int64  `json:"id"`
	FacileID string `json:"facile_id"`
	Name     string `json:"name"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
}

type PublicFolder struct {
	ID       int64  `json:"id"`
	FacileID string `json:"facile_id"`
	Name     string `json:"name"`
	Size     int64  `json:"size"`
}

type PublicShareResponse struct {
	Token      string        `json:"token"`
	Permission string        `json:"permission"`
	File       *PublicFile   `json:"file,omitempty"`
	Folder     *PublicFolder `json:"folder,omitempty"`
}

type SharedFolderContentsResponse struct {
	Permission string         `json:"permission"`
	Files      []PublicFile   `json:"files"`
	Folders    []PublicFolder `json:"folders"`
}

type SpaceResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	IsDefault bool   `json:"is_default"`
	Role      string `json:"role,omitempty"`
	CreatedAt string `json:"created_at"`
}

type CreateSpaceRequest struct {
	Name string `json:"name" validate:"required"`
}

type UpdateSpaceRequest struct {
	Name string `json:"name" validate:"required"`
}

type SpaceMemberResponse struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	CreatedAt string `json:"created_at"`
}

type AddSpaceMemberRequest struct {
	UserID string `json:"user_id" validate:"required"`
	Role   string `json:"role" validate:"required"`
}

type UpdateSpaceMemberRequest struct {
	Role string `json:"role" validate:"required"`
}

type TrashItemResponse struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Name         string `json:"name"`
	Size         int64  `json:"size,omitempty"`
	DeletedAt    string `json:"deleted_at"`
	OriginalPath string `json:"original_path,omitempty"`
}

type TrashListResponse struct {
	Items []TrashItemResponse `json:"items"`
}

type SyncChange struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Action    string `json:"action"`
	Timestamp string `json:"timestamp"`
}

type SyncChangesResponse struct {
	Changes    []SyncChange `json:"changes"`
	NextCursor string       `json:"next_cursor"`
}

type SyncStateResponse struct {
	Files   []FileResponse   `json:"files"`
	Folders []FolderResponse `json:"folders"`
	Cursor  string           `json:"cursor"`
}

type ActivityItemResponse struct {
	ID         string `json:"id"`
	Action     string `json:"action"`
	ActorID    string `json:"actor_id"`
	ActorName  string `json:"actor_name"`
	TargetID   string `json:"target_id"`
	TargetName string `json:"target_name"`
	CreatedAt  string `json:"created_at"`
}

type ActivityListResponse struct {
	Activities []ActivityItemResponse `json:"activities"`
	NextCursor string                 `json:"next_cursor,omitempty"`
}

type QuotaResponse struct {
	UsedBytes  int64 `json:"used_bytes"`
	TotalBytes int64 `json:"total_bytes"`
	FileCount  int64 `json:"file_count"`
}

type AdminUsageResponse struct {
	Users []QuotaResponse `json:"users"`
}

type SetQuotaRequest struct {
	StorageLimit int64 `json:"storage_limit" validate:"required"`
}

type SearchResponse struct {
	Files   []FileResponse   `json:"files"`
	Folders []FolderResponse `json:"folders"`
}

type SettingsResponse struct {
	RegistrationOpen bool   `json:"registration_open"`
	MaxUploadSizeMB  int64  `json:"max_upload_size_mb"`
	NookURL          string `json:"nook_url,omitempty"`
}

type UpdateSettingsRequest struct {
	RegistrationOpen *bool   `json:"registration_open,omitempty"`
	MaxUploadSizeMB  *int64  `json:"max_upload_size_mb,omitempty"`
	NookURL          *string `json:"nook_url,omitempty"`
}

type TestNookResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
}

type NookDeliveryResponse struct {
	ID        string `json:"id"`
	Event     string `json:"event"`
	Status    string `json:"status"`
	Attempts  int    `json:"attempts"`
	CreatedAt string `json:"created_at"`
}

// Mount mounts the reference page and the OpenAPI document on router.
func Mount(router chi.Router) {
	apiref.Mount(router, Reference())
}

// Reference returns Nuage's API reference configuration.
func Reference() apiref.Config {
	return apiref.Config{
		Title:       "Nuage API",
		Description: "Self-hosted cloud storage for the Facile Suite.",
		Servers:     []string{"/api"},
		Registry: Registry{
			Modules: []Module{
				{
					Name:        "auth",
					Description: "Local and OIDC authentication",
					Routes: []Route{
						{
							Method:       "POST",
							Path:         "/auth/register",
							Summary:      "Register a new user account",
							RequestBody:  RegisterRequest{},
							ResponseBody: AuthResponse{},
							Status:       http.StatusCreated,
						},
						{
							Method:       "POST",
							Path:         "/auth/login",
							Summary:      "Log in with email and password",
							RequestBody:  LoginRequest{},
							ResponseBody: AuthResponse{},
						},
					},
				},
				{
					Name:        "users",
					Description: "User accounts, profile management, and API tokens",
					Routes: []Route{
						{
							Method:       "GET",
							Path:         "/users",
							Summary:      "List users in instance",
							Auth:         "bearer",
							ResponseBody: UserListResponse{},
						},
						{
							Method:       "GET",
							Path:         "/users/me",
							Summary:      "Get current user profile",
							Auth:         "bearer",
							ResponseBody: UserResponse{},
						},
						{
							Method:       "PATCH",
							Path:         "/users/me",
							Summary:      "Update user profile details",
							Auth:         "bearer",
							RequestBody:  UpdateUserRequest{},
							ResponseBody: UserResponse{},
						},
						{
							Method:       "POST",
							Path:         "/users/me/avatar",
							Summary:      "Upload user avatar image",
							Auth:         "bearer",
							ResponseBody: AvatarResponse{},
						},
						{
							Method:       "DELETE",
							Path:         "/users/me/avatar",
							Summary:      "Delete user avatar image",
							Auth:         "bearer",
							ResponseBody: AvatarResponse{},
						},
						{
							Method:       "GET",
							Path:         "/users/me/api-token",
							Summary:      "List user API tokens",
							Auth:         "bearer",
							ResponseBody: []APITokenResponse{},
						},
						{
							Method:       "POST",
							Path:         "/users/me/api-token",
							Summary:      "Create a new personal API token",
							Auth:         "bearer",
							RequestBody:  CreateAPITokenRequest{},
							ResponseBody: APITokenResponse{},
							Status:       http.StatusCreated,
						},
						{
							Method:       "DELETE",
							Path:         "/users/me/api-token/{id}",
							Summary:      "Revoke personal API token",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "id", Type: "string", Description: "Token ID"}},
							ResponseBody: map[string]bool{"deleted": true},
						},
						{
							Method:       "GET",
							Path:         "/users/{id}",
							Summary:      "Get user profile by ID",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "id", Type: "string", Description: "User ID"}},
							ResponseBody: UserResponse{},
						},
					},
				},
				{
					Name:        "files",
					Description: "File uploads, downloads, versioning, and chunked sessions",
					Routes: []Route{
						{
							Method:       "POST",
							Path:         "/files",
							Summary:      "Upload a single file (multipart)",
							Auth:         "bearer",
							ResponseBody: FileResponse{},
							Status:       http.StatusCreated,
						},
						{
							Method:       "GET",
							Path:         "/files",
							Summary:      "List files in folder or space",
							Auth:         "bearer",
							QueryParams:  []Field{{Name: "folder_id", Type: "string", Description: "Folder ID"}, {Name: "space_id", Type: "string", Description: "Space ID"}},
							ResponseBody: FileListResponse{},
						},
						{
							Method:       "GET",
							Path:         "/files/{id}",
							Summary:      "Get file metadata",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "id", Type: "string", Description: "File ID"}},
							ResponseBody: FileResponse{},
						},
						{
							Method:     "GET",
							Path:       "/files/{id}/download",
							Summary:    "Download file bytes",
							Auth:       "bearer",
							PathParams: []Field{{Name: "id", Type: "string", Description: "File ID"}},
						},
						{
							Method:     "DELETE",
							Path:       "/files/{id}",
							Summary:    "Soft delete a file to trash",
							Auth:       "bearer",
							PathParams: []Field{{Name: "id", Type: "string", Description: "File ID"}},
							Status:     http.StatusNoContent,
						},
						{
							Method:       "PUT",
							Path:         "/files/{id}",
							Summary:      "Rename or move a file",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "id", Type: "string", Description: "File ID"}},
							RequestBody:  UpdateFileRequest{},
							ResponseBody: FileResponse{},
						},
						{
							Method:       "POST",
							Path:         "/files/{id}/link",
							Summary:      "Link an external URL as a file reference",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "id", Type: "string", Description: "File ID"}},
							RequestBody:  LinkFileRequest{},
							ResponseBody: FileResponse{},
						},
						{
							Method:       "POST",
							Path:         "/files/{id}/presign",
							Summary:      "Generate temporary presigned download link",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "id", Type: "string", Description: "File ID"}},
							RequestBody:  PresignRequest{},
							ResponseBody: PresignResponse{},
						},
						{
							Method:       "POST",
							Path:         "/files/{id}/reupload",
							Summary:      "Upload a new version of an existing file",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "id", Type: "string", Description: "File ID"}},
							ResponseBody: FileResponse{},
						},
						{
							Method:       "GET",
							Path:         "/files/{id}/versions",
							Summary:      "List historical versions of a file",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "id", Type: "string", Description: "File ID"}},
							ResponseBody: VersionListResponse{},
						},
						{
							Method:       "POST",
							Path:         "/files/{id}/versions/{versionId}/restore",
							Summary:      "Restore a past version of a file",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "id", Type: "string", Description: "File ID"}, {Name: "versionId", Type: "string", Description: "Version ID"}},
							ResponseBody: FileResponse{},
						},
						{
							Method:       "POST",
							Path:         "/files/upload/init",
							Summary:      "Initialize chunked multipart upload session",
							Auth:         "bearer",
							RequestBody:  InitChunkedUploadRequest{},
							ResponseBody: InitChunkedUploadResponse{},
							Status:       http.StatusCreated,
						},
						{
							Method:       "PUT",
							Path:         "/files/upload/{sessionId}/part/{partNumber}",
							Summary:      "Upload single part chunk",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "sessionId", Type: "string", Description: "Upload Session ID"}, {Name: "partNumber", Type: "int", Description: "Part index (1-based)"}},
							ResponseBody: UploadPartResponse{},
						},
						{
							Method:       "POST",
							Path:         "/files/upload/{sessionId}/complete",
							Summary:      "Finalize chunked upload session",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "sessionId", Type: "string", Description: "Upload Session ID"}},
							ResponseBody: FileResponse{},
						},
						{
							Method:       "GET",
							Path:         "/files/upload/{sessionId}/status",
							Summary:      "Check progress of chunked upload session",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "sessionId", Type: "string", Description: "Upload Session ID"}},
							ResponseBody: UploadStatusResponse{},
						},
						{
							Method:     "DELETE",
							Path:       "/files/upload/{sessionId}",
							Summary:    "Abort chunked upload session",
							Auth:       "bearer",
							PathParams: []Field{{Name: "sessionId", Type: "string", Description: "Upload Session ID"}},
							Status:     http.StatusNoContent,
						},
						{
							Method:     "GET",
							Path:       "/presigned/{token}",
							Summary:    "Public download via presigned token",
							PathParams: []Field{{Name: "token", Type: "string", Description: "Presigned signature token"}},
						},
					},
				},
				{
					Name:        "folders",
					Description: "Directory structure and folder hierarchy",
					Routes: []Route{
						{
							Method:       "POST",
							Path:         "/folders",
							Summary:      "Create a new folder",
							Auth:         "bearer",
							RequestBody:  CreateFolderRequest{},
							ResponseBody: FolderResponse{},
							Status:       http.StatusCreated,
						},
						{
							Method:       "GET",
							Path:         "/folders",
							Summary:      "List folders in parent or space",
							Auth:         "bearer",
							QueryParams:  []Field{{Name: "parent_id", Type: "string", Description: "Parent Folder ID"}, {Name: "space_id", Type: "string", Description: "Space ID"}},
							ResponseBody: FolderListResponse{},
						},
						{
							Method:       "GET",
							Path:         "/folders/{id}",
							Summary:      "Get folder metadata",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "id", Type: "string", Description: "Folder ID"}},
							ResponseBody: FolderResponse{},
						},
						{
							Method:       "PUT",
							Path:         "/folders/{id}",
							Summary:      "Rename or move folder",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "id", Type: "string", Description: "Folder ID"}},
							RequestBody:  UpdateFolderRequest{},
							ResponseBody: FolderResponse{},
						},
						{
							Method:     "DELETE",
							Path:       "/folders/{id}",
							Summary:    "Delete folder and soft-delete contents",
							Auth:       "bearer",
							PathParams: []Field{{Name: "id", Type: "string", Description: "Folder ID"}},
							Status:     http.StatusNoContent,
						},
					},
				},
				{
					Name:        "shares",
					Description: "Public and authenticated share links",
					Routes: []Route{
						{
							Method:       "POST",
							Path:         "/shares",
							Summary:      "Create a share link for file or folder",
							Auth:         "bearer",
							RequestBody:  CreateShareRequest{},
							ResponseBody: ShareResponse{},
							Status:       http.StatusCreated,
						},
						{
							Method:       "GET",
							Path:         "/shares/by-me",
							Summary:      "List share links created by current user",
							Auth:         "bearer",
							QueryParams:  []Field{{Name: "space_id", Type: "string", Description: "Space ID filter"}},
							ResponseBody: ShareListResponse{},
						},
						{
							Method:     "DELETE",
							Path:       "/shares/{id}",
							Summary:    "Revoke share link",
							Auth:       "bearer",
							PathParams: []Field{{Name: "id", Type: "string", Description: "Share ID"}},
							Status:     http.StatusNoContent,
						},
						{
							Method:       "GET",
							Path:         "/shared/{token}",
							Summary:      "Inspect public share metadata",
							PathParams:   []Field{{Name: "token", Type: "string", Description: "Share Token"}},
							ResponseBody: PublicShareResponse{},
						},
						{
							Method:     "GET",
							Path:       "/shared/{token}/download/{fileId}",
							Summary:    "Download shared file from tokenized link",
							PathParams: []Field{{Name: "token", Type: "string", Description: "Share Token"}, {Name: "fileId", Type: "string", Description: "File ID"}},
						},
						{
							Method:       "GET",
							Path:         "/shared/{token}/files",
							Summary:      "List contents of shared folder",
							PathParams:   []Field{{Name: "token", Type: "string", Description: "Share Token"}},
							QueryParams:  []Field{{Name: "folder_id", Type: "string", Description: "Subfolder ID"}},
							ResponseBody: SharedFolderContentsResponse{},
						},
					},
				},
				{
					Name:        "spaces",
					Description: "Collaborative workspaces and space memberships",
					Routes: []Route{
						{
							Method:       "POST",
							Path:         "/spaces",
							Summary:      "Create a new space",
							Auth:         "bearer",
							RequestBody:  CreateSpaceRequest{},
							ResponseBody: SpaceResponse{},
							Status:       http.StatusCreated,
						},
						{
							Method:       "GET",
							Path:         "/spaces",
							Summary:      "List spaces accessible to user",
							Auth:         "bearer",
							ResponseBody: []SpaceResponse{},
						},
						{
							Method:       "GET",
							Path:         "/spaces/{id}",
							Summary:      "Get space details",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "id", Type: "string", Description: "Space ID"}},
							ResponseBody: SpaceResponse{},
						},
						{
							Method:       "PUT",
							Path:         "/spaces/{id}",
							Summary:      "Update space name",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "id", Type: "string", Description: "Space ID"}},
							RequestBody:  UpdateSpaceRequest{},
							ResponseBody: SpaceResponse{},
						},
						{
							Method:     "DELETE",
							Path:       "/spaces/{id}",
							Summary:    "Delete space and all contents",
							Auth:       "bearer",
							PathParams: []Field{{Name: "id", Type: "string", Description: "Space ID"}},
							Status:     http.StatusNoContent,
						},
						{
							Method:       "GET",
							Path:         "/spaces/{id}/members",
							Summary:      "List members of a space",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "id", Type: "string", Description: "Space ID"}},
							ResponseBody: []SpaceMemberResponse{},
						},
						{
							Method:       "POST",
							Path:         "/spaces/{id}/members",
							Summary:      "Add a member to space",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "id", Type: "string", Description: "Space ID"}},
							RequestBody:  AddSpaceMemberRequest{},
							ResponseBody: SpaceMemberResponse{},
							Status:       http.StatusCreated,
						},
						{
							Method:       "PUT",
							Path:         "/spaces/{id}/members/{memberId}",
							Summary:      "Update space member role",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "id", Type: "string", Description: "Space ID"}, {Name: "memberId", Type: "string", Description: "Member ID"}},
							RequestBody:  UpdateSpaceMemberRequest{},
							ResponseBody: SpaceMemberResponse{},
						},
						{
							Method:     "DELETE",
							Path:       "/spaces/{id}/members/{memberId}",
							Summary:    "Remove member from space",
							Auth:       "bearer",
							PathParams: []Field{{Name: "id", Type: "string", Description: "Space ID"}, {Name: "memberId", Type: "string", Description: "Member ID"}},
							Status:     http.StatusNoContent,
						},
						{
							Method:     "POST",
							Path:       "/spaces/{id}/leave",
							Summary:    "Leave a space",
							Auth:       "bearer",
							PathParams: []Field{{Name: "id", Type: "string", Description: "Space ID"}},
							Status:     http.StatusNoContent,
						},
					},
				},
				{
					Name:        "trash",
					Description: "Soft-deleted files, folders, and trash management",
					Routes: []Route{
						{
							Method:       "GET",
							Path:         "/trash",
							Summary:      "List soft-deleted items in trash",
							Auth:         "bearer",
							QueryParams:  []Field{{Name: "space_id", Type: "string", Description: "Space ID"}},
							ResponseBody: TrashListResponse{},
						},
						{
							Method:      "DELETE",
							Path:        "/trash",
							Summary:     "Permanently empty trash",
							Auth:        "bearer",
							QueryParams: []Field{{Name: "space_id", Type: "string", Description: "Space ID"}},
							Status:      http.StatusNoContent,
						},
						{
							Method:     "POST",
							Path:       "/trash/{type}/{id}/restore",
							Summary:    "Restore a deleted file or folder",
							Auth:       "bearer",
							PathParams: []Field{{Name: "type", Type: "string", Description: "file or folder"}, {Name: "id", Type: "string", Description: "Item ID"}},
							Status:     http.StatusNoContent,
						},
						{
							Method:     "DELETE",
							Path:       "/trash/{type}/{id}",
							Summary:    "Permanently delete a single item",
							Auth:       "bearer",
							PathParams: []Field{{Name: "type", Type: "string", Description: "file or folder"}, {Name: "id", Type: "string", Description: "Item ID"}},
							Status:     http.StatusNoContent,
						},
					},
				},
				{
					Name:        "sync",
					Description: "Delta sync feed and full state snapshots for desktop/CLI sync",
					Routes: []Route{
						{
							Method:       "GET",
							Path:         "/sync/changes",
							Summary:      "Get incremental changes since cursor",
							Auth:         "bearer",
							QueryParams:  []Field{{Name: "since_cursor", Type: "string", Description: "Last known sync cursor"}},
							ResponseBody: SyncChangesResponse{},
						},
						{
							Method:       "GET",
							Path:         "/sync/state",
							Summary:      "Fetch full workspace file and folder snapshot",
							Auth:         "bearer",
							ResponseBody: SyncStateResponse{},
						},
					},
				},
				{
					Name:        "activity",
					Description: "Audit logging and user activity feed",
					Routes: []Route{
						{
							Method:       "GET",
							Path:         "/activity",
							Summary:      "List all activity entries (Admin only)",
							Auth:         "bearer",
							QueryParams:  []Field{{Name: "limit", Type: "int", Description: "Max entries"}, {Name: "cursor", Type: "string", Description: "Pagination cursor"}},
							ResponseBody: ActivityListResponse{},
						},
						{
							Method:       "GET",
							Path:         "/activity/me",
							Summary:      "Get recent activity for current user",
							Auth:         "bearer",
							QueryParams:  []Field{{Name: "limit", Type: "int", Description: "Max entries"}, {Name: "cursor", Type: "string", Description: "Pagination cursor"}},
							ResponseBody: ActivityListResponse{},
						},
						{
							Method:       "GET",
							Path:         "/activity/files/{id}",
							Summary:      "Get audit activity log for specific file",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "id", Type: "string", Description: "File ID"}},
							QueryParams:  []Field{{Name: "limit", Type: "int", Description: "Max entries"}, {Name: "cursor", Type: "string", Description: "Pagination cursor"}},
							ResponseBody: ActivityListResponse{},
						},
					},
				},
				{
					Name:        "quota",
					Description: "Storage consumption metrics and limits",
					Routes: []Route{
						{
							Method:       "GET",
							Path:         "/quota/me",
							Summary:      "Get storage usage and quota allocation",
							Auth:         "bearer",
							ResponseBody: QuotaResponse{},
						},
						{
							Method:       "POST",
							Path:         "/quota/me/recalculate",
							Summary:      "Force recalculation of storage usage",
							Auth:         "bearer",
							ResponseBody: QuotaResponse{},
						},
						{
							Method:       "GET",
							Path:         "/quota/users",
							Summary:      "List storage usage for all users (Admin only)",
							Auth:         "bearer",
							ResponseBody: AdminUsageResponse{},
						},
						{
							Method:       "PUT",
							Path:         "/quota/users/{userId}",
							Summary:      "Set storage limit for user (Admin only)",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "userId", Type: "string", Description: "User ID"}},
							RequestBody:  SetQuotaRequest{},
							ResponseBody: QuotaResponse{},
						},
					},
				},
				{
					Name:        "search",
					Description: "Full-text file and folder query",
					Routes: []Route{
						{
							Method:       "GET",
							Path:         "/search",
							Summary:      "Search files and folders by query string",
							Auth:         "bearer",
							QueryParams:  []Field{{Name: "q", Type: "string", Description: "Search query"}, {Name: "space_id", Type: "string", Description: "Space ID filter"}},
							ResponseBody: SearchResponse{},
						},
					},
				},
				{
					Name:        "settings",
					Description: "Instance configuration and webhook delivery status",
					Routes: []Route{
						{
							Method:       "GET",
							Path:         "/settings",
							Summary:      "Get instance configuration settings",
							Auth:         "bearer",
							ResponseBody: SettingsResponse{},
						},
						{
							Method:       "PUT",
							Path:         "/settings",
							Summary:      "Update instance settings (Admin only)",
							Auth:         "bearer",
							RequestBody:  UpdateSettingsRequest{},
							ResponseBody: SettingsResponse{},
						},
						{
							Method:       "POST",
							Path:         "/settings/test-nook",
							Summary:      "Send test ping to Antenne / Nook alert bus",
							Auth:         "bearer",
							ResponseBody: TestNookResponse{},
						},
						{
							Method:       "GET",
							Path:         "/settings/nook/deliveries",
							Summary:      "List recent alert bus webhook delivery attempts",
							Auth:         "bearer",
							ResponseBody: []NookDeliveryResponse{},
						},
					},
				},
			},
		},
	}
}
