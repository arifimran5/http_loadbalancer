package balancer

import (
	"log"
	"time"

	"github.com/go-co-op/gocron"
)

func StartHealthCheck(lb *LoadBalancer) {
	sch := gocron.NewScheduler(time.Local)
	for _, host := range lb.servers {
		_, err := sch.Every(2).Seconds().Do(func(s *Server) {
			healthy := s.CheckHealth()
			if !healthy {
				log.Printf("'%s' is not healthy", s.Proxy.Host)
			}
		}, host)
		if err != nil {
			log.Fatalln(err)
		}
	}
	sch.StartAsync()
}
