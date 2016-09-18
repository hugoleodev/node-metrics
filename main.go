package main

import (
	"fmt"
	"github.com/influxdata/influxdb/client/v2"
	"github.com/shirou/gopsutil/mem"
	"log"
	"os"
	"time"
)

const (
	database = "metrics"
)

func main() {
	c := influxDBClient()
	createDatabase(c)

	for {
		writeMetric(c)
		fmt.Println("OK!")
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

func writeMetric(c client.Client) {
	vm, _ := mem.VirtualMemory()
	sm, _ := mem.SwapMemory()

	bp, _ := client.NewBatchPoints(client.BatchPointsConfig{
		Database:  database,
		Precision: "s",
	})

	hostname, err := os.Hostname()

	if err != nil {
		log.Fatalln("Hostname Retrieving Error: ", err.Error())
	}

	tags := map[string]string{"hostname": hostname}
	fields := map[string]interface{}{
		"memsize":  vm.Total,
		"memused":  vm.Used,
		"memfree":  vm.Free,
		"swapsize": sm.Total,
		"swapused": sm.Used,
		"swapfree": sm.Free,
	}

	pt, err := client.NewPoint("node_metrics", tags, fields, time.Now())

	if err != nil {
		fmt.Println("Error: ", err.Error())
	}

	bp.AddPoint(pt)

	c.Write(bp)
}
