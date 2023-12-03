package queue

import (
	"time"

	"cncamp/pkg/third_party/nightingale/alert/astats"
	"github.com/toolkits/pkg/container/list"
)

var EventQueue = list.NewSafeListLimited(10000000)

func ReportQueueSize(stats *astats.Stats) {
	for {
		time.Sleep(time.Second)

		stats.GaugeAlertQueueSize.Set(float64(EventQueue.Len()))
	}
}
