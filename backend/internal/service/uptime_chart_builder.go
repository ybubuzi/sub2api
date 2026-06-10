package service

import (
	"sort"
	"time"
)

func buildUptimeResponse(query UptimeChartQuery, rows []UptimeBucketRow) *UptimeChartResponse {
	seriesMap := make(map[string]*UptimeSeries, len(rows))
	starts := buildUptimeBucketStarts(query.StartTime, query.EndTime, query.BucketSeconds)
	for _, row := range rows {
		series := ensureUptimeSeries(seriesMap, row, starts)
		applyUptimeRow(series, row)
	}
	series := orderedUptimeSeries(seriesMap)
	return &UptimeChartResponse{
		Scope: query.Scope, Window: query.Window, Dimension: query.Dimension,
		StartTime: query.StartTime, EndTime: query.EndTime,
		BucketSeconds: query.BucketSeconds, SuccessSource: "usage_logs",
		FailureSource: "ops_error_logs", SLAExcludesBusiness: true,
		Series: series, Summary: summarizeUptimeSeries(series),
	}
}

func buildUptimeBucketStarts(start, end time.Time, seconds int) []time.Time {
	aligned := alignUptimeBucketStart(start.UTC(), seconds)
	capacity := int(end.Sub(start).Seconds()/float64(seconds)) + 2
	out := make([]time.Time, 0, capacity)
	for t := aligned; t.Before(end); t = t.Add(time.Duration(seconds) * time.Second) {
		out = append(out, t)
	}
	return out
}

func alignUptimeBucketStart(t time.Time, seconds int) time.Time {
	if seconds <= 0 {
		seconds = 60
	}
	epoch := t.Unix()
	return time.Unix((epoch/int64(seconds))*int64(seconds), 0).UTC()
}

func ensureUptimeSeries(rows map[string]*UptimeSeries, row UptimeBucketRow, starts []time.Time) *UptimeSeries {
	if existing := rows[row.Key]; existing != nil {
		return existing
	}
	series := &UptimeSeries{Key: row.Key, Label: row.Label, Buckets: make([]UptimeBucket, len(starts))}
	for i, start := range starts {
		series.Buckets[i] = UptimeBucket{Start: start}
	}
	rows[row.Key] = series
	return series
}

func applyUptimeRow(series *UptimeSeries, row UptimeBucketRow) {
	for i := range series.Buckets {
		if !series.Buckets[i].Start.Equal(row.Bucket.UTC()) {
			continue
		}
		updateUptimeBucket(&series.Buckets[i], row)
		updateUptimeSummary(&series.Summary, row)
		return
	}
}

func updateUptimeBucket(bucket *UptimeBucket, row UptimeBucketRow) {
	bucket.Success += row.Success
	bucket.Failure += row.Failure
	bucket.Total = bucket.Success + bucket.Failure
	bucket.Availability = uptimeAvailability(bucket.Success, bucket.Total)
}

func updateUptimeSummary(summary *UptimeSeriesStats, row UptimeBucketRow) {
	summary.Success += row.Success
	summary.Failure += row.Failure
	summary.Total = summary.Success + summary.Failure
	summary.Availability = uptimeAvailability(summary.Success, summary.Total)
}

func orderedUptimeSeries(seriesMap map[string]*UptimeSeries) []UptimeSeries {
	series := make([]UptimeSeries, 0, len(seriesMap))
	for _, item := range seriesMap {
		series = append(series, *item)
	}
	sort.Slice(series, func(i, j int) bool {
		return compareUptimeSeries(series[i], series[j])
	})
	return series
}

func compareUptimeSeries(left, right UptimeSeries) bool {
	if left.Summary.Total != right.Summary.Total {
		return left.Summary.Total > right.Summary.Total
	}
	if left.Label != right.Label {
		return left.Label < right.Label
	}
	return left.Key < right.Key
}

func summarizeUptimeSeries(series []UptimeSeries) UptimeSeriesStats {
	var out UptimeSeriesStats
	for _, item := range series {
		out.Success += item.Summary.Success
		out.Failure += item.Summary.Failure
	}
	out.Total = out.Success + out.Failure
	out.Availability = uptimeAvailability(out.Success, out.Total)
	return out
}

func uptimeAvailability(success, total int64) *float64 {
	if total <= 0 {
		return nil
	}
	value := float64(success) / float64(total)
	return &value
}
