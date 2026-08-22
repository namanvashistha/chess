package controller

import (
	"chess-engine/app/constant"
	"chess-engine/app/domain/dao"
	"chess-engine/app/domain/dto"
	"chess-engine/app/middleware"
	"chess-engine/app/pkg"
	"chess-engine/app/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

type UserController interface {
	GetAllUserData(c *gin.Context)
	AddUserData(c *gin.Context)
	GetUserByToken(c *gin.Context)
	UpdateUserData(c *gin.Context)
	DeleteUser(c *gin.Context)
}

type UserControllerImpl struct {
	svc service.UserService
}

func (u UserControllerImpl) GetAllUserData(c *gin.Context) {
	data, err := u.svc.GetAllUser()
	respond(c, data, err)
}

func (u UserControllerImpl) AddUserData(c *gin.Context) {
	data, err := u.svc.AddUser()
	respond(c, data, err)
}

func (u UserControllerImpl) GetUserByToken(c *gin.Context) {
	var request dto.TokenGetRequest
	if err := bindJSON(c, &request); err != nil {
		respond(c, nil, err)
		return
	}
	data, err := u.svc.GetUserByToken(request.Token)
	respond(c, data, err)
}

func (u UserControllerImpl) UpdateUserData(c *gin.Context) {
	actor, userID, err := actorAndPathUser(c)
	if err != nil {
		respond(c, nil, err)
		return
	}
	var request dto.UpdateUserRequest
	if err := bindJSON(c, &request); err != nil {
		respond(c, nil, err)
		return
	}
	data, updErr := u.svc.UpdateUser(actor, userID, request.Name)
	respond(c, data, updErr)
}

func (u UserControllerImpl) DeleteUser(c *gin.Context) {
	actor, userID, err := actorAndPathUser(c)
	if err != nil {
		respond(c, nil, err)
		return
	}
	respond(c, pkg.Null(), u.svc.DeleteUser(actor, userID))
}

// actorAndPathUser pulls the authenticated user from the context and the target
// id from the path.
func actorAndPathUser(c *gin.Context) (actor dao.User, userID int, err error) {
	actor, ok := middleware.AuthUser(c)
	if !ok {
		return actor, 0, pkg.NewError(constant.Unauthorized, "not authenticated", nil)
	}
	userID, convErr := strconv.Atoi(c.Param("userID"))
	if convErr != nil {
		return actor, 0, pkg.NewError(constant.InvalidRequest, "invalid user id", convErr)
	}
	return actor, userID, nil
}

func UserControllerInit(userService service.UserService) *UserControllerImpl {
	return &UserControllerImpl{svc: userService}
}
