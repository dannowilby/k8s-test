package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func main() {
	http.HandleFunc("/", handler)
	http.HandleFunc("/name", name_handler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func name_handler(w http.ResponseWriter, r *http.Request) {
	podName := os.Getenv("POD_NAME")
	if podName == "" {
		podName = "unknown"
	}
	response := fmt.Sprintf("<p>%s</p>", podName)

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, response)
}

func handler(w http.ResponseWriter, r *http.Request) {

	fmt.Println("HANDLER")

	// get env
	podName := os.Getenv("POD_NAME")
	ips := os.Getenv("IPS")
	replicas := os.Getenv("REPLICAS")
	port := os.Getenv("PORT")

	i, _ := strconv.Atoi(replicas)

	output := ""

	t := strings.LastIndex(podName, "-")

	for x := 0; x < i; x++ {

		url := fmt.Sprintf("http://%s-%d.%s:%s/name", podName[:t], x, "go-pod-headless-service", port)
		resp, err0 := http.Get(url)
		if err0 != nil {
			fmt.Println("e0", err0)
			return
		}
		r, err1 := io.ReadAll(resp.Body)
		if err1 != nil {
			fmt.Println("e1", err1)
			return
		}
		output += string(r)
	}

	response := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Pod Info Server</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .container { max-width: 600px; margin: 0 auto; text-align: center; }
        .pod-name { color: #2196F3; font-size: 24px; font-weight: bold; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Hello from Kubernetes!</h1>
        <p>This request is being served by pod:</p>
        <div class="pod-name">%s</div>
        <p><small>Path: %s</small></p>
		<p>IPs: %s</p>
		<div>%s</div>
    </div>
</body>
</html>
    `, podName, r.URL.Path, ips, output)

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, response)
}
