package main

import (
	pb "github.com/thatlq1812/policy-system/shared/pkg/api/user"
	"github.com/thatlq1812/policy-system/user/internal/handler"
)

func main() {
	// This will fail at compile time if UserHandler doesn't implement UserServiceServer
	var _ pb.UserServiceServer = &handler.UserHandler{}

	// Check if service implements IsTokenBlacklisted
	var _ pb.UserServiceServer = (*handler.UserHandler)(nil)

	println("UserHandler implements UserServiceServer correctly including IsTokenBlacklisted")
}
