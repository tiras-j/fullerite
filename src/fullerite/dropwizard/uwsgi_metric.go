package dropwizard

import (
	"encoding/json"
	"fullerite/metric"
)

// UWSGIMetric parser for UWSGI metrics
type UWSGIMetric struct {
	BaseParser
	Format int `json:"format"`
}

type newUWSGIFormat struct {
	ServiceDims map[string]interface{} `json:"service_dims"`
	Counters    []map[string]interface{}
	Gauges      []map[string]interface{}
	Histograms  []map[string]interface{}
	Meters      []map[string]interface{}
	Timers      []map[string]interface{}
}

// NewUWSGIMetric creates new parser for uwsgi metrics
func NewUWSGIMetric(data []byte, schemaVer string, ccEnabled bool) *UWSGIMetric {
	parser := new(UWSGIMetric)
	parser.data = data
	parser.schemaVer = schemaVer
	parser.ccEnabled = ccEnabled
	parser.Format = 1
	// Overwrite the data format version if it exists in the payload
	json.Unmarshal(data, parser)
	return parser
}

func (parser *UWSGIMetric) parseArrOfMap(metricArray []map[string]interface{}, metricType string) []metric.Metric {
	results := []metric.Metric{}

	for _, metricData := range metricArray {
		if name, ok := metricData["name"]; ok {
			delete(metricData, "name")
			tempResults := parser.metricFromMap(metricData, name.(string), metricType)
			results = append(results, tempResults...)
		}
	}
	return results
}

func (parser *UWSGIMetric) parseMapOfMap(metricMap map[string]map[string]interface{}, metricType string) []metric.Metric {
	results := []metric.Metric{}

	for metricName, metricData := range metricMap {
		tempResults := parser.metricFromMap(metricData, metricName, metricType)
		results = append(results, tempResults...)
	}
	return results
}

// Parse method parses metrics and returns
func (parser *UWSGIMetric) Parse() ([]metric.Metric, error) {
	var results []metric.Metric

	switch parser.Format {
	case 2:
		parsed := new(newUWSGIFormat)
		// Sane defaults for ServiceDims to avoid conditional later
		parsed.ServiceDims = map[string]interface{}{}
		err := json.Unmarshal(parser.data, parsed)
		if err != nil {
			return []metric.Metric{}, err
		}
		results = extractNewUWSGIParsedMetric(parser, parsed)
		// Unfortunately we have to do this in both locations due to the type difference
		// between `parsed` variable in the branches
		for k, v := range parsed.ServiceDims {
			metric.AddToAll(&results, map[string]string{k: v.(string)})
		}
	default:
		parsed := new(Format)
		parsed.ServiceDims = map[string]interface{}{}
		err := json.Unmarshal(parser.data, parsed)
		if err != nil {
			return []metric.Metric{}, err
		}
		results = extractParsedMetric(parser, parsed)
		for k, v := range parsed.ServiceDims {
			metric.AddToAll(&results, map[string]string{k: v.(string)})
		}
	}

	return results, nil
}

func extractNewUWSGIParsedMetric(parser *UWSGIMetric, parsed *newUWSGIFormat) []metric.Metric {
	results := []metric.Metric{}
	appendIt := func(metrics []metric.Metric, typeDimVal string) {
		if !parser.isCCEnabled() {
			metric.AddToAll(&metrics, map[string]string{"type": typeDimVal})
		}
		results = append(results, metrics...)
	}

	appendIt(parser.parseArrOfMap(parsed.Gauges, metric.Gauge), "gauge")
	appendIt(parser.parseArrOfMap(parsed.Counters, metric.Counter), "counter")
	appendIt(parser.parseArrOfMap(parsed.Histograms, metric.Gauge), "histogram")
	appendIt(parser.parseArrOfMap(parsed.Meters, metric.Gauge), "meter")
	appendIt(parser.parseArrOfMap(parsed.Timers, metric.Gauge), "timer")

	return results
}
