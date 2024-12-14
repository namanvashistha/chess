// go:build wireinject
//go:build wireinject
// +build wireinject

package config

import (
	"chess-engine/app/controller"
	"chess-engine/app/repository"
	"chess-engine/app/service"

	"github.com/google/wire"
)

var db = wire.NewSet(ConnectToDB)

var userServiceSet = wire.NewSet(service.UserServiceInit,
	wire.Bind(new(service.UserService), new(*service.UserServiceImpl)),
)

var userRepoSet = wire.NewSet(repository.UserRepositoryInit,
	wire.Bind(new(repository.UserRepository), new(*repository.UserRepositoryImpl)),
)

var userCtrlSet = wire.NewSet(controller.UserControllerInit,
	wire.Bind(new(controller.UserController), new(*controller.UserControllerImpl)),
)

var roleRepoSet = wire.NewSet(repository.RoleRepositoryInit,
	wire.Bind(new(repository.RoleRepository), new(*repository.RoleRepositoryImpl)),
)

var chessRepoSet = wire.NewSet(repository.ChessRepositoryInit,
	wire.Bind(new(repository.ChessRepository), new(*repository.ChessRepositoryImpl)),
)

var chessCtrlSet = wire.NewSet(controller.ChessControllerInit,
	wire.Bind(new(controller.ChessController), new(*controller.ChessControllerImpl)),
)

var chessSvcSet = wire.NewSet(service.ChessServiceInit,
	wire.Bind(new(service.ChessService), new(*service.ChessServiceImpl)),
)

var socketSvcSet = wire.NewSet(service.WebSocketServiceInit,
	wire.Bind(new(service.WebSocketService), new(*service.WebSocketServiceImpl)),
)

var socketCtrlSet = wire.NewSet(controller.WebSocketControllerInit,
	wire.Bind(new(controller.WebSocketController), new(*controller.WebSocketControllerImpl)),
)

func Init() *Initialization {
	wire.Build(NewInitialization, db, userCtrlSet, userServiceSet, userRepoSet, roleRepoSet, chessCtrlSet, chessSvcSet, chessRepoSet, socketCtrlSet, socketSvcSet)
	return nil
}
