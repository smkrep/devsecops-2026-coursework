package metrics

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry *prometheus.Registry
	requests *prometheus.CounterVec
	errors   *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inFlight *prometheus.GaugeVec
}

func New(methods ...string) *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "gateway_requests_total",
			Help: "Количество завершённых запросов шлюза по методу и коду gRPC.",
		}, []string{
			"method",
			"code",
		}),
		errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "gateway_request_errors_total",
			Help: "Количество запросов шлюза, завершившихся ошибкой.",
		}, []string{
			"method",
		}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "gateway_request_duration_seconds",
			Help:    "Длительность обработки запроса шлюзом в секундах.",
			Buckets: prometheus.DefBuckets,
		}, []string{
			"method",
		}),
		inFlight: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "gateway_requests_in_flight",
			Help: "Количество запросов, обрабатываемых шлюзом в данный момент.",
		}, []string{
			"method",
		}),
	}
	m.registry.MustRegister(m.requests, m.errors, m.duration, m.inFlight)
	// Показываем метрики известных методов даже до первого запроса.
	for _, method := range methods {
		m.requests.WithLabelValues(method, "OK").Add(0)
		m.errors.WithLabelValues(method).Add(0)
		m.duration.WithLabelValues(method)
		m.inFlight.WithLabelValues(method).Set(0)
	}
	return m
}

// StartRequest учитывает активный запрос и возвращает функцию завершения.
// Функция завершения вызывается перехватчиком ровно один раз с кодом gRPC.
func (m *Metrics) StartRequest(method string) func(string) {
	started := time.Now()
	inFlight := m.inFlight.WithLabelValues(method)
	inFlight.Inc()
	return func(code string) {
		defer inFlight.Dec()
		m.requests.WithLabelValues(method, code).Inc()
		if code != "OK" {
			m.errors.WithLabelValues(method).Inc()
		}
		m.duration.WithLabelValues(method).Observe(time.Since(started).Seconds())
	}
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}
