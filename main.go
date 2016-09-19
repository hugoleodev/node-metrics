package main

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/structs"
	"github.com/influxdata/influxdb/client/v2"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/net"
	"log"
	"os"
	"time"
)

const (
	database = "metrics"
)

type ResourceMetric struct {

	// Disk Usage
	DiskSize uint64  `json:"diskSize"`
	DiskUsed uint64  `json:"diskUsed"`
	DiskFree uint64  `json:"diskFree"`
	DiskPct  float64 `json:"diskPercent"`

	// Memory Usage
	MemSize uint64 `json:"memSize"`
	MemUsed uint64 `json:"memUsed"`
	MemFree uint64 `json:"memFree"`

	// SwapMemory Usage
	SwapSize uint64 `json:"swapSize"`
	SwapUsed uint64 `json:"swapUsed"`
	SwapFree uint64 `json:"swapFree"`

	// CPU Usage
	CpuUsage float64 `json:"cpuUsage"`

	// Network Usage
	RxBytes uint64 `json:"rxBytes"`
	TxBytes uint64 `json:"txBytes"`
}

func main() {
	c := influxDBClient()
	createDatabase(c)

	for {
		m := collectMetric(c)
		writeMetric(c, m)
		j, _ := json.Marshal(m)
		fmt.Println(string(j))
		time.Sleep(1 * time.Second)
	}

	defer c.Close()
}

func influxDBClient() client.Client {
	c, err := client.NewHTTPClient(client.HTTPConfig{
		Addr:     os.Getenv("INFLUXDB_HOST"),
		Username: os.Getenv("INFLUXDB_USER"),
		Password: os.Getenv("INFLUXDB_PWD"),
	})

	if err != nil {
		log.Fatalln("Error: ", err)
	}

	return c
}

func createDatabase(c client.Client) *client.Response {
	q := client.NewQuery(fmt.Sprintf("CREATE DATABASE %s", database), "", "")

	response, err := c.Query(q)

	if err != nil {
		log.Fatalln("Conection Error: ", err)
	}

	if response.Error() != nil {
		log.Fatalln("Response Error: ", response.Error())
	}

	return response
}

func collectMetric(c client.Client) *ResourceMetric {
	vm, _ := mem.VirtualMemory()
	sm, _ := mem.SwapMemory()

	ct, err := cpu.Times(false)
	fmt.Println(ct)

	if err != nil {
		log.Fatalln("CPU INFO Error: ", err.Error())
	}

	du, err := disk.Usage("/")

	if err != nil {
		log.Fatalln("DISK USAGE Error: ", err.Error())
	}

	n, err := net.IOCounters(false)

	if err != nil {
		log.Fatalln("NET USAGE Error: ", err.Error())
	}

	m := new(ResourceMetric)

	m.DiskSize = du.Total
	m.DiskUsed = du.Used
	m.DiskFree = du.Free
	m.DiskPct = du.UsedPercent
	m.MemSize = vm.Total
	m.MemUsed = vm.Used
	m.MemFree = vm.Free
	m.SwapSize = sm.Total
	m.SwapUsed = sm.Used
	m.SwapFree = sm.Free
	m.CpuUsage = ct[0].System
	m.RxBytes = n[0].BytesRecv
	m.TxBytes = n[0].BytesSent

	return m
}

func writeMetric(c client.Client, m *ResourceMetric) {

	bp, _ := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  database,
		Precision: "s",
	})

	hostname, err := os.Hostname()

	if err != nil {
		log.Fatalln("Hostname Retrieving Error: ", err.Error())
	}

	tags := map[string]string{"hostname": hostname}
	fields := structs.Map(m)

	pt, err := client.NewPoint("node_metrics", tags, fields, time.Now())

	if err != nil {
		fmt.Println("Error: ", err.Error())
	}

	bp.AddPoint(pt)

	c.Write(bp)
}
