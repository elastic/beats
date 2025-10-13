package otel

import (
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

func GetHttpCountsFromHistogram(ignoreScope string, rm *metricdata.ResourceMetrics) *metricdata.ResourceMetrics {
	for i, s := range rm.ScopeMetrics {
		if s.Scope.Name != ignoreScope {
			extraMetrics := make([]metricdata.Metrics, 0)
			for _, m := range s.Metrics {
				if m.Name == "http.client.request.duration" {
					sdata := TransformToSumData(m.Data)
					if sdata != nil {
						counted := metricdata.Metrics{
							Name:        m.Name + ".counted",
							Description: m.Description + " counted",
							Unit:        "",
							Data:        sdata,
						}
						extraMetrics = append(extraMetrics, counted)
					}

				}
			}
			if len(extraMetrics) > 0 {
				s.Metrics = append(s.Metrics, extraMetrics...)
				rm.ScopeMetrics[i] = s
			}
		}
	}
	return rm
}

func TransformToSumData(a metricdata.Aggregation) metricdata.Aggregation {
	switch any(a).(type) {
	case metricdata.Histogram[int64]:
		m := a.(metricdata.Histogram[int64])
		dps := HistogramInt64DataPointsToSumDataPoints(m.DataPoints)
		if len(dps) > 0 {
			dps := make([]metricdata.DataPoint[int64], len(m.DataPoints))
			return metricdata.Sum[int64]{
				DataPoints:  dps,
				Temporality: metricdata.DeltaTemporality,
				IsMonotonic: true,
			}
		}
	case metricdata.Histogram[float64]:
		m := a.(metricdata.Histogram[float64])
		dps := HistogramFloat64DataPointsToSumDataPoints(m.DataPoints)
		if len(dps) > 0 {
			return metricdata.Sum[int64]{
				DataPoints:  dps,
				Temporality: metricdata.DeltaTemporality,
				IsMonotonic: true,
			}
		}

	default:
		return nil
	}

	return nil

}

func HistogramInt64DataPointsToSumDataPoints(dps []metricdata.HistogramDataPoint[int64]) []metricdata.DataPoint[int64] {

	newdps := make([]metricdata.DataPoint[int64], 0, len(dps))

	for i, dp := range dps {
		//grab the last exemplar to get the span and trace information
		exemplars := []metricdata.Exemplar[int64]{}
		if len(dp.Exemplars) > 0 {
			ex := dp.Exemplars[len(dp.Exemplars)-1]
			exemplar := metricdata.Exemplar[int64]{Time: ex.Time, Value: int64(dp.Count), SpanID: ex.SpanID, TraceID: ex.TraceID}
			exemplars = append(exemplars, exemplar)
		}
		ndp := metricdata.DataPoint[int64]{
			Attributes: dp.Attributes,
			Value:      int64(dp.Count),
			StartTime:  dp.StartTime,
			Time:       dp.Time,
			Exemplars:  exemplars,
		}
		newdps[i] = ndp
	}
	return newdps
}

func HistogramFloat64DataPointsToSumDataPoints(dps []metricdata.HistogramDataPoint[float64]) []metricdata.DataPoint[int64] {

	newdps := []metricdata.DataPoint[int64]{}

	for _, dp := range dps {
		//grab the last exemplar to get the span and trace information
		exemplars := []metricdata.Exemplar[int64]{}
		if len(dp.Exemplars) > 0 {
			ex := dp.Exemplars[len(dp.Exemplars)-1]
			exemplar := metricdata.Exemplar[int64]{Time: ex.Time, Value: int64(dp.Count), SpanID: ex.SpanID, TraceID: ex.TraceID}
			exemplars = append(exemplars, exemplar)
		}
		ndp := metricdata.DataPoint[int64]{
			Attributes: dp.Attributes,
			Value:      int64(dp.Count),
			StartTime:  dp.StartTime,
			Time:       dp.Time,
			Exemplars:  exemplars,
		}
		newdps = append(newdps, ndp)
	}
	return newdps
}
