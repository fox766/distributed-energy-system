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

func InitUserTable(db *sql.DB) error {
	var err error
	sqlStr := "DROP TABLE IF EXISTS users"
	_, err = db.Exec(sqlStr)
	if err != nil {
		return err
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS users (user_id VARCHAR(50) PRIMARY KEY, username VARCHAR(50) UNIQUE NOT NULL, passwordhash VARCHAR(255) NOT NULL)")
	if err != nil {
		return err
	}
	return nil 
}


func InitOrderTable(db *sql.DB) error {
	var err error
	sqlStr := "DROP TABLE IF EXISTS orders"
	_, err = db.Exec(sqlStr)
	if err != nil {
		return err
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS orders (order_id VARCHAR(50) PRIMARY KEY, partyA VARCHAR(50) NOT NULL, partyB VARCHAR(50) NOT NULL, status VARCHAR(50) NOT NULL, amount FLOAT)")
	if err != nil {
		return err
	}
	return nil 
}

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

	err = InitUserTable(db)
	if err != nil {
		return fmt.Errorf("failed to InitUserTable.")
	}

	err = InitOrderTable(db) 
	if err != nil {
		return fmt.Errorf("failed to InitUserTable.")
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

func UpdateOrderParty(orderid, partyB string) (err error) {
	var cnt int64
    sqlStr := "select count(order_id) from orders where order_id = ?"
    err = db.QueryRow(sqlStr, orderid).Scan(&cnt)
    if err != nil {
        return err
    }

    if cnt == 0 {
        return errors.New("order_id does not exist.")
    }

    sqlStr = "update orders set partyB = ? where order_id = ?"
    _, err = db.Exec(sqlStr, partyB, orderid)
    if err != nil {
        return err
    }
	
    return nil
}

func UpdateOrderStatus(orderid, newStatus string) (err error) {
    var cnt int64
    sqlStr := "select count(order_id) from orders where order_id = ?"
    err = db.QueryRow(sqlStr, orderid).Scan(&cnt)
    if err != nil {
        return err
    }

    if cnt == 0 {
        return errors.New("order_id does not exist.")
    }

    sqlStr = "update orders set status = ? where order_id = ?"
    _, err = db.Exec(sqlStr, newStatus, orderid)
    if err != nil {
        return err
    }
	
    return nil
}


func InsertOrder(orderid, partyA, partyB, status string, amount float64) (err error) {
	var cnt int64

	sqlStr := "select count(order_id) from orders where order_id = ?"
	err = db.QueryRow(sqlStr, orderid).Scan(&cnt)
	if err != nil {
		return err
	}

	if cnt > 0 {
		return errors.New("order_id already exists.")
	}

	sqlStr = "insert into orders(order_id,partyA,partyB,status,amount) values(?,?,?,?,?)"
	_, err = db.Exec(sqlStr, orderid, partyA, partyB, status, amount)
	if err != nil {
		return err
	}

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

func ReturnOrders(status, userid string) ([]MysqlOrder, error) {
    var orders []MysqlOrder
    var sqlStr string
    var args []interface{}
    
   // 开始构建 SQL 查询语句
   sqlStr = "SELECT order_id, partyA, partyB, status, amount FROM orders WHERE 1=1"

   // 根据 status 过滤
   if status != "ALL" {
	   sqlStr += " AND status = ?"
	   args = append(args, status)
   }

   // 根据 orderType 过滤
   if userid != "ALL" {
	   sqlStr += " AND (partyA = ? OR partyB = ?)"
	   args = append(args, userid, userid)
   }

    rows, err := db.Query(sqlStr, args...)
    if err != nil {
        return nil, fmt.Errorf("query failed: %w", err)
    }
    defer rows.Close()

    for rows.Next() {
        var o MysqlOrder
        if err := rows.Scan(&o.OrderID, &o.PartyA, &o.PartyB, &o.Status, &o.Amount); err != nil {
            return nil, fmt.Errorf("scan failed: %w", err)
        }
        orders = append(orders, o)
    }

    if err := rows.Err(); err != nil {
        return nil, fmt.Errorf("rows iteration error: %w", err)
    }
    return orders, nil
}

func ReturnOrderNum() (error, int){
	var count int
	var err error

	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return err, -1
	}
	return nil, count
}

func ReturnUserNum() (error, int){
	var count int
	var err error

	err = db.QueryRow("SELECT COUNT(*) FROM orders").Scan(&count)
	if err != nil {
		return err, -1
	}
	return nil, count
}