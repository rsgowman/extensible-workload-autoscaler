package internal

import (
	"fmt"
	"log/slog"
	"strings"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/common/expfmt"
	"github.com/prometheus/common/model"
	corev1 "k8s.io/api/core/v1"

	pb "github.com/gke-labs/extensible-workload-autoscaler/api/server/v1alpha"
	xasv1 "github.com/gke-labs/extensible-workload-autoscaler/api/xas/v1"
)

func (a *CoreNodeMetricsProvider) scrapePod(ip string, port string, path string, whitelist []string) ([]*pb.MetricSample, error) {
	if port == "" {
		port = "8080"
	}
	if path == "" {
		path = "/metrics"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	url := fmt.Sprintf("http://%s:%s%s", ip, port, path)

	resp, err := a.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	parser := expfmt.NewTextParser(model.LegacyValidation)
	families, err := parser.TextToMetricFamilies(resp.Body)
	if err != nil {
		return nil, err
	}

	var samples []*pb.MetricSample
	for name, fam := range families {
		samples = append(samples, extractFamilySamples(name, fam, whitelist)...)
	}
	return samples, nil
}

// extractFamilySamples converts matching metrics in a Prometheus MetricFamily
// into MetricSamples according to whitelist.
func extractFamilySamples(name string, fam *dto.MetricFamily, whitelist []string) []*pb.MetricSample {
	var wantBase, wantSum, wantCount bool
	for _, w := range whitelist {
		switch w {
		case name, name + "_bucket":
			wantBase = true
		case name + "_sum":
			wantSum = true
		case name + "_count":
			wantCount = true
		}
	}
	if !wantBase && !wantSum && !wantCount {
		return nil
	}

	var samples []*pb.MetricSample
	for _, m := range fam.Metric {
		labels := make(map[string]string, len(m.Label))
		for _, pair := range m.Label {
			labels[pair.GetName()] = pair.GetValue()
		}

		switch fam.GetType() {
		case dto.MetricType_GAUGE:
			if m.Gauge != nil && wantBase {
				samples = append(samples, pb.MetricSample_builder{
					Name:   new(name),
					Labels: labels,
					Gauge:  new(m.Gauge.GetValue()),
				}.Build())
			}
		case dto.MetricType_COUNTER:
			if m.Counter != nil && wantBase {
				samples = append(samples, pb.MetricSample_builder{
					Name:    new(name),
					Labels:  labels,
					Counter: new(m.Counter.GetValue()),
				}.Build())
			}
		case dto.MetricType_HISTOGRAM:
			if m.Histogram != nil {
				samples = append(samples, extractHistogramSamples(name, labels, m.Histogram, wantBase, wantSum, wantCount)...)
			}
		}
	}
	return samples
}

// extractHistogramSamples builds the requested histogram, _sum, and _count
// samples for a single Prometheus histogram metric observation.
func extractHistogramSamples(name string, labels map[string]string, hist *dto.Histogram, wantBase, wantSum, wantCount bool) []*pb.MetricSample {
	var samples []*pb.MetricSample
	if wantBase {
		buckets := make(map[string]uint64, len(hist.Bucket))
		for _, b := range hist.Bucket {
			le := fmt.Sprintf("%g", b.GetUpperBound())
			buckets[le] = b.GetCumulativeCount()
		}

		samples = append(samples, pb.MetricSample_builder{
			Name:   new(name),
			Labels: labels,
			Histogram: pb.Histogram_builder{
				Buckets: buckets,
				Sum:     new(hist.GetSampleSum()),
				Count:   new(hist.GetSampleCount()),
			}.Build(),
		}.Build())
	}

	// The sum and the count of the observations of a histogram are cumulative,
	// like counters: they only increase (the sum, as long as no observation is
	// negative), except when the source resets them.
	if wantSum {
		samples = append(samples, pb.MetricSample_builder{
			Name:    new(name + "_sum"),
			Labels:  labels,
			Counter: new(hist.GetSampleSum()),
		}.Build())
	}

	if wantCount {
		samples = append(samples, pb.MetricSample_builder{
			Name:    new(name + "_count"),
			Labels:  labels,
			Counter: new(float64(hist.GetSampleCount())),
		}.Build())
	}
	return samples
}

func (a *CoreNodeMetricsProvider) processPrometheusMetric(pod corev1.Pod, m *pb.MetricDefinition, class *xasv1.MetricProviderClass) []*pb.MetricBatch {
	getParam := func(key string) string {
		if val, ok := m.GetParams()[key]; ok {
			return val
		}
		if class != nil {
			if val, ok := class.Spec.Config[key]; ok {
				return val
			}
		}
		return ""
	}

	metricName := getParam("metric")
	if metricName == "" {
		return nil
	}
	port := getParam("port")
	path := getParam("path")

	ip := pod.Status.PodIP
	samples, err := a.scrapePod(ip, port, path, []string{metricName})
	if err != nil {
		slog.Error("Failed to scrape prometheus metric from pod", "pod", pod.Name, "ip", ip, "port", port, "path", path, "metric", metricName, "error", err)
		return nil
	}

	for i := range samples {
		samples[i].SetName(m.GetName())
		samples[i].SetRecommenderName(m.GetRecommenderName())
	}

	if len(samples) > 0 {
		return []*pb.MetricBatch{pb.MetricBatch_builder{
			PodName: new(pod.Name),
			Samples: samples,
		}.Build()}
	}
	return nil
}
