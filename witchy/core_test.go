package witchy

import (
	"context"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormLogger "gorm.io/gorm/logger"
	"testing"
)

var logger gormLogger.Interface
var db *gorm.DB
var dsn = "root:123456@tcp(127.0.0.1:3306)/my_gorm?charset=utf8mb4&parseTime=True&loc=Local"

type Student struct {
	Age int
	Id  int64
}

func TestMain(m *testing.M) {
	log, _ := zap.NewDevelopment(zap.AddCaller(), zap.AddCallerSkip(2))

	logger = NewPaper(log, gormLogger.Config{LogLevel: gormLogger.Info})
	var err error
	db, err = gorm.Open(mysql.New(mysql.Config{
		DSN: dsn,
	}), &gorm.Config{
		Logger: logger,
	})
	if err != nil {
		panic(err)
	}
	db.WithContext(context.TODO()).AutoMigrate(&Student{})
	m.Run()

}
