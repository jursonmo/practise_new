package main

// 自定义plugin 做法，可以在gorm的插件里面做一些事情，比如统计数据的插入、更新、删除的数量，耗时等。
// 从而实现metrics的收集。以及tracing.
import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

//import "gorm.io/plugin/optimisticlock"

type User struct {
	//gorm.Model
	ID   int
	Name string
	Age  uint
	//Version optimisticlock.Version
}

type CustomePlugin struct{}

func NewCustomePlugin() *CustomePlugin {
	return &CustomePlugin{}
}

func (p *CustomePlugin) Name() string {
	return "CustomePlugin"
}

func (p *CustomePlugin) Initialize(db *gorm.DB) error {
	fmt.Println("CustomePlugin initialize")
	db.Callback().Create().Before("mjw:gorm:create").Register("custome_plugin:create", func(db *gorm.DB) {
		fmt.Println("custome_plugin:create")
		now := time.Now()
		db.InstanceSet("gorm:create_at", now)
		time.Sleep(100 * time.Millisecond)
	})
	//mjw:gorm:create 这个名称可以随便取。
	db.Callback().Create().After("mjw:gorm:create").Register("custome_plugin:after_create", func(db *gorm.DB) {
		fmt.Println("custome_plugin:after_create")
		v, ok := db.InstanceGet("gorm:create_at")
		if ok {
			fmt.Printf("%v since create_at:%v \n", time.Since(v.(time.Time)), v)
		}
	})
	return nil
}

type Config struct {
	DSN          string
	MaxOpenConns int
	MaxIdleConns int
	MaxLifetime  int
}

type DB struct {
	*gorm.DB
}

func NewMysql(conf *Config) (*DB, error) {
	if conf.MaxIdleConns == 0 {
		conf.MaxIdleConns = 10
	}
	if conf.MaxOpenConns == 0 {
		conf.MaxOpenConns = 100
	}
	if conf.MaxLifetime == 0 {
		conf.MaxLifetime = 3600
	}
	db, err := gorm.Open(mysql.Open(conf.DSN), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sdb, err := db.DB()
	if err != nil {
		return nil, err
	}
	sdb.SetMaxIdleConns(conf.MaxIdleConns)
	sdb.SetMaxOpenConns(conf.MaxOpenConns)
	sdb.SetConnMaxLifetime(time.Second * time.Duration(conf.MaxLifetime))

	err = db.Use(NewCustomePlugin())
	if err != nil {
		return nil, err
	}

	return &DB{DB: db}, nil
}

func main() {
	conf := &Config{
		DSN: "root:@tcp(127.0.0.1:3306)/gorm_plugin_test?parseTime=true&loc=Local",
	}
	db, err := NewMysql(conf)
	if err != nil {
		panic(err)
	}

	//创建user表
	db.AutoMigrate(&User{})
	//插入数据
	user := User{Name: "Jinzhu", Age: 18}
	result := db.Create(&user) // pass pointer of data to Create
	if result.Error != nil {
		panic(result.Error)
	}
	fmt.Printf("user id:%d, result.RowsAffected:%d\n", user.ID, result.RowsAffected)

	// //插入多条数据
	// users := []*User{
	// 	{Name: "Jinzhu", Age: 18},
	// 	{Name: "Jackson", Age: 19},
	// }
	// result = db.Create(users) // pass a slice to insert multiple row
	// if result.Error != nil {
	// 	panic(result.Error)
	// }
	// fmt.Printf("result.RowsAffected:%d\n", result.RowsAffected) //result.RowsAffected:2

	return
}

/*
CustomePlugin initialize
custome_plugin:create
custome_plugin:after_create
101.036958ms since create_at:2024-12-02 00:49:40.495748 +0800 CST m=+0.021301876
user id:5, result.RowsAffected:1
*/
