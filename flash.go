package main

import (
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/johneliades/flash/routes"
)

func getOutboundInterface() (*net.Interface, error) {
	conn, err := net.Dial("udp", "1.1.1.1:80") // Doesn't actually send traffic
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	for _, iface := range interfaces {
		addrs, _ := iface.Addrs()
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if ok && ipNet.IP.Equal(localAddr.IP) {
				return &iface, nil
			}
		}
	}

	return nil, fmt.Errorf("interface not found")
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && 
		strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func main() {
	iface, err := getOutboundInterface()
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	fmt.Println("Using interface:", iface.Name)

	if !containsIgnoreCase(iface.Name, "nordlynx") {
		fmt.Println("❌ Not using NordLynx. Aborting.")
		os.Exit(1)
	}

	fmt.Println("✅ NordLynx confirmed. Proceeding.")

	r := gin.Default()
	r.Use(cors.Default())

	routes.RegisterRoutes(r)

	r.Run(":8080")
}