package sharing

import (
	"time"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
)

func publicFile(file schemas.File) PublicFile {
	return PublicFile{
		ID:        file.ID,
		FacileID:  file.FacileID,
		Name:      file.Name,
		MimeType:  file.MimeType,
		Size:      file.Size,
		UpdatedAt: file.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func publicFileRef(file schemas.File) *PublicFile {
	mapped := publicFile(file)
	return &mapped
}
