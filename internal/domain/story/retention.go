package story

import "time"

func RetentionDays(record StoryRecord, cfg Config) int {
	days := cfg.MinRetentionDays + record.PlayCount*2 + record.CompleteCount*3
	if days < cfg.MinRetentionDays {
		days = cfg.MinRetentionDays
	}
	if days > cfg.MaxRetentionDays {
		days = cfg.MaxRetentionDays
	}
	return days
}

func ShouldEvict(record StoryRecord, now time.Time, cfg Config) bool {
	if record.SeriesID != "" && !record.SeriesComplete {
		return false
	}
	retention := RetentionDays(record, cfg)
	expireAt := record.CreatedAt.Add(time.Duration(retention) * 24 * time.Hour)
	return now.After(expireAt)
}
