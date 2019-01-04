 package main

import (

	
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/easyCZ/grpc-web-hacker-news/server/proxy"
	"github.com/go-chi/chi"
	chiMiddleware "github.com/go-chi/chi/middleware"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/rs/cors"
	"github.com/salihkemaloglu/Demserver-beta-01/proto"
	"github.com/salihkemaloglu/Demserver-beta-01/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/status"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var db *mgo.Database

//DB string
var DB string

type server struct {
}


type userItem struct {
	ID       bson.ObjectId `bson:"_id,omitempty"`
	UserNameSurname string            `bson:"author_id"`
	Username  string            `bson:"content"`
	Password    string            `bson:"title"`
}


func (*server) RegisterUser(ctx context.Context, req *dempb.RegisterUserRequest) (*dempb.RegisterUserResponse, error) {
	fmt.Println("Create blog request")
	blog := req.GetUser()

	data := userItem{
		Username: blog.GetId(),
		UserNameSurname:    blog.GetUserNameSurname(),
		Password:  blog.GetUsername(),
	}

	err := db.C("item").Insert(&data)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error: %v", err),
		)
	}
	

	return &dempb.RegisterUserResponse{
		User: &dempb.User{
			Id:       blog.GetId(),
			UserNameSurname: blog.GetUserNameSurname(),
			Username:    blog.GetUsername(),
			Password:  blog.GetPassword(),
		},
	}, nil

}

func (*server) LoginUser(ctx context.Context, req *dempb.LoginUserRequest) (*dempb.LoginUserResponse, error) {
	fmt.Println("Read blog request")

	// blogID := req.GetUserId()
	//  oid := bson.ObjectIdHex(*blogID)
	
	// // create an empty struct
	 data := &userItem{}
	// filter := bson.NewDocument(bson.EC.ObjectID("_id", oid))

	// res := collection.FindOne(context.Background(), filter)
	// if err := res.Decode(data); err != nil {
	// 	return nil, status.Errorf(
	// 		codes.NotFound,
	// 		fmt.Sprintf("Cannot find blog with specified ID: %v", err),
	// 	)
	// }

	return &dempb.LoginUserResponse{
		User: dataToDemPb(data),
	}, nil
}

func dataToDemPb(data *userItem) *dempb.User {
	return &dempb.User{
		Id:       data.ID.Hex(),
		UserNameSurname: data.UserNameSurname,
		Username:  data.Username,
		Password:    data.Password,
	}
}

func (*server) UpdateUser(ctx context.Context, req *dempb.UpdateUserRequest) (*dempb.UpdateUserResponse, error) {
	fmt.Println("Update blog request")
	// blog := req.GetBlog()
	// oid, err := objectid.FromHex(blog.GetId())
	// if err != nil {
	// 	return nil, status.Errorf(
	// 		codes.InvalidArgument,
	// 		fmt.Sprintf("Cannot parse ID"),
	// 	)
	// }

	// create an empty struct
	data := &userItem{}
	// filter := bson.NewDocument(bson.EC.ObjectID("_id", oid))

	// res := collection.FindOne(context.Background(), filter)
	// if err := res.Decode(data); err != nil {
	// 	return nil, status.Errorf(
	// 		codes.NotFound,
	// 		fmt.Sprintf("Cannot find blog with specified ID: %v", err),
	// 	)
	// }

	// // we update our internal struct
	// data.AuthorID = blog.GetAuthorId()
	// data.Content = blog.GetContent()
	// data.Title = blog.GetTitle()

	// _, updateErr := collection.ReplaceOne(context.Background(), filter, data)
	// if updateErr != nil {
	// 	return nil, status.Errorf(
	// 		codes.Internal,
	// 		fmt.Sprintf("Cannot update object in MongoDB: %v", updateErr),
	// 	)
	// }

	return &dempb.UpdateUserResponse{
		User: dataToDemPb(data),
	}, nil

}

func (*server) DeleteUser(ctx context.Context, req *dempb.DeleteUserRequest) (*dempb.DeleteUserResponse, error) {
	fmt.Println("Delete blog request")
	// oid, err := objectid.FromHex(req.GetBlogId())
	// if err != nil {
	// 	return nil, status.Errorf(
	// 		codes.InvalidArgument,
	// 		fmt.Sprintf("Cannot parse ID"),
	// 	)
	// }

	// filter := bson.NewDocument(bson.EC.ObjectID("_id", oid))

	// res, err := collection.DeleteOne(context.Background(), filter)

	// if err != nil {
	// 	return nil, status.Errorf(
	// 		codes.Internal,
	// 		fmt.Sprintf("Cannot delete object in MongoDB: %v", err),
	// 	)
	// }

	// if res.DeletedCount == 0 {
	// 	return nil, status.Errorf(
	// 		codes.NotFound,
	// 		fmt.Sprintf("Cannot find blog in MongoDB: %v", err),
	// 	)
	// }

	return &dempb.DeleteUserResponse{UserId: req.GetUserId()}, nil
}

func (s *server) SayHello(ctx context.Context, in *dempb.HelloRequest) (*dempb.HelloResponse, error) {
	fmt.Printf("received rpc from client, name=%s\n", in.GetName())
	return &dempb.HelloResponse{Message: "Hello " + in.Name}, nil
}
func main() {
	// if we crash the go code, we get the file name and line number
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	fmt.Println("Connecting to MongoDB")
	LoadConfiguration()

	fmt.Println("Blog Service Started")
	// collection = client.Database("mydb").Collection("blog")

	opts := []grpc.ServerOption{}
	grpcServer := grpc.NewServer(opts...)
	dempb.RegisterDemServiceServer(grpcServer, &server{})

	wrappedGrpc := grpcweb.WrapServer(grpcServer)

	router := chi.NewRouter()
	router.Use(
		chiMiddleware.Logger,
		chiMiddleware.Recoverer,
		middleware. NewGrpcWebMiddleware(wrappedGrpc).Handler,// Must come before general CORS handling
		cors.New(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: true,
			MaxAge:           300, // Maximum value not ignored by any of major browsers
		}).Handler,
	)

	router.Get("/article-proxy", proxy.Article)

	if err := http.ListenAndServe(":8900", router); err != nil {
		grpclog.Fatalf("failed starting http2 server: %v", err)
	}
}

//Connect Establish a connection to database
func Connect(connectionUrl string) {
	info := &mgo.DialInfo{
		Addrs:    []string{connectionUrl},
		Timeout:  5 * time.Second,
		Database: DB,
		Username: "",
		Password: "",
	}
	session, err := mgo.DialWithInfo(info)
	if err != nil {
		fmt.Println(err.Error())
	}
	db = session.DB(DB)
}

//LoadConfiguration Parse the configuration file 'config.toml', and establish a connection to DB
func LoadConfiguration() {
	var url = "localhost:27017"
	DB = "Blogs"
	Connect(url)
}
