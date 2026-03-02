package db

import (
	"fmt"

	"github.com/kingbenny101/kbarr/internal/models"
)

func InsertAnime(a models.Anime) (int64, error) {
	altTitles, err := StringSliceToJSON(a.AlternateTitles)
	if err != nil {
		return 0, fmt.Errorf("failed to encode alternate titles: %w", err)
	}

	result, err := DB.Exec(`
		INSERT INTO anime (title, title_jp, alternate_titles, description, status, episodes, anidb_id, cover_image)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		a.Title, a.TitleJP, altTitles, a.Description, a.Status, a.Episodes, a.AnidbID, a.CoverImage,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to insert anime: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get inserted id: %w", err)
	}

	return id, nil
}

func GetAllAnime() ([]models.Anime, error) {
	rows, err := DB.Query("SELECT id, title, title_jp, alternate_titles, description, status, episodes, anidb_id, cover_image, added_at FROM anime")
	if err != nil {
		return nil, fmt.Errorf("failed to query anime: %w", err)
	}
	defer rows.Close()

	var animeList []models.Anime

	for rows.Next() {
		var a models.Anime
		var altTitles string

		err := rows.Scan(
			&a.ID, &a.Title, &a.TitleJP, &altTitles,
			&a.Description, &a.Status, &a.Episodes,
			&a.AnidbID, &a.CoverImage, &a.AddedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		a.AlternateTitles, err = JSONToStringSlice(altTitles)
		if err != nil {
			return nil, fmt.Errorf("failed to decode alternate titles: %w", err)
		}

		animeList = append(animeList, a)
	}

	return animeList, nil
}
