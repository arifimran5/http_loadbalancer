## Load Balancer Flow - (OLD)
1. defined load balancer servers list in the config.json
2. define load balancing algorithm -> roundrobin, least_conn, iphash
3. create load balancer instance, add servers list to loadbalancer with health response time threshold
4. start health check using gocron package
5. start http server to listen for incoming requests
6. forward incoming requests to the selected server using the load balancing algorithm
7. check for configs (rate limiting, blacklisting, whitelisting)
8. increment active connection count
9. Forwards the request using the selected server's proxy (httputil.ReverseProxy)
10. handle response from the selected server and send it back to the client
11. decrement active connection count
12. repeat steps 6-11 for each incoming request