package config

import "os"

type Config struct {
	Address                               string
	AuthAndStakeholdersGRPCServiceAddress string
	AuthAndStakeholdersHTTPServiceAddress string
	TourServiceAddress                    string
	BlogServiceAddress                    string
}

func GetConfig() Config {
	authStakeholdersAddrGrpc := os.Getenv("AUTH_STAKEHOLDERS_SERVICE_GRPC_ADDRESS")
	if authStakeholdersAddrGrpc == "" {
		authStakeholdersAddrGrpc = "127.0.0.1:8888"
	}
	authStakeholderAddHttp := os.Getenv("AUTH_STAKEHOLDERS_SERVICE_HTTP_ADDRESS")
	if authStakeholderAddHttp == "" {
		authStakeholderAddHttp = "http://localhost:8080"
	}
	gatewayAddr := os.Getenv("GATEWAY_ADDRESS")
	if gatewayAddr == "" {
		gatewayAddr = "localhost:7070"
	}
	blogAddr := os.Getenv("BLOG_SERVICE_ADDRESS")
	if blogAddr == "" {
		blogAddr = "http://localhost:8083"
	}

	tourAddr := os.Getenv("TOUR_ADDRESS")
	if tourAddr == "" {
		tourAddr = "http://localhost:8081"
	}
	return Config{
		AuthAndStakeholdersGRPCServiceAddress: authStakeholdersAddrGrpc,
		Address:                               gatewayAddr,
		AuthAndStakeholdersHTTPServiceAddress: authStakeholderAddHttp,
		TourServiceAddress:                    tourAddr,
		BlogServiceAddress:                    blogAddr,
	}
}
