package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"rahulchhabra.io/jwt"
	orderproto "rahulchhabra.io/proto/order"
)

var OrderCollection *mongo.Collection

type OrderService struct {
	orderproto.UnimplementedOrderServiceServer
}
type Order struct {
	Id        primitive.ObjectID `bson:"_id,omitempty"`
	Name      string             `bson:"name,omitempty"`
	Price     int64              `bson:"price,omitempty"`
	Quantity  int64              `bson:"quantity,omitempty"`
	CreatedAt primitive.DateTime `bson:"createdAt,omitempty"`
	UpdatedAt primitive.DateTime `bson:"updatedAt,omitempty"`
}
type Orders struct {
	UserEmail  string             `bson:"email,omitempty"`
	OrderArray []Order            `bson:"orders,omitempty"`
	TotalPrice int64              `bson:"totalprice,omitempty"`
}

func (*OrderService) CreateOrder(ctx context.Context, req *orderproto.OrderRequest) (*orderproto.OrderResponse, error) {
	fmt.Println("Creating Order")
	email, ok := ctx.Value("email").(string)
	if !ok {
		return nil, status.Errorf(
			codes.Internal,
			"Unable to fetch userID from ctx",
		)
	}
	var orders []Order
	for _, order := range req.Orders {
		orders = append(orders, Order{
			Name:      order.Name,
			Price:     order.Price,
			Quantity:  order.Quantity,
			CreatedAt: primitive.DateTime(time.Now().UnixNano() / int64(time.Millisecond)),
			UpdatedAt: primitive.DateTime(time.Now().UnixNano() / int64(time.Millisecond)),
		})
	}

	myorder := Orders{
		UserEmail: email,
		OrderArray: orders,
		TotalPrice: int64(20),
	}
	_, err := OrderCollection.InsertOne(context.Background(), myorder)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error: %s", err),
		)
	}
	return &orderproto.OrderResponse{
		TotalPrice: int64(20),
		Orders:     req.Orders,
	}, nil
}

// Responsible for starting the server
func startServer() {
	godotenv.Load()
	// Log a message
	fmt.Println("Starting server...")
	// Initialize the gotenv file..
	godotenv.Load()

	// Create a new context
	ctx := context.TODO()

	// Connect to the MongoDB database
	db, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("MONGO_URI")))

	// Set the global variable to the collection
	OrderCollection = db.Database("testdb").Collection("orders")

	// Check for errors
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %s", err)
	}

	listner, err := net.Listen("tcp", "localhost:50052")
	if err != nil {
		log.Fatalf("Failed to start server: %s", err)
	}
	fmt.Println("Database connected Successfully")

	// Create a new gRPC server
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(jwt.UnaryInterceptor),
	)
	orderproto.RegisterOrderServiceServer(grpcServer, &OrderService{})

	go func() {
		if err := grpcServer.Serve(listner); err != nil {
			log.Fatalf("Failed to serve: %s", err)
		}
	}()
	// Create a new gRPC-Gateway server (gateway).
	connection, err := grpc.DialContext(
		context.Background(),
		"localhost:50052",
		grpc.WithBlock(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalln("Failed to dial server:", err)
	}
	// Create a new gRPC-Gateway mux (gateway).
	gwmux := runtime.NewServeMux()

	orderproto.RegisterOrderServiceHandler(context.Background(), gwmux, connection)
	// Create a new HTTP server (gateway). (Serve). (ListenAndServe)
	gwServer := &http.Server{
		Addr:    ":8091",
		Handler: gwmux,
	}

	log.Println("Serving gRPC-Gateway on http://0.0.0.0:8091")
	log.Fatalln(gwServer.ListenAndServe())
}

func main() {
	startServer()
}
