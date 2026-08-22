package service

import (
	"chess-engine/app/constant"
	"chess-engine/app/domain/dao"
	"chess-engine/app/domain/dto"
	"chess-engine/app/middleware"
	"chess-engine/app/pkg"
	"chess-engine/app/repository"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
)

type UserService interface {
	GetAllUser(c *gin.Context)
	GetUserByToken(c *gin.Context)
	AddUserData(c *gin.Context)
	UpdateUserData(c *gin.Context)
	DeleteUser(c *gin.Context)
}

type UserServiceImpl struct {
	userRepository repository.UserRepository
}

// requireSelf resolves the authenticated user and the :userID path parameter and
// rejects the request unless they are the same account. Without this, PUT/DELETE
// on /api/user/:userID acted on any id the caller supplied.
func requireSelf(c *gin.Context) dao.User {
	authUser, ok := middleware.AuthUser(c)
	if !ok {
		log.Error("route is missing the auth middleware")
		pkg.PanicException(constant.Unauthorized)
	}

	userID, err := strconv.Atoi(c.Param("userID"))
	if err != nil {
		log.Error("Invalid userID path parameter:", err)
		pkg.PanicException(constant.InvalidRequest)
	}
	if userID != authUser.ID {
		log.Errorf("user %d attempted to act on user %d", authUser.ID, userID)
		pkg.PanicException(constant.Unauthorized)
	}
	return authUser
}

func (u UserServiceImpl) UpdateUserData(c *gin.Context) {
	defer pkg.PanicHandler(c)

	log.Info("start to execute program update user data by id")
	authUser := requireSelf(c)

	var request dto.UpdateUserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Error("Happened error when mapping request from FE. Error", err)
		pkg.PanicException(constant.InvalidRequest)
	}

	name := strings.TrimSpace(request.Name)
	if name == "" {
		log.Error("Update rejected: name is empty")
		pkg.PanicException(constant.InvalidRequest)
	}

	data, err := u.userRepository.FindUserById(authUser.ID)
	if err != nil {
		log.Error("Happened error when get data from database. Error", err)
		pkg.PanicException(constant.DataNotFound)
	}

	// Actually apply the request. This method used to bind the body, discard it
	// entirely, hardcode Status = 1, and then re-check the *previous* call's
	// error variable instead of the one from Save.
	data.Name = name

	saved, err := u.userRepository.Save(&data)
	if err != nil {
		log.Error("Happened error when updating data to database. Error", err)
		pkg.PanicException(constant.UnknownError)
	}

	c.JSON(http.StatusOK, pkg.BuildResponse(constant.Success, saved))
}

func (u UserServiceImpl) GetUserByToken(c *gin.Context) {
	defer pkg.PanicHandler(c)

	var request dto.TokenGetRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		log.Error("Happened error when mapping request from FE. Error", err)
		pkg.PanicException(constant.InvalidRequest)
	}
	log.Info("start to execute program get user by token")
	data, err := u.userRepository.FindUserByToken(request.Token)
	if err != nil || data.ID == 0 {
		log.Error("Happened error when get data from database. Error", err)
		pkg.PanicException(constant.DataNotFound)
	}

	c.JSON(http.StatusOK, pkg.BuildResponse(constant.Success, data))
}

func (u UserServiceImpl) AddUserData(c *gin.Context) {
	defer pkg.PanicHandler(c)

	log.Info("start to execute program add data user")
	request := dao.User{
		Token:  pkg.GenerateRandomString(80),
		Name:   pkg.GenerateRandomUserName(),
		Status: 1,
	}

	data, err := u.userRepository.Save(&request)
	if err != nil {
		log.Error("Happened error when saving data to database. Error", err)
		pkg.PanicException(constant.UnknownError)
	}

	// The only response that carries the token: dao.User hides it so the game
	// broadcast cannot leak it.
	c.JSON(http.StatusOK, pkg.BuildResponse(constant.Success, dto.CreatedUser{
		ID:       data.ID,
		Name:     data.Name,
		Status:   data.Status,
		MetaData: data.MetaData,
		Token:    data.Token,
	}))
}

func (u UserServiceImpl) GetAllUser(c *gin.Context) {
	defer pkg.PanicHandler(c)

	log.Info("start to execute get all data user")

	data, err := u.userRepository.FindAllUser()
	if err != nil {
		log.Error("Happened Error when find all user data. Error: ", err)
		pkg.PanicException(constant.UnknownError)
	}

	c.JSON(http.StatusOK, pkg.BuildResponse(constant.Success, data))
}

func (u UserServiceImpl) DeleteUser(c *gin.Context) {
	defer pkg.PanicHandler(c)

	log.Info("start to execute delete data user by id")
	authUser := requireSelf(c)

	if err := u.userRepository.DeleteUserById(authUser.ID); err != nil {
		log.Error("Happened Error when try delete data user from DB. Error:", err)
		pkg.PanicException(constant.UnknownError)
	}

	c.JSON(http.StatusOK, pkg.BuildResponse(constant.Success, pkg.Null()))
}

func UserServiceInit(userRepository repository.UserRepository) *UserServiceImpl {
	return &UserServiceImpl{
		userRepository: userRepository,
	}
}
