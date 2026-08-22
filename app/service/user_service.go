package service

import (
	"chess-engine/app/constant"
	"chess-engine/app/domain/dao"
	"chess-engine/app/domain/dto"
	"chess-engine/app/pkg"
	"chess-engine/app/repository"
	"strings"
)

// UserService is transport-agnostic: it takes plain values and returns
// (value, error). It used to take *gin.Context, which welded HTTP into the
// business layer and made every method untestable without a live request.
type UserService interface {
	GetAllUser() ([]dao.User, error)
	GetUserByToken(token string) (dao.User, error)
	AddUser() (dto.CreatedUser, error)
	UpdateUser(actor dao.User, userID int, name string) (dao.User, error)
	DeleteUser(actor dao.User, userID int) error
}

type UserServiceImpl struct {
	userRepository repository.UserRepository
}

// authorizeSelf rejects any attempt to act on an account other than the
// caller's own.
func authorizeSelf(actor dao.User, userID int) error {
	if actor.ID == 0 {
		return pkg.NewError(constant.Unauthorized, "not authenticated", nil)
	}
	if actor.ID != userID {
		return pkg.NewError(constant.Unauthorized, "cannot act on another account", nil)
	}
	return nil
}

func (u UserServiceImpl) UpdateUser(actor dao.User, userID int, name string) (dao.User, error) {
	if err := authorizeSelf(actor, userID); err != nil {
		return dao.User{}, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return dao.User{}, pkg.NewError(constant.InvalidRequest, "name is required", nil)
	}

	data, err := u.userRepository.FindUserById(actor.ID)
	if err != nil {
		return dao.User{}, pkg.NewError(constant.DataNotFound, "user not found", err)
	}

	data.Name = name
	saved, err := u.userRepository.Save(&data)
	if err != nil {
		return dao.User{}, pkg.NewError(constant.UnknownError, "could not save user", err)
	}
	return saved, nil
}

func (u UserServiceImpl) GetUserByToken(token string) (dao.User, error) {
	user, err := u.userRepository.FindUserByToken(token)
	if err != nil || user.ID == 0 {
		return dao.User{}, pkg.NewError(constant.DataNotFound, "user not found", err)
	}
	return user, nil
}

func (u UserServiceImpl) AddUser() (dto.CreatedUser, error) {
	request := dao.User{
		Token:  pkg.GenerateRandomString(80),
		Name:   pkg.GenerateRandomUserName(),
		Status: 1,
	}

	data, err := u.userRepository.Save(&request)
	if err != nil {
		return dto.CreatedUser{}, pkg.NewError(constant.UnknownError, "could not create user", err)
	}

	// The only response carrying the token: dao.User hides it so the game
	// broadcast cannot leak it.
	return dto.CreatedUser{
		ID:       data.ID,
		Name:     data.Name,
		Status:   data.Status,
		MetaData: data.MetaData,
		Token:    data.Token,
	}, nil
}

func (u UserServiceImpl) GetAllUser() ([]dao.User, error) {
	data, err := u.userRepository.FindAllUser()
	if err != nil {
		return nil, pkg.NewError(constant.UnknownError, "could not list users", err)
	}
	return data, nil
}

func (u UserServiceImpl) DeleteUser(actor dao.User, userID int) error {
	if err := authorizeSelf(actor, userID); err != nil {
		return err
	}
	if err := u.userRepository.DeleteUserById(actor.ID); err != nil {
		return pkg.NewError(constant.UnknownError, "could not delete user", err)
	}
	return nil
}

func UserServiceInit(userRepository repository.UserRepository) *UserServiceImpl {
	return &UserServiceImpl{userRepository: userRepository}
}
