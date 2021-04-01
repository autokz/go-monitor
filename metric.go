package metric_for_services

import (
	"context"
	"log"
	"net"
	"runtime"
	"runtime/metrics"
	"time"

	pb "github.com/autokz/go-monitor/proto"
	"github.com/google/uuid"
	"google.golang.org/grpc"
)

type config struct {
	addr, port, dur, name, uuid string
	connectionTimeout int
}

var cfg config

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
	timeoutDuration := time.Duration(cfg.connectionTimeout) * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)

	startTime := int32(time.Now().Unix())

	for {
		go func (ctx context.Context, cancel context.CancelFunc) {
			defer cancel()
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
		}(ctx, cancel)
		time.Sleep(5 * time.Second)
	}
}

func GetUuid() string {
	return cfg.uuid
}

func Handle(addr, port, dur, name string, t int) {
	newUuid, _ := uuid.NewRandom()

	cfg.addr = addr
	cfg.port = port
	cfg.dur = dur
	cfg.name = name
	cfg.connectionTimeout =  t
	cfg.uuid = newUuid.String()


	go startMetrics(&cfg)
}
