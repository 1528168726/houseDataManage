package backEnd

import (
	"errors"
	"golang.org/x/crypto/bcrypt"
)

var PwdNotMatch = errors.New("backEnd passwordManage.go: password not match")
var CanNotDeleteRoot = errors.New("backEnd passwordManage.go: can not delete root user")

func AddUser(id string, pwd string) error {
	hashedPSW, err := hashAndSalt(pwd)
	if err != nil {
		return err
	}
	sqlStr := `insert into tbPassword (id,pwd) values (?,?);`
	_, err = DB.Exec(sqlStr, id, hashedPSW)
	if err != nil {
		return err
	}
	return nil
}

func GetAllUser() (*[]string, error) {
	sqlStr := `select id from tbPassword;`
	tmpRes, err := DB.Query(sqlStr)
	if err != nil {
		return nil, err
	}
	res := new([]string)
	for tmpRes.Next() {
		var tmp string
		err = tmpRes.Scan(&tmp)
		*res = append(*res, tmp)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func CheckPSW(id string, pwd string) error {
	sqlStr := `select pwd from tbPassword where id = ?;`
	re := DB.QueryRow(sqlStr, id)
	var hashedPWD string
	err := re.Scan(&hashedPWD)
	if err != nil {
		return err
	}
	if !comparePassword(hashedPWD, pwd) {
		return PwdNotMatch
	}
	return nil
}

func DeleteUserbyRoot(id string) error {
	if id == "root" {
		return CanNotDeleteRoot
	}
	sqlStr := `delete from tbPassword where id = ?;`
	_, err := DB.Exec(sqlStr, id)
	return err
}
func ResetPwdByRoot(id string, newPwd string) error {
	hashedPSW, err := hashAndSalt(newPwd)
	if err != nil {
		return err
	}
	sqlStr := `update tbPassword set pwd = ? where id=?`
	_, err = DB.Exec(sqlStr, hashedPSW, id)
	return err
}

//func ResetPwd(id string,oldPwd string,newPwd string) error {
//
//	return nil
//}

func hashAndSalt(pwd string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.MinCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func comparePassword(hashedPWD string, pwd string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPWD), []byte(pwd))
	if err != nil {
		return false
	}
	return true
}
