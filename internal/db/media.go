package db

import (
	"fmt"

	"github.com/kingbenny101/kbarr/internal/models"
)

func InsertMedia(m models.Media) (int64, error) {
	altTitles, err := StringSliceToJSON(m.AlternateTitles)
	if err != nil {
		return 0, fmt.Errorf("failed to encode alternate titles: %w", err)
	}

	result, err := DB.Exec(`
		INSERT INTO media (title, title_original, alternate_titles, description, status, type, episodes, seasons, year, cover_image, banner_image, external_id, source)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.Title, m.TitleOriginal, altTitles, m.Description, m.Status, m.Type, m.Episodes, m.Seasons, m.Year, m.CoverImage, m.BannerImage, m.ExternalID, m.Source,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert media: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get inserted id: %w", err)
	}

	return id, nil
}

func DeleteMedia(id string) error {
	_, err := DB.Exec("DELETE FROM media WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete media: %w", err)
	}
	return nil
}

func GetAllMedia() ([]models.Media, error) {
	rows, err := DB.Query("SELECT id, title, title_original, alternate_titles, description, status, type, episodes, seasons, year, cover_image, banner_image, external_id, source, added_at, updated_at FROM media")
	if err != nil {
		return nil, fmt.Errorf("failed to query media: %w", err)
	}
	defer rows.Close()

	var mediaList []models.Media

	for rows.Next() {
		var m models.Media
		var altTitles string

		err := rows.Scan(
			&m.ID, &m.Title, &m.TitleOriginal, &altTitles,
			&m.Description, &m.Status, &m.Type, &m.Episodes,
			&m.Seasons, &m.Year, &m.CoverImage, &m.BannerImage,
			&m.ExternalID, &m.Source, &m.AddedAt, &m.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		m.AlternateTitles, err = JSONToStringSlice(altTitles)
		if err != nil {
			return nil, fmt.Errorf("failed to decode alternate titles: %w", err)
		}

		mediaList = append(mediaList, m)
	}

	return mediaList, nil
}
