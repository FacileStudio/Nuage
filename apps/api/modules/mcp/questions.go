package mcp

import (
	"fmt"

	"github.com/FacileStudio/Nuage/apps/api/schemas"
)

// trashQuestion is what the person is asked before trashing.
func trashQuestion(file *schemas.File, folder *schemas.Folder) string {
	if file != nil {
		return fmt.Sprintf("Move the file %q to the trash? It can be restored from the trash in Nuage.", file.Name)
	}
	return fmt.Sprintf("Move the folder %q and everything in it to the trash? It can be restored from the trash in Nuage.", folder.Name)
}

// shareQuestion is what the person is asked before a link is made.
func shareQuestion(file *schemas.File, folder *schemas.Folder, days int) string {
	what := fmt.Sprintf("the file %q", nameOf(file, folder))
	if folder != nil {
		what = fmt.Sprintf("the folder %q and everything in it", folder.Name)
	}
	lasting := "with no expiry"
	if days > 0 {
		lasting = fmt.Sprintf("for %d days", days)
	}
	return fmt.Sprintf("Create a public link to %s? Anyone with the link can open it, %s.", what, lasting)
}

// nameOf is the name of a file or a folder.
func nameOf(file *schemas.File, folder *schemas.Folder) string {
	if file != nil {
		return file.Name
	}
	return folder.Name
}
