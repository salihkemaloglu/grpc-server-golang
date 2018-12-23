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
	// "github.com/salihkemaloglu/gRPCExample/greet/greetpb"
	// "github.com/salihkemaloglu/gRPCExample/middleware"
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

type blogItem struct {
	ID       bson.ObjectId `bson:"_id,omitempty"`
	AuthorID string        `bson:"author_id"`
	Content  string        `bson:"content"`
	Title    string        `bson:"title"`
}

func (*server) CreateBlog(ctx context.Context, req *blogpb.CreateBlogRequest) (*blogpb.CreateBlogResponse, error) {
	fmt.Println("Create blog request")
	blog := req.GetBlog()

	data := blogItem{
		AuthorID: blog.GetAuthorId(),
		Title:    blog.GetTitle(),
		Content:  blog.GetContent(),
	}

	err := db.C("item").Insert(&data)
	if err != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Internal error: %v", err),
		)
	}

	return &blogpb.CreateBlogResponse{
		Blog: &blogpb.Blog{
			Id:       blog.GetId(),
			AuthorId: blog.GetAuthorId(),
			Title:    blog.GetTitle(),
			Content:  blog.GetContent(),
		},
	}, nil

}

func (*server) ReadBlog(ctx context.Context, req *blogpb.ReadBlogRequest) (*blogpb.ReadBlogResponse, error) {
	fmt.Println("Read blog request")

	blogID := req.GetBlogId()
	oid := bson.ObjectIdHex(blogID)

	// create an empty struct
	data := &blogItem{}

	err := db.C("item").FindId(oid).One(&data)
	if err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Cannot find blog with specified ID: %v", err),
		)
	}
	return &blogpb.ReadBlogResponse{
		Blog: dataToBlogPb(data),
	}, nil
}

func dataToBlogPb(data *blogItem) *blogpb.Blog {
	return &blogpb.Blog{
		Id:       data.ID.Hex(),
		AuthorId: data.AuthorID,
		Content:  data.Content,
		Title:    data.Title,
	}
}

func (*server) UpdateBlog(ctx context.Context, req *blogpb.UpdateBlogRequest) (*blogpb.UpdateBlogResponse, error) {
	fmt.Println("Update blog request")
	blog := req.GetBlog()
	oid := bson.ObjectIdHex(blog.GetId())

	// create an empty struct
	data := &blogItem{}
	err := db.C("item").FindId(oid).One(&data)
	if err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Cannot find blog with specified ID: %v", err),
		)
	}

	// we update our internal struct
	data.AuthorID = blog.GetAuthorId()
	data.Content = blog.GetContent()
	data.Title = blog.GetTitle()

	updateErr := db.C("item").Update(bson.M{"_id": data.ID}, &data)
	if updateErr != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Cannot update object in MongoDB: %v", updateErr),
		)
	}

	return &blogpb.UpdateBlogResponse{
		Blog: dataToBlogPb(data),
	}, nil

}

func (*server) DeleteBlog(ctx context.Context, req *blogpb.DeleteBlogRequest) (*blogpb.DeleteBlogResponse, error) {
	fmt.Println("Delete blog request")
	oid := bson.ObjectIdHex(req.GetBlogId())

	data := &blogItem{}
	err := db.C("item").FindId(oid).One(&data)
	if err != nil {
		return nil, status.Errorf(
			codes.NotFound,
			fmt.Sprintf("Cannot find blog with specified ID: %v", err),
		)
	}

	errDelete := db.C("item").Remove(&data)

	if errDelete != nil {
		return nil, status.Errorf(
			codes.Internal,
			fmt.Sprintf("Cannot delete object in MongoDB: %v", err),
		)
	}

	return &blogpb.DeleteBlogResponse{BlogId: req.GetBlogId()}, nil
}

func (*server) ListBlog(req *blogpb.ListBlogRequest, stream blogpb.BlogService_ListBlogServer) error {
	fmt.Println("List blog request")
	var blog []blogItem
	err := db.C("user").Find(bson.M{}).All(&blog)
	if err != nil {
		return status.Errorf(
			codes.Internal,
			fmt.Sprintf("Unknown internal error: %v", err),
		)
	}
	// for cur.Next(context.Background()) {
	// 	data := &blogItem{}
	// 	err := cur.Decode(data)
	// 	if err != nil {
	// 		return status.Errorf(
	// 			codes.Internal,
	// 			fmt.Sprintf("Error while decoding data from MongoDB: %v", err),
	// 		)

	// 	}
	// 	stream.Send(&blogpb.ListBlogResponse{Blog: dataToBlogPb(data)})
	// }
	// if err := cur.Err(); err != nil {
	// 	return status.Errorf(
	// 		codes.Internal,
	// 		fmt.Sprintf("Unknown internal error: %v", err),
	// 	)
	// }
	return nil
}

func (*server) SayHello(ctx context.Context, req *blogpb.HelloRequest) (*blogpb.HelloResponse, error) {
	fmt.Printf("received rpc from client, name=%s\n", req.GetName())
	return &blogpb.HelloResponse{Message: "Hello " + req.Name}, nil
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
	blogpb.RegisterBlogServiceServer(grpcServer, &server{})

	wrappedGrpc := grpcweb.WrapServer(grpcServer)

	router := chi.NewRouter()
	router.Use(
		chiMiddleware.Logger,
		chiMiddleware.Recoverer,
		middleware.NewGrpcWebMiddleware(wrappedGrpc).Handler, // Must come before general CORS handling
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
