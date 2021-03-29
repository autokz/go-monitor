package metric_for_services

import (
	"context"
	"log"
	"net"
	"runtime"
	"runtime/metrics"
	"time"

	"github.com/google/uuid"
	pb "github.com/autokz/go-monitor"
	"google.golang.org/grpc"
)

type config struct {
	addr, port, dur, name, uuid string
}

func getIpAddress (domain string) string {
    ips, _ := net.LookupIP(domain)
    for _, ip := range ips {
        if ipv4 := ip.To4(); ipv4 != nil {
            return ipv4.String()
        }
    }
    return ""
}

func getMemory() uint64 {
	const myMetric = "/memory/classes/heap/objects:bytes"

	sample := make([]metrics.Sample, 1)
	sample[0].Name = myMetric

	metrics.Read(sample)

	if sample[0].Value.Kind() == metrics.KindBad {
		log.Print("Error")
	}

	freeBytes := sample[0].Value.Uint64()
	return freeBytes
}

func createGrpcConnect(address, port string) *grpc.ClientConn {
	ip := getIpAddress(address)
	conn, err := grpc.Dial(ip + ":" + port, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}

	return conn
}

func startMetrics(cfg *config) {
	conn := createGrpcConnect(cfg.addr, cfg.port)
	client := pb.NewSendMetricClient(conn)
	ctx := context.Background()
	startTime := int32(time.Now().Unix())

	for {
		log.Print("Send: ", cfg.name)
		_, err := client.Send(ctx, &pb.Metrics{
			Name:           cfg.name,
			Uuid:           cfg.uuid,
			GoroutineCount: int32(runtime.NumGoroutine()),
			Memory:         getMemory(),
			Lifetime:       startTime,
		})
		if err != nil {
			log.Print(err)
		}
		time.Sleep(5 * time.Second)
	}
}

func Handle(addr, port, dur, name string) {
	newUuid, _ := uuid.NewRandom()

	var cfg config
	cfg.addr = addr
	cfg.port = port
	cfg.dur = dur
	cfg.name = name
	cfg.uuid = newUuid.String()

	go startMetrics(&cfg)
}
