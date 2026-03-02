package models

type Anime struct {
	ID             int      `db:"id"`
	Title          string   `db:"title"`
	TitleJP        string   `db:"title_jp"`
	AlternateTitles []string `db:"alternate_titles"`
	Description    string   `db:"description"`
	Status         string   `db:"status"`
	Episodes       int      `db:"episodes"`
	AnidbID        string   `db:"anidb_id"`
	CoverImage     string   `db:"cover_image"`
	AddedAt        string   `db:"added_at"`
}