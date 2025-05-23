// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package config

import (
	"chess-engine/app/controller"
	"chess-engine/app/repository"
	"chess-engine/app/service"
	"github.com/google/wire"
)

// Injectors from injector.go:

func Init() *Initialization {
	gormDB := ConnectToDB()
	redisClient := InitRedis() // Initialize Redis

	userRepositoryImpl := repository.UserRepositoryInit(gormDB)
	userServiceImpl := service.UserServiceInit(userRepositoryImpl)
	userControllerImpl := controller.UserControllerInit(userServiceImpl)
	roleRepositoryImpl := repository.RoleRepositoryInit(gormDB)
	chessRepositoryImpl := repository.ChessRepositoryInit(gormDB, redisClient)
	chessServiceImpl := service.ChessServiceInit(chessRepositoryImpl)
	chessControllerImpl := controller.ChessControllerInit(chessServiceImpl)
	socketServiceImpl := service.WebSocketServiceInit(chessRepositoryImpl)
	socketControllerImpl := controller.WebSocketControllerInit(socketServiceImpl)

	go service.WebSocketServiceInit(chessRepositoryImpl)
	
	initialization := NewInitialization(userRepositoryImpl, userServiceImpl, userControllerImpl, roleRepositoryImpl, chessControllerImpl, chessServiceImpl, chessRepositoryImpl, socketServiceImpl, socketControllerImpl)
	return initialization
}

// injector.go:

var db = wire.NewSet(ConnectToDB)

var userServiceSet = wire.NewSet(service.UserServiceInit, wire.Bind(new(service.UserService), new(*service.UserServiceImpl)))

var userRepoSet = wire.NewSet(repository.UserRepositoryInit, wire.Bind(new(repository.UserRepository), new(*repository.UserRepositoryImpl)))

var userCtrlSet = wire.NewSet(controller.UserControllerInit, wire.Bind(new(controller.UserController), new(*controller.UserControllerImpl)))

var roleRepoSet = wire.NewSet(repository.RoleRepositoryInit, wire.Bind(new(repository.RoleRepository), new(*repository.RoleRepositoryImpl)))

var chessRepoSet = wire.NewSet(repository.ChessRepositoryInit, wire.Bind(new(repository.ChessRepository), new(*repository.ChessRepositoryImpl)))

var chessCtrlSet = wire.NewSet(controller.ChessControllerInit, wire.Bind(new(controller.ChessController), new(*controller.ChessControllerImpl)))

var chessSvcSet = wire.NewSet(service.ChessServiceInit, wire.Bind(new(service.ChessService), new(*service.ChessServiceImpl)))

var socketSvcSet = wire.NewSet(service.WebSocketServiceInit, wire.Bind(new(service.WebSocketService), new(*service.WebSocketServiceImpl)))

var socketCtrlSet = wire.NewSet(controller.WebSocketControllerInit, wire.Bind(new(controller.WebSocketController), new(*controller.WebSocketControllerImpl)))