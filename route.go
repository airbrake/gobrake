package gobrake

import (
	"context"
)

type routes struct {
	filters []routeFilter

	stats      *routeStats
	breakdowns *routeBreakdowns
}

func newRoutes(opt *NotifierOptions) *routes {
	return &routes{
		stats:      newRouteStats(opt),
		breakdowns: newRouteBreakdowns(opt),
	}
}

// AddFilter adds filter that can change route stat or ignore it by returning nil.
func (rs *routes) AddFilter(fn func(*RouteMetric) *RouteMetric) {
	rs.filters = append(rs.filters, fn)
}

func (rs *routes) Flush() {
	rs.stats.Flush()
	rs.breakdowns.Flush()
}

func (rs *routes) Notify(c context.Context, metric *RouteMetric) error {
	metric.finish()

	for _, fn := range rs.filters {
		metric = fn(metric)
		if metric == nil {
			return nil
		}
	}

	err := rs.stats.Notify(c, metric)
	if err != nil {
		return err
	}

	err = rs.breakdowns.Notify(c, metric)
	if err != nil {
		return err
	}

	return nil
}
