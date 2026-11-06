package appfolders

// Migration actions, one per file in the report.
const (
	ActionMove        = "move"
	ActionDeduplicate = "deduplicate"
	ActionSkip        = "skip"
)

// MigrationReport is what a migration found and, when Applied, did.
type MigrationReport struct {
	Applied    bool          `json:"applied"`
	Spaces     []SpaceReport `json:"spaces"`
	Moved      int           `json:"moved"`
	Duplicates int           `json:"deduplicated"`
	Skipped    int           `json:"skipped"`
	MovedBytes int64         `json:"moved_bytes"`
	Failed     int           `json:"failed"`
}

// SpaceReport is one app-provisioned space and the fate of each of its files.
type SpaceReport struct {
	SpaceID   int64        `json:"space_id"`
	FacileID  string       `json:"facile_id"`
	Name      string       `json:"name"`
	App       string       `json:"app"`
	ChargedTo int64        `json:"charged_to"`
	Skipped   string       `json:"skipped,omitempty"`
	Files     []FileReport `json:"files"`
}

// FileReport is one file: where it goes, or why it stays.
type FileReport struct {
	FileID int64  `json:"file_id"`
	Name   string `json:"name"`
	Size   int64  `json:"size"`
	Key    string `json:"key,omitempty"`
	Action string `json:"action"`
	Reason string `json:"reason,omitempty"`
	Error  string `json:"error,omitempty"`
}

// add counts one space's files into the totals.
func (r *MigrationReport) add(space SpaceReport) {
	r.Spaces = append(r.Spaces, space)
	for _, file := range space.Files {
		switch {
		case file.Error != "":
			r.Failed++
		case file.Action == ActionMove:
			r.Moved++
			r.MovedBytes += file.Size
		case file.Action == ActionDeduplicate:
			r.Duplicates++
		default:
			r.Skipped++
		}
	}
}
