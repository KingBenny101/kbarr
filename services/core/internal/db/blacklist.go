package db

import "context"

func IsBlacklisted(ctx context.Context, torrentHash, torrentName string) bool {
	if err := ensureDB(); err != nil {
		return false
	}
	q := DB.NewSelect().Model((*TorrentBlacklist)(nil))
	if torrentHash != "" {
		q = q.WhereOr("torrent_hash = ?", torrentHash)
	}
	if torrentName != "" {
		q = q.WhereOr("torrent_name = ?", torrentName)
	}
	exists, _ := q.Exists(ctx)
	return exists
}

func AddToBlacklist(ctx context.Context, hash, name, title string) error {
	if err := ensureDB(); err != nil {
		return err
	}
	entry := TorrentBlacklist{
		TorrentHash: ptrStr(hash),
		TorrentName: ptrStr(name),
		Title:       ptrStr(title),
	}
	_, err := DB.NewInsert().Model(&entry).Exec(ctx)
	return err
}

func ClearBlacklist(ctx context.Context) error {
	if err := ensureDB(); err != nil {
		return err
	}
	_, err := DB.NewDelete().Model((*TorrentBlacklist)(nil)).Where("1=1").Exec(ctx)
	return err
}

func ptrStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
