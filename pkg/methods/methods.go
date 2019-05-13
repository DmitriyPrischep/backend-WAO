package methods

import (
	"github.com/DmitriyPrischep/backend-WAO/pkg/model"
)

type UserMethods interface {
	GetUser(userdata model.NicknameUser) (user *model.User, err error)
	GetUsers() (users []model.User, err error)
	CreateUser(user model.UserRegister) (nickname string, err error)
	UpdateUser(user model.UpdateDataImport) (out model.UpdateDataExport, err error)
	CheckUser(user model.SigninUser) (out *model.UserRegister, err error)
}