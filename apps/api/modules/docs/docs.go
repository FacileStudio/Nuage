package docs

import (
	"net/http"

	"github.com/FacileStudio/Nuage/apps/api/modules/mcp"
	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/tronc/apiref"
	"github.com/FacileStudio/tronc/bulle"
	"github.com/go-chi/chi/v5"
)

type (
	Registry = apiref.Registry
	Module   = apiref.Module
	Route    = apiref.Route
	Field    = apiref.Field
	Error    = apiref.Error
)

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
	Name           string   `json:"name" validate:"required"`
	App            string   `json:"app,omitempty"`
	SpaceFacileIDs []string `json:"space_facile_ids,omitempty"`
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
	LinkedTo string `json:"linked_to,omitempty"`
}

type InitChunkedUploadResponse struct {
	SessionID    string `json:"session_id,omitempty"`
	ChunkSize    int64  `json:"chunk_size,omitempty"`
	ExistingFile any    `json:"existing_file,omitempty"`
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
	UserID string `json:"user_id"`
	Email  string `json:"email"`
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

type TransferBillingRequest struct {
	UserID int64 `json:"user_id" validate:"required"`
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
// invitationsRoute documents porte/kit's GET /me/invitations with its real
// response schema.
func invitationsRoute() Route {
	route := kit.InvitationsRoute()
	route.ResponseBody = kit.InvitationsResponse{}
	return route
}

func Mount(router chi.Router) {
	apiref.Mount(router, Reference())
}

// Reference returns Nuage's API reference configuration.
type AppObjectResponse struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	MimeType  string `json:"mime_type"`
	Size      int64  `json:"size"`
	Hash      string `json:"hash"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type AppObjectListResponse struct {
	Objects []AppObjectResponse `json:"objects"`
	Next    string              `json:"next,omitempty"`
}

type AppDataEntry struct {
	App       string `json:"app"`
	Name      string `json:"name"`
	Active    bool   `json:"active"`
	Objects   int64  `json:"objects"`
	UsedBytes int64  `json:"used_bytes"`
	CreatedAt string `json:"created_at"`
}

type IdeaDecisionRequest struct {
	Approve bool `json:"approve"`
}

type AppDataResponse struct {
	Apps []AppDataEntry `json:"apps"`
}

func Reference() apiref.Config {
	return apiref.Config{
		Title:       "Nuage API",
		Description: "Self-hosted cloud storage for the Facile Suite.",
		Servers:     []string{"/api"},
		Registry: Registry{
			Modules: []Module{
				{
					Name:        "users",
					Description: "User accounts, profile management, and API tokens",
					Routes: []Route{
						{
							Method:       "GET",
							Path:         "/users",
							Summary:      "List the caller and the people it shares a space with (every account for an administrator)",
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
							Summary:      "Create a personal API token. With app plus space_facile_ids the token is restricted: it reaches only the listed spaces through /files, never personal files or other endpoints, and its uploads record app as origin_app. provision is retired and answers 400.",
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
							Summary:      "Get the profile of the caller or a space peer by ID; anyone else answers 404",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "id", Type: "string", Description: "User ID"}},
							ResponseBody: UserResponse{},
						},
						invitationsRoute(),
					},
				},
				{
					Name:        "files",
					Description: "File uploads, downloads, versioning, and chunked sessions",
					Routes: []Route{
						{
							Method:       "POST",
							Path:         "/files",
							Summary:      "Upload a single file (multipart). A restricted token may send linked_to (at most 200 printable characters) to publish idempotently: if its app already holds a live file with that linked_to in the space, that file is returned with 200 and nothing is stored; a new file answers 201. Chunked uploads accept the same linked_to on init.",
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
							Method:      "GET",
							Path:        "/files/{id}/thumbnail",
							Summary:     "Get a JPEG thumbnail of an image",
							Auth:        "bearer",
							PathParams:  []Field{{Name: "id", Type: "string", Description: "File ID"}},
							QueryParams: []Field{{Name: "w", Type: "string", Description: "Width in pixels, 320 or 640"}},
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
							Summary:      "Initialize chunked multipart upload session. A restricted token may send linked_to: if its app already holds a live file with that key in the space, the response is 200 with existing_file and no session, so a retry never re-uploads. Completing a keyed session returns 200 with the existing file when a concurrent upload won, else 201.",
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
							Method:      "GET",
							Path:        "/shared/{token}/thumbnail/{fileId}",
							Summary:     "Get a thumbnail of a shared image",
							PathParams:  []Field{{Name: "token", Type: "string", Description: "Share Token"}, {Name: "fileId", Type: "string", Description: "File ID"}},
							QueryParams: []Field{{Name: "w", Type: "string", Description: "Width in pixels, 320 or 640"}},
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
							Summary:      "Create a new space. Answers 409 with the Registre spaces page when OIDC is configured: shared spaces are created in Registre and mirrored here.",
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
							Summary:      "Update a space's name and description. A space linked to Registre (registre_space_id set) keeps Registre's name: renaming it answers 409.",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "id", Type: "string", Description: "Space ID"}},
							RequestBody:  UpdateSpaceRequest{},
							ResponseBody: SpaceResponse{},
						},
						{
							Method:     "DELETE",
							Path:       "/spaces/{id}",
							Summary:    "Delete space and all contents. Answers 409 for a space linked to Registre.",
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
							Summary:      "Add a member to space by exact email, or by user_id for someone the caller already shares a space with (anyone else answers 404). Answers 409 for a space linked to Registre, which alone manages its members.",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "id", Type: "string", Description: "Space ID"}},
							RequestBody:  AddSpaceMemberRequest{},
							ResponseBody: SpaceMemberResponse{},
							Status:       http.StatusCreated,
						},
						{
							Method:       "PUT",
							Path:         "/spaces/{id}/members/{memberId}",
							Summary:      "Update space member role. Answers 409 for a space linked to Registre.",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "id", Type: "string", Description: "Space ID"}, {Name: "memberId", Type: "string", Description: "Member ID"}},
							RequestBody:  UpdateSpaceMemberRequest{},
							ResponseBody: SpaceMemberResponse{},
						},
						{
							Method:     "DELETE",
							Path:       "/spaces/{id}/members/{memberId}",
							Summary:    "Remove member from space. Answers 409 for a space linked to Registre.",
							Auth:       "bearer",
							PathParams: []Field{{Name: "id", Type: "string", Description: "Space ID"}, {Name: "memberId", Type: "string", Description: "Member ID"}},
							Status:     http.StatusNoContent,
						},
						{
							Method:     "POST",
							Path:       "/spaces/{id}/leave",
							Summary:    "Leave a space. Answers 409 for a space linked to Registre: leave it there.",
							Auth:       "bearer",
							PathParams: []Field{{Name: "id", Type: "string", Description: "Space ID"}},
							Status:     http.StatusNoContent,
						},
						{
							Method:      "POST",
							Path:        "/spaces/{id}/billing",
							Summary:     "Offer who pays for an app-provisioned space's storage to another owner, open for 7 days. Only the current payer may call it, and nothing moves until that owner accepts. The payer cannot leave or be removed while other members remain.",
							Auth:        "bearer",
							PathParams:  []Field{{Name: "id", Type: "string", Description: "Space ID"}},
							RequestBody: TransferBillingRequest{},
							Status:      http.StatusNoContent,
						},
						{
							Method:     "DELETE",
							Path:       "/spaces/{id}/billing",
							Summary:    "Withdraw the payer's pending billing offer.",
							Auth:       "bearer",
							PathParams: []Field{{Name: "id", Type: "string", Description: "Space ID"}},
							Status:     http.StatusNoContent,
						},
						{
							Method:     "POST",
							Path:       "/spaces/{id}/billing/accept",
							Summary:    "Accept an open billing offer made to you: you become the payer of the space's storage.",
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
					Name:        "apps",
					Description: "App folders: each app registered in NUAGE_APPS stores private data per user, counted on that user's personal plan and never shown as a space or in the file tree. The app calls /app with a delegated JWT Registre minted for it by token exchange (sub = the user, aud = Nuage, act.sub = the app's client id); it reaches only that user's folder for its own app. Cookie sessions, API tokens (restricted or not) and JWTs of other actors are refused there with 403, and a registered app's JWT is refused everywhere else. A delegated JWT whose actor is neither an active registered app nor listed in NUAGE_FULL_ACCESS_ACTORS is refused on every route, and WebDAV refuses every delegated JWT.",
					Routes: []Route{
						{
							Method:       "GET",
							Path:         "/app/objects",
							Summary:      "List the acting app's objects for the user, in key order. Requires nuage.read.",
							Auth:         "bearer",
							QueryParams:  []Field{{Name: "prefix", Type: "string", Description: "Only keys starting with this"}, {Name: "after", Type: "string", Description: "Start after this key (the previous page's next)"}, {Name: "limit", Type: "integer", Description: "1 to 1000, default 100"}},
							ResponseBody: AppObjectListResponse{},
						},
						{
							Method:      "GET",
							Path:        "/app/object",
							Summary:     "Download one object's bytes (Range supported). 404 when the key holds nothing. Requires nuage.read.",
							Auth:        "bearer",
							QueryParams: []Field{{Name: "key", Type: "string", Description: "Object key"}},
						},
						{
							Method:       "PUT",
							Path:         "/app/object",
							Summary:      "Store the raw request body under key, replacing what the key held (201 new, 200 replaced). At most 100 MiB; Content-Length must match the body. The folder is created on first use. Only the growth is charged to the user's personal plan (402 when it does not fit). Requires nuage.write.",
							Auth:         "bearer",
							QueryParams:  []Field{{Name: "key", Type: "string", Description: "1 to 512 bytes of UTF-8; no control characters, backslashes, or empty, . or .. segments"}, {Name: "name", Type: "string", Description: "Download file name; defaults to the key's last segment"}},
							ResponseBody: AppObjectResponse{},
						},
						{
							Method:      "DELETE",
							Path:        "/app/object",
							Summary:     "Delete one object and give its bytes back to the user's plan. Requires nuage.write.",
							Auth:        "bearer",
							QueryParams: []Field{{Name: "key", Type: "string", Description: "Object key"}},
							Status:      http.StatusNoContent,
						},
						{
							Method:       "GET",
							Path:         "/app-data",
							Summary:      "The caller's app folders: per app, object count and bytes used. Content is never listed here.",
							Auth:         "bearer",
							ResponseBody: AppDataResponse{},
						},
						{
							Method:     "DELETE",
							Path:       "/app-data/{app}",
							Summary:    "Delete everything an app stores for the caller. A sign-in session or an unrestricted personal API token may do it (porte does not tell them apart); delegated JWTs and API tokens restricted to an app are refused.",
							Auth:       "bearer",
							PathParams: []Field{{Name: "app", Type: "string", Description: "App key"}},
							Status:     http.StatusNoContent,
						},
					},
				},
				{
					Name:        "ideas",
					Description: "Proactive AI suggestions generated by Bulle for Nuage, run in place. Nuage calls Bulle as the signed-in user through a Registre token exchange. Mounted only when OIDC and BULLE_URL are set; answers 404 otherwise. The optional space_id must be a space the caller belongs to and that Registre knows. Ideas are paid (ai.access): a plan without it gets ai_access false on the list and 402 with an upgrade_url on the other routes.",
					Routes: []Route{
						{
							Method:       "GET",
							Path:         "/ideas",
							Summary:      "The open ideas of the active space, whether Bulle is generating, and ai_access.",
							Auth:         "bearer",
							QueryParams:  []Field{{Name: "space_id", Type: "integer", Description: "Space to read ideas for; omit for personal"}},
							ResponseBody: bulle.IdeasPage{},
						},
						{
							Method:       "POST",
							Path:         "/ideas/generate",
							Summary:      "Ask Bulle for a new batch of ideas. 202 when started, 429 past Bulle's daily cap, 409 while a batch runs.",
							Auth:         "bearer",
							QueryParams:  []Field{{Name: "space_id", Type: "integer", Description: "Space to generate for; omit for personal"}},
							RequestBody:  struct{}{},
							ResponseBody: bulle.Batch{},
							Status:       http.StatusAccepted,
						},
						{
							Method:       "POST",
							Path:         "/ideas/{id}/run",
							Summary:      "Run an idea as a background Bulle run. Poll the run for its state.",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "id", Type: "string", Description: "Idea id"}},
							QueryParams:  []Field{{Name: "space_id", Type: "integer", Description: "Space the idea belongs to; omit for personal"}},
							RequestBody:  struct{}{},
							ResponseBody: bulle.RunStart{},
							Status:       http.StatusAccepted,
						},
						{
							Method:      "POST",
							Path:        "/ideas/{id}/dismiss",
							Summary:     "Hide an idea.",
							Auth:        "bearer",
							PathParams:  []Field{{Name: "id", Type: "string", Description: "Idea id"}},
							RequestBody: struct{}{},
							Status:      http.StatusNoContent,
						},
						{
							Method:       "GET",
							Path:         "/ideas/runs/{run}",
							Summary:      "A run's state (running, needs_approval, done, failed), summary, output links and pending approvals.",
							Auth:         "bearer",
							PathParams:   []Field{{Name: "run", Type: "string", Description: "Run id"}},
							ResponseBody: bulle.Run{},
						},
						{
							Method:      "POST",
							Path:        "/ideas/runs/{run}/approvals/{call}",
							Summary:     "Approve or decline a call the run is holding. Approving needs ai.access.",
							Auth:        "bearer",
							PathParams:  []Field{{Name: "run", Type: "string", Description: "Run id"}, {Name: "call", Type: "string", Description: "Call id"}},
							RequestBody: IdeaDecisionRequest{},
							Status:      http.StatusNoContent,
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
							Summary:      "Set storage limit for user (Admin only). Once Registre plans apply it replaces only that person's plan ceiling, for their personal files and the app-provisioned spaces charged to them; it does not apply to uploads into Registre-linked spaces, which follow the space's plan.",
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
				mcp.Documentation,
			},
		},
	}
}
