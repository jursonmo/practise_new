package main

// 自定义plugin 做法，可以在gorm的插件里面做一些事情，比如统计数据的插入、更新、删除的数量，耗时等。
// 从而实现metrics的收集。以及tracing. 参考beyond项目。
import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

//import "gorm.io/plugin/optimisticlock"

type User struct {
	gorm.Model
	// ID   int
	Name string
	Age  uint
	//Version optimisticlock.Version
}

// https://gorm.io/docs/hooks.html
func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	fmt.Printf("User object, before create, now:%v\n", time.Now())
	time.Sleep(100 * time.Millisecond)
	return
}

func (u *User) AfterCreate(tx *gorm.DB) (err error) {
	fmt.Printf("User object, after create, now:%v\n", time.Now())
	return
}
func (u *User) BeforeSave(tx *gorm.DB) (err error) {
	fmt.Println("User object, BeforeSave")
	return
}

func (u *User) AfterSave(tx *gorm.DB) (err error) {
	fmt.Println("User object, AfterSave")
	return
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
	// db.Callback().Create().Before("mjw:gorm:create").Register("custome_plugin:create", func(db *gorm.DB) {
	// 	fmt.Println("custome_plugin:before_create")
	// 	now := time.Now()
	// 	db.InstanceSet("gorm:create_at", now)
	// })
	//mjw:gorm:create 这个名称不可以随便取。有固定的名称，这样在create 操作时, gorm内部才能找到对应的方法。否则执行的顺序是就不按正常的来了。
	//https://gorm.io/docs/write_plugins.html#Registering-a-Plugin  参考官方文档定义的名称。
	if err := db.Callback().Create().Before("gorm:create").Register("custome_plugin:create", func(db *gorm.DB) {
		fmt.Println("custome_plugin:before_create")
		now := time.Now()
		db.InstanceSet("gorm:create_at", now)
	}); err != nil {
		fmt.Println("register error:", err)
		panic(err)
	}

	// // beyond 项目中，gorm的插件是这样定义的，"gorm:createBefore",实际效果不行，也是在“User object, AfterSave”之后才执行输出的。
	// if err := db.Callback().Create().Before("gorm:createBefore").Register("custome_plugin:createBefore", func(db *gorm.DB) {
	// 	fmt.Println("custome_plugin:before_createBefore")
	// }); err != nil {
	// 	fmt.Println("register error:", err)
	// 	panic(err)
	// }

	db.Callback().Create().After("gorm:create").Register("custome_plugin:after_create", func(db *gorm.DB) {
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

	//如果表不存在，会自动创建user表
	db.AutoMigrate(&User{})
	//插入数据
	user := User{Name: "Jinzhu", Age: 18}
	fmt.Println("create user now")
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
代码是 db.Callback().Create().Before("mjw:gorm:create")
output结果：
CustomePlugin initialize
create user now
User object, BeforeSave
User object, before create, now:2024-12-02 11:52:11.807078 +0800 CST m=+0.034576251
User object, after create, now:2024-12-02 11:52:11.910098 +0800 CST m=+0.137598626
User object, AfterSave
custome_plugin:before_create
custome_plugin:after_create
6µs since create_at:2024-12-02 11:52:11.91252 +0800 CST m=+0.140021043
user id:10, result.RowsAffected:1

会有问题： “custome_plugin:before_create” 怎么在“User object, AfterSave” 之后执行了？
原因是db.Callback().Create().Before("mjw:gorm:create")  "mjw:gorm:create"这个名称不可以随便取。
这是有固定的名称的:"gorm:create"，这样在create 操作时, gorm内部才能找到对应的方法。否则执行的顺序是就不按正常的来了。
*/

/*
把代码改回 db.Callback().Create().Before("gorm:create")
输出结果：
CustomePlugin initialize
create user now
User object, BeforeSave
User object, before create, now:2024-12-02 12:15:57.550728 +0800 CST m=+0.136660251
custome_plugin:before_create
User object, after create, now:2024-12-02 12:15:57.660559 +0800 CST m=+0.246490751
User object, AfterSave
custome_plugin:after_create
13.725708ms since create_at:2024-12-02 12:15:57.651883 +0800 CST m=+0.237815293
user id:12, result.RowsAffected:1

这样就正常了,  “custome_plugin:before_create” 在“User object, AfterSave” 之前执行了。
*/
