package story

import "unicode/utf8"

// PlaybackProgress 根据断点与全文/分段计算播放进度（管理端展示用）。
func PlaybackProgress(rec *StoryRecord) (segmentIndex, segmentTotal, percent int) {
	if rec == nil {
		return 0, 0, 0
	}
	segmentTotal = len(rec.Segments)
	segmentIndex = rec.LastPosition.SegmentIndex
	if rec.LastPlayStatus == PlayStatusCompleted {
		if segmentTotal > 0 {
			return segmentTotal - 1, segmentTotal, 100
		}
		return 0, 0, 100
	}
	if rec.FullText != "" && rec.LastPosition.CharOffset > 0 {
		totalRunes := utf8.RuneCountInString(rec.FullText)
		if totalRunes > 0 {
			percent = rec.LastPosition.CharOffset * 100 / totalRunes
			if percent > 100 {
				percent = 100
			}
		}
		if segmentTotal > 0 {
			segmentIndex = SegmentIndexForCharOffset(rec.Segments, rec.LastPosition.CharOffset)
			if segmentIndex >= segmentTotal {
				segmentIndex = segmentTotal - 1
			}
		}
		return segmentIndex, segmentTotal, percent
	}
	if segmentTotal == 0 {
		return segmentIndex, 0, 0
	}
	if segmentIndex < 0 {
		segmentIndex = 0
	}
	percent = (segmentIndex + 1) * 100 / segmentTotal
	if percent > 100 {
		percent = 100
	}
	return segmentIndex, segmentTotal, percent
}
