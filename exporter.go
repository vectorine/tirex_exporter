package main

import (
	"encoding/json"
	"log"
	"os/exec"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	queueSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "tirex_queue_size",
		Help: "Current tirex render queue size",
	})
	prioQueueSizes = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "tirex_prio_queue_size",
		Help: "Current priority queue sizes",
	}, []string{"prio"})
	currentlyRendering = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "tirex_rendering",
		Help: "Number of currently rendering workers",
	})
)

type pQueue struct {
	Size int
	Prio int
}

type tirexOutput struct {
	Queue struct {
		Size       int
		PrioQueues []pQueue
	}
	RM struct {
		NumRendering int `json:"num_rendering"`
		Stats        struct {
			CountError     int `json:"count_error"`
			CountTimeouted int `json:"count_timeouted"`
			CountRequested int `json:"count_requested"`
			CountExpired   int `json:"count_expired"`
		}
	}
}

func parseTirexOutput(buf []byte) (tirexOutput, error) {
	to := tirexOutput{}
	err := json.Unmarshal(buf, &to)
	return to, err
}

func crawl() ([]byte, error) {
	out, err := exec.Command("tirex-status", "-r").Output()
	if err != nil {
		log.Printf("crawling tirex-status failed: %v", err)
		return nil, err
	}
	return out, nil
}

func crawlAndSet() {
	buf, err := crawl()
	if err != nil {
		return
	}
	to, err := parseTirexOutput(buf)

	queueSize.Set(float64(to.Queue.Size))
	for _, queue := range to.Queue.PrioQueues {
		prioQueueSizes.With(prometheus.Labels{"prio": strconv.Itoa(queue.Prio)}).Set(float64(queue.Size))
	}

	currentlyRendering.Set(float64(to.RM.NumRendering))
}

func main() {
	log.Println("tirex_exporter started")
	for {
		crawlAndSet()
		time.Sleep(10 * time.Second)
	}
}
