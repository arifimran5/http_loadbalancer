## Switchblade

Switchblade is a simple load balancer written in Go, designed to efficiently distribute incoming requests across multiple backend servers while ensuring high availability and performance.

## **Key Features**

### **Load Balancer**

- **Round-Robin Scheduling**: The load balancer uses a round-robin scheduling algorithm to distribute incoming requests across multiple servers.
- **Least Connections Algorithm**: In addition to round-robin, it supports a least connections algorithm that directs requests to the server with the fewest active connections.
- **Server Health Checks**: The load balancer periodically checks the health of each server by sending HEAD requests. If a server is unhealthy, it is skipped until it becomes healthy again.
- **Automatic Server Failover**: If a server becomes unhealthy, the load balancer automatically redirects requests to the next available server.

## **Usage**

### **Running the Load Balancer**

1. Run the load balancer by executing the **`cmd/loadbalancer/main.go`** file.
2. The load balancer will start listening on port 8000.
3. You can add servers to the load balancer by calling the **`AddServer`** method.

### **Running the Test Server**

1. Change the function name to **`main`** in **`test-server.go`** file. Run the test server by executing it.
2. Pass the port number as a command-line argument, for example, **`go run test-server.go 8080`**.
3. The test server will start listening on the specified port.

### **Example Usage**

1. Start four test servers on ports 8080, 8081, 8082, and 8083.
2. Run the load balancer and add the test servers to it.
3. Send requests to the load balancer at **`http://localhost:8000/test`**.
4. The load balancer will distribute the requests across the test servers based on the selected algorithm.

### **Command-Line Flags**

You can specify which load balancing algorithm to use when running the load balancer:

- To use **Round Robin** (Default):
  `go run cmd/loadbalancer/main.go --algorithm roundrobin`
- To use **Least Connections**:
  `go run cmd/loadbalancer/main.go --algorithm leastconnections`

## **Dependencies**

- **`net/http/httputil`**: For creating a reverse proxy.
- **`github.com/go-co-op/gocron`**: For scheduling server health checks.

## **License**

This project is licensed under the MIT License.
