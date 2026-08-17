package telemetry

import (
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type MeterProvider struct {
	mu      sync.RWMutex
	meters  map[string]metric.Meter
	meter   metric.Meter
	enabled bool
}

func NewMeterProvider() *MeterProvider {
	return &MeterProvider{
		meters: make(map[string]metric.Meter),
	}
}

func (mp *MeterProvider) SetMeter(m metric.Meter) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.meter = m
	mp.enabled = m != nil
}

func (mp *MeterProvider) Enabled() bool {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	return mp.enabled
}

// Meter returns the registered metric.Meter, or nil if none is configured.
// Callers must check Enabled() before calling Meter() to avoid nil panics.
func (mp *MeterProvider) Meter(name string) metric.Meter {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	if mp.meter != nil {
		return mp.meter
	}
	if m, ok := mp.meters[name]; ok {
		return m
	}
	return nil
}

func (mp *MeterProvider) RegisterMeter(name string, m metric.Meter) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.meters[name] = m
}

func TagsToAttrs(tags map[string]string) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, len(tags))
	for k, v := range tags {
		attrs = append(attrs, attribute.String(k, v))
	}
	return attrs
}
