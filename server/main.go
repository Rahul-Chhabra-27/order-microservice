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
	"rahulchhabra.io/config"
	"rahulchhabra.io/jwt"
	"rahulchhabra.io/model"
	orderproto "rahulchhabra.io/proto/order"
)

type OrderService struct {
	orderproto.UnimplementedOrderServiceServer
}

func (*OrderService) CreateOrder(ctx context.Context, req *orderproto.OrderRequest) (*orderproto.OrderResponse, error) {
	fmt.Println("Creating Order")
	email, ok := ctx.Value("email").(string)
	if !ok {
		// Internal server error..
		return nil, config.ErrorMessage("Email not found in context", codes.Internal)
	}
	// Create a new order
	myOrder := model.Order{
		UserEmail: email,
		Name:      req.Order.Name,
		Price:     req.Order.Price,
		Quantity:  req.Order.Quantity,
		CreatedAt: primitive.DateTime(time.Now().UnixNano() / int64(time.Millisecond)),
		UpdatedAt: primitive.DateTime(time.Now().UnixNano() / int64(time.Millisecond)),
	}
	// Check if the order is already placed
	if config.CheckIfTheOrderIsAlreadyExist(email,req.Order.Name) {
		err := config.UpdateOrder(myOrder,ctx);
		if err != nil {
			log.Fatalf("Failed to update order: %s", err)
			return nil,err;
		} else {
			return &orderproto.OrderResponse{
				Message:    "Bravo! Order has been updated successfully",
				StatusCode: int64(codes.OK),
			},nil
		}
	} else {
		return config.PlaceOrder(myOrder)
	}
}
func (*OrderService) GetOrders(ctx context.Context, request *orderproto.GetOrdersRequest) (*orderproto.GetOrdersResponse, error) {
	fmt.Println("Fetching orders")
	email, ok := ctx.Value("email").(string)
	if !ok {
		return nil, config.ErrorMessage("Email not found in context", codes.Internal)
	}
	return config.GetOrders(email);
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
	model.OrderCollection = db.Database("testdb").Collection("orders")

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
