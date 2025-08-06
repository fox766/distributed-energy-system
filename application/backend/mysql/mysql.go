package mysql

import (
	"fmt"
	"time"
	"database/sql"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
)

// password_hash, return a string
func PasswordHash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// password_verify, return boolean type
func PasswordVerify(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

var db *sql.DB

func Initmysql() (err error) {
	db_name := "fabric_mysql"
	dsn := "root:fabric@tcp(127.0.0.1:3337)/"
	//连接数据集
	db, err = sql.Open("mysql", dsn) //open不会检验用户名和密码
	if err != nil {
		fmt.Printf("dsn:%s invalid,err:%v\n", dsn, err)
		return err
	}
	err = db.Ping() //尝试连接数据库
	if err != nil {
		fmt.Printf("open %s faild,err:%v\n", dsn, err)
		return err
	}

	db.SetConnMaxLifetime(time.Minute)
	db.Exec("CREATE DATABASE IF NOT EXISTS " + db_name)
	db.Exec("USE " + db_name)

	// for test, every time create a new table
	sqlStr := "DROP TABLE IF EXISTS users"
	_, err = db.Exec(sqlStr)
	if err != nil {
		panic(err.Error())
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS users (user_id VARCHAR(50) PRIMARY KEY, username VARCHAR(50) UNIQUE NOT NULL, passwordhash VARCHAR(255) NOT NULL)")
	if err != nil {
		panic(err.Error())
	}

	dsn += db_name
	db, err = sql.Open("mysql", dsn) //open不会检验用户名和密码
	if err != nil {
		fmt.Printf("dsn:%s invalid,err:%v\n", dsn, err)
		return err
	}
	err = db.Ping() //尝试连接数据库
	if err != nil {
		fmt.Printf("open %s faild,err:%v\n", dsn, err)
		return err
	}
	fmt.Println("Successfully connect to the database!")

	return nil
}



// Insert User
func InsertUser(userid, username, password string) (err error) {
	var cnt int64
	var passwordhash string

	sqlStr := "select count(user_id) from users where username = ?"
	err = db.QueryRow(sqlStr, username).Scan(&cnt)
	if err != nil {
		return err
	}

	if cnt > 0 {
		return errors.New("username already exists.")
	}

	sqlStr = "insert into users(user_id,username,passwordhash) values(?,?,?)"
	passwordhash, err = PasswordHash(password)
	if err != nil {
		return err
	}

	_, err = db.Exec(sqlStr, userid, username, passwordhash)
	if err != nil {
		return err
	}

	return nil
}


// get userid
func GetUserID(username string) (userid string, err error) {
	sqlStr := "select user_id from users where username = ?"
	err = db.QueryRow(sqlStr, username).Scan(&userid)
	if err != nil {
		return "", err
	}
	return userid, nil
}

// get PasswordHash
func GetPasswordHash(username string) (passwordhash string, err error) {
	sqlStr := "select passwordhash from users where username = ?"
	err = db.QueryRow(sqlStr, username).Scan(&passwordhash)
	if err != nil {
		return "", err
	}
	return passwordhash, nil
}


// Verify the password
func CheckPassword(username, password string) (err error) {
	var passwordhash string

	passwordhash, err = GetPasswordHash(username)
	if err != nil {
		return err
	}

	if PasswordVerify(password, passwordhash) {
		return nil
	}

	return errors.New("password wrong.")
}