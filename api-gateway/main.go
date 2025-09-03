package main

import (
	"context"
	"gateway/config"
	"gateway/proto/auth"
	"gateway/proto/stakeholders"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func newProxy(targetHost string) *httputil.ReverseProxy {
	target, _ := url.Parse(targetHost)
	return httputil.NewSingleHostReverseProxy(target)
}
func pathHasPrefix(path, prefix string) bool {
	return len(path) >= len(prefix) && path[:len(prefix)] == prefix
}
func authMiddleware(client auth.AuthServiceClient, excludedFromAuth []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, path := range excludedFromAuth {
				if r.URL.Path == path {
					next.ServeHTTP(w, r)
					return
				}
			}
			token := r.Header.Get("Authorization")
			if token == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			req := &auth.EmptyMessage{}
			md := metadata.Pairs("authorization", token)
			ctx := metadata.NewOutgoingContext(r.Context(), md)
			resp, err := client.GetToken(ctx, req)
			if err != nil || !resp.IsValid {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			r.Header.Set("X-User-Role", resp.Role)
			r.Header.Set("X-User-Id", strconv.FormatInt(resp.UserId, 10))
			r.Header.Set("X-Person-Id", strconv.FormatInt(resp.PersonId, 10))
			next.ServeHTTP(w, r)
		})
	}
}
func main() {
	cfg := config.GetConfig()

	conn, err := grpc.DialContext(
		context.Background(),
		cfg.AuthAndStakeholdersGRPCServiceAddress,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)

	if err != nil {
		log.Fatalln("Failed to dial auth server:", err)
	}

	gwmux := runtime.NewServeMux()
	// Register Auth
	authClient := auth.NewAuthServiceClient(conn)
	err = auth.RegisterAuthServiceHandlerClient(
		context.Background(),
		gwmux,
		authClient,
	)
	if err != nil {
		log.Fatalln("Failed to register auth service:", err)
	}
	userClient := stakeholders.NewUserServiceClient(conn)
	err = stakeholders.RegisterUserServiceHandlerClient(
		context.Background(),
		gwmux,
		userClient,
	)
	if err != nil {
		log.Fatalln("Failed to register user service:", err)
	}
	personClient := stakeholders.NewPersonServiceClient(conn)
	err = stakeholders.RegisterPersonServiceHandlerClient(
		context.Background(),
		gwmux,
		personClient,
	)
	if err != nil {
		log.Fatalln("Failed to register person service:", err)
	}

	mux := http.NewServeMux()

	mux.Handle("/api/users", newProxy(cfg.AuthAndStakeholdersHTTPServiceAddress))
	mux.Handle("/api/users/", newProxy(cfg.AuthAndStakeholdersHTTPServiceAddress))
	mux.Handle("/api/profile", newProxy(cfg.AuthAndStakeholdersHTTPServiceAddress))
	mux.Handle("/api/profile/", newProxy(cfg.AuthAndStakeholdersHTTPServiceAddress))
	mux.Handle("/api/tours", newProxy(cfg.TourServiceAddress))
	mux.Handle("/api/tours/", newProxy(cfg.TourServiceAddress))
	mux.Handle("/api/blogs", newProxy(cfg.BlogServiceAddress))
	mux.Handle("/api/blogs/", newProxy(cfg.BlogServiceAddress))
	

	combinedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if pathHasPrefix(path, "/api/auth") {
			gwmux.ServeHTTP(w, r)
			return
		}
		log.Println(path)
		mux.ServeHTTP(w, r)
	})

	excludedFromAuth := []string{
		"/api/auth/validate",
		"/api/auth/login",
		"/api/auth",
		"/api/blogs",
	}

	handlerWithMiddleware := authMiddleware(authClient, excludedFromAuth)(combinedHandler)

	gwServer := &http.Server{
		Addr:    cfg.Address,
		Handler: handlerWithMiddleware,
	}

	go func() {
		log.Println("Starting HTTP server on", cfg.Address)
		if err := gwServer.ListenAndServe(); err != nil {
			log.Fatal("server error: ", err)
		}
	}()

	stopCh := make(chan os.Signal)
	signal.Notify(stopCh, syscall.SIGTERM)

	<-stopCh

	if err = gwServer.Close(); err != nil {
		log.Fatalln("error while stopping server: ", err)
	}
}
