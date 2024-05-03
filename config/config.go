package config

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc/codes"
	"rahulchhabra.io/model"
	orderproto "rahulchhabra.io/proto/order"
)

func CheckIfTheOrderIsAlreadyExist(email string, name string) bool {
	orderplaced := model.Order{UserEmail: email, Name: name}
	if model.OrderCollection.FindOne(context.Background(), orderplaced).Err() == nil {
		return true
	} else {
		return false
	}
}

// create a function that updates order from the database

func UpdateOrder(order model.Order, ctx context.Context) error {
	update := bson.M{
		"$set": bson.M{
			"quantity":  order.Quantity,
			"updatedAt": order.UpdatedAt,
		},
	}
	err := model.OrderCollection.FindOneAndUpdate(context.Background(), model.Order{Name: order.Name, UserEmail: order.UserEmail}, update, options.FindOneAndUpdate().SetReturnDocument(1))
	if err.Err() != nil {
		return ErrorMessage(err.Err().Error(), codes.Internal)
	}
	return nil
}
func PlaceOrder(myorder model.Order) (*orderproto.OrderResponse, error) {
	_, err := model.OrderCollection.InsertOne(context.Background(), myorder)
	if err != nil {
		return nil, ErrorMessage("Could not place order", codes.Internal)
	}
	return &orderproto.OrderResponse{
		Message:    "Bravo! Order has been placed successfully",
		StatusCode: int64(codes.OK),
	}, nil
}
