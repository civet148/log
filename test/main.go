package main

import (
	"github.com/civet148/log"
	"sync"
	"sync/atomic"
	"time"
)

type SubSubSt struct {
	Name string
}

type TestSubSt struct {
	SubInt int
	SubStr string
	sst    *SubSubSt
}

type testSt struct {
	MyPtr      *string
	MyInt      int
	MyFloat64  float64
	MyMap      map[string]string
	MyMapPtr   *map[string]string
	abc        int       //非导出字段(不处理会报panic错误)
	str        string    //非导出字段
	flt32      float32   //非导出字段
	flt64      float64   //非导出字段
	ui32       uint32    //非导出字段
	ui8        uint8     //非导出字段
	i8         int8      //非导出字段
	i64        int64     //非导出字段
	slice      []string  //非导出字段: 切片
	arr2       [5]byte   //非导出字段: 数组
	test       TestSubSt //非导出字段: 结构体
	ip         *int32    //非导出字段
	MySubSt    TestSubSt
	MySubStPtr *TestSubSt
}

var totalSeconds int64
var mutex sync.Mutex

func main() {

	log.Enter()
	defer log.Leave()

	log.Open("test.log", log.Option{
		LogLevel:   log.LEVEL_TRACE,
		FileSize:   1, //MB
		MaxBackups: 3,
	})
	defer log.Close()

	log.SetLevel("trace") //设置日志输出级别
	//log.SetFileSize(1) //设置最大单个文件大小(单位：MB)
	//log.SetMaxBackup(5) //最多保留备份日志文件数量

	for i := 0; i < 10000; i++ {
		log.Tracef("This is trace message")
		log.Debugf("This is debug message")
		log.Infof("This is info message")
		log.Warnf("This is warn message")
		log.Errorf("This is error message")
		log.Fatalf("This is fatal message")
		log.Truncate(log.LEVEL_INFO, 16, "this is a truncate message log [%s]", "hello")
		time.Sleep(50 * time.Millisecond)
	}

	//log.Panic("this function will call panic")

	//wg := &sync.WaitGroup{}
	//for i := 0; i < 1; i++ {
	//
	//	wg.Add(1)
	//	PrintFuncExecuteTime(i, wg)
	//	time.Sleep(5 * time.Millisecond)
	//}
	//
	//wg.Wait()
	//log.Leave()
	//
	////打印方法执行调用次数、总时间、平均时间和错误次数
	//log.Info("report summary: %v", log.Report())
	//log.Info("total seconds %v", totalSeconds)
	//log.Debug("This is debug message level = ", 0)
	//log.Info("This is info message level = ", 1)
	//log.Warn("This is warn message level = ", 2)
	//log.Error("This is error message level = ", 3)
	//log.Fatal("This is fatal message level = ", 4)
	//
	//os.Setenv("LOG_LEVEL", "info") //set log level to [info] by environment
	//
	//log.Debugw("This is debug message level = %w", 0, "Debugw")
	//log.Infow("This is info message level = ", 1, "Infow")
	//log.Warnw("This is warn message level = ", 2, "Warnw")
	//log.Errorw("This is error message level = ", 3, "Errorw")
	//log.Fatalw("This is fatal message level = ", 4, "Fatalw")
	log.StartProf("127.0.0.1:4000")
}

func PrintFuncExecuteTime(i int, wg *sync.WaitGroup) {

	log.Enter()
	defer log.Leave()
	beg := time.Now().Unix()
	var ip int32 = 10
	st1 := testSt{
		flt32:      0.58,
		flt64:      0.96666,
		ui8:        25,
		ui32:       10032,
		i8:         44,
		i64:        100000000000019,
		str:        "hello...",
		slice:      []string{"str1", "str2"},
		MyInt:      1,
		MyFloat64:  2.00,
		ip:         &ip,
		MySubSt:    TestSubSt{SubInt: 1, SubStr: "My sub str"},
		MySubStPtr: &TestSubSt{SubInt: 19, SubStr: "MySubStPtr", sst: &SubSubSt{Name: "I'm subsubst object"}}}

	st2 := &testSt{MyInt: 2, MyFloat64: 4.00, abc: 9999}
	log.Json(st1, st2)
	log.Struct(st1, st2)
	log.Debugf("[%d] %v", i, log.JsonDebugString(st1))
	elapse := time.Now().Unix() - beg
	atomic.AddInt64(&totalSeconds, elapse)
	wg.Done()
	//log.Errorf("done")
}
