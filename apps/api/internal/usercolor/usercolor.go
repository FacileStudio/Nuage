package usercolor

import (
	"context"
	"strings"

	"gorm.io/gorm"
)

var palette = []string{
	"AD9EF0",
	"F09ED6",
	"EE7E89",
	"EEB47E",
	"A9EE7E",
	"7EEEDB",
}

var allowed = map[string]struct{}{
	"AD9EF0": {},
	"F09ED6": {},
	"EE7E89": {},
	"EEB47E": {},
	"A9EE7E": {},
	"7EEEDB": {},
}

func AllowedColors() []string {
	return append([]string(nil), palette...)
}

func Normalize(value string) (string, bool) {
	normalized := strings.ToUpper(strings.TrimSpace(value))
	normalized = strings.TrimPrefix(normalized, "#")
	if _, ok := allowed[normalized]; !ok {
		return "", false
	}
	return normalized, true
}

func NextAvailable(ctx context.Context, db *gorm.DB) (string, error) {
	counts, err := loadCounts(db.WithContext(ctx))
	if err != nil {
		return "", err
	}
	return chooseColor(counts), nil
}

func EnsureForUser(ctx context.Context, db *gorm.DB, userID int64) (string, error) {
	type userRecord struct {
		Color string `gorm:"column:color"`
	}

	var selected string
	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record userRecord
		if err := tx.Table("users").Select("color").Where("id = ?", userID).Take(&record).Error; err != nil {
			return err
		}

		if normalized, ok := Normalize(record.Color); ok {
			selected = normalized
			if record.Color == normalized {
				return nil
			}
			return tx.Table("users").Where("id = ?", userID).Update("color", normalized).Error
		}

		counts, err := loadCounts(tx)
		if err != nil {
			return err
		}

		selected = chooseColor(counts)
		return tx.Table("users").Where("id = ?", userID).Update("color", selected).Error
	})
	if err != nil {
		return "", err
	}
	return selected, nil
}

func BackfillMissing(ctx context.Context, db *gorm.DB) error {
	type userRecord struct {
		ID    int64  `gorm:"column:id"`
		Color string `gorm:"column:color"`
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		counts, err := loadCounts(tx)
		if err != nil {
			return err
		}

		var users []userRecord
		if err := tx.Table("users").Select("id, color").Order("id asc").Find(&users).Error; err != nil {
			return err
		}

		for _, user := range users {
			if normalized, ok := Normalize(user.Color); ok {
				if user.Color != normalized {
					if err := tx.Table("users").Where("id = ?", user.ID).Update("color", normalized).Error; err != nil {
						return err
					}
				}
				continue
			}

			color := chooseColor(counts)
			if err := tx.Table("users").Where("id = ?", user.ID).Update("color", color).Error; err != nil {
				return err
			}
			counts[color]++
		}

		return nil
	})
}

func loadCounts(db *gorm.DB) (map[string]int64, error) {
	type colorCount struct {
		Color string `gorm:"column:color"`
		Count int64  `gorm:"column:count"`
	}

	counts := map[string]int64{}
	for _, color := range palette {
		counts[color] = 0
	}

	var rows []colorCount
	if err := db.Table("users").
		Select("upper(color) as color, count(*) as count").
		Where("upper(color) IN ?", palette).
		Group("upper(color)").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	for _, row := range rows {
		counts[row.Color] = row.Count
	}

	return counts, nil
}

func chooseColor(counts map[string]int64) string {
	selected := palette[0]
	lowest := counts[selected]

	for _, color := range palette[1:] {
		if counts[color] < lowest {
			selected = color
			lowest = counts[color]
		}
	}

	return selected
}
