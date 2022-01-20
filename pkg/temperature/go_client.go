package temperature

import (
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"log"
)

func main() {
	var conn *grpc.ClientConn
	conn, err := grpc.Dial(":9090", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}
	defer conn.Close()

	client := NewTempServiceClient(conn)

	response, err := client.GetDailyMaxTempByDateAndByID(
		context.Background(), &SensorIdDate{SensorId: "123", Date: "18/01/2022"})
	if err != nil {
		log.Fatalf("Error when calling GetDailyMaxTempByDateAndByID: %s", err)
	}
	log.Printf("Response from server: %s", response.Value)
}
