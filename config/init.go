package config

import (
	"chess-engine/app/controller"
	"chess-engine/app/repository"
	"chess-engine/app/service"
)

type Initialization struct {
	userRepo   repository.UserRepository
	userSvc    service.UserService
	UserCtrl   controller.UserController
	RoleRepo   repository.RoleRepository
	ChessCtrl  controller.ChessController
	chessSvc   service.ChessService
	chessRepo  repository.ChessRepository
	SocketCtrl controller.WebSocketController
	socketSvc  service.WebSocketService
}

func NewInitialization(userRepo repository.UserRepository,
	userService service.UserService,
	userCtrl controller.UserController,
	roleRepo repository.RoleRepository,
	ChessCtrl controller.ChessController,
	chessSvc service.ChessService,
	chessRepo repository.ChessRepository,
	socketSvc service.WebSocketService,
	SocketCtrl controller.WebSocketController) *Initialization {
	return &Initialization{
		userRepo:   userRepo,
		userSvc:    userService,
		UserCtrl:   userCtrl,
		RoleRepo:   roleRepo,
		ChessCtrl:  ChessCtrl,
		chessSvc:   chessSvc,
		chessRepo:  chessRepo,
		socketSvc:  socketSvc,
		SocketCtrl: SocketCtrl,
	}
}
