package log

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/color"
	"github.com/mattn/go-colorable"
	"log"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var colorStdout = colorable.NewColorableStdout()

var LevelName = []string{"[TRACE]", "[DEBUG]", "[INFO]", "[WARN]", "[ERROR]", "[FATAL]", "[PANIC]", "[JSON]"}

const (
	DefaultLogSize    = 1024 //MB
	DefaultMaxBackups = 31
)

const (
	ENV_LOG_LEVEL = "LOG_LEVEL"
)

const (
	LEVEL_TRACE = 0
	LEVEL_DEBUG = 1
	LEVEL_INFO  = 2
	LEVEL_WARN  = 3
	LEVEL_ERROR = 4
	LEVEL_FATAL = 5
	LEVEL_PANIC = 6
	LEVEL_JSON  = 7
)

type logInfo struct {
	locker  sync.RWMutex
	logFile *os.File    //日志文件对象
	logger  *log.Logger //日志输出对象
}

type Option struct {
	LogLevel     int    //文件日志输出级别
	FileSize     int    //文件日志分割大小(MB)
	MaxBackups   int    //文件最大分割数
	CloseConsole bool   //开启/关闭终端屏幕输出
	filePath     string //文件日志路径
}

// 全局变量
var (
	loginf logInfo //日志信息对象
	option = Option{
		LogLevel:   LEVEL_INFO,
		FileSize:   DefaultLogSize,
		MaxBackups: DefaultMaxBackups,
	} //日志参数选项
)

func init() {
}

func EnableStats(enable bool) {
	enableStats = enable
}

func Open(strPath string, opts ...Option) error {
	if strPath == "" {
		return Error("log file is nil")
	}
	err := loginf.open(strPath, opts...)
	if err != nil {
		return Error("%s", err)
	}
	go backupLogFile()
	return nil
}

// 关闭日志
func Close() {
	err := loginf.closeFile()
	if err != nil {
		Error("%s", err)
		return
	}
}

// 设置日志文件分割大小（MB)
func SetFileSize(size int) {
	option.FileSize = size
}

// 设置日志级别(字符串型: trace/debug/info/warn/error/fatal 数值型: 0=TRACE 1 =DEBUG 2=INFO 3=WARN 4=ERROR 5=FATAL)
func SetLevel(level interface{}) {

	var nLevel int
	switch level.(type) {
	case string:
		strLevel := strings.ToLower(level.(string))
		switch strLevel {
		case "trace":
			nLevel = LEVEL_TRACE
		case "debug":
			nLevel = LEVEL_DEBUG
		case "info":
			nLevel = LEVEL_INFO
		case "warn", "warning":
			nLevel = LEVEL_WARN
		case "error":
			nLevel = LEVEL_ERROR
		case "fatal":
			nLevel = LEVEL_FATAL
		}
	case int8, int16, int, int32, int64, uint8, uint16, uint, uint32, uint64:
		nLevel, _ = strconv.Atoi(fmt.Sprintf("%v", level))
	default:
		nLevel = LEVEL_INFO
	}
	option.LogLevel = nLevel
}

// 设置关闭/开启屏幕输出
func CloseConsole(ok bool) {
	option.CloseConsole = ok
}

// 设置最大备份数量
func SetMaxBackup(nMaxBackups int) {
	option.MaxBackups = nMaxBackups
}

// 定期清理日志，仅保留MaxBackups个数的日志
func backupLogFile() {
	for {
		_ = loginf.renameFile()
		_ = loginf.cleanBackupLog()
		time.Sleep(30 * time.Second)
	}
}

func (m *logInfo) Println(args ...interface{}) {
	if loginf.logger == nil {
		return
	}
	m.locker.Lock()
	defer m.locker.Unlock()
	loginf.logger.Println(args...)
}

func (m *logInfo) open(strPath string, opts ...Option) (err error) {
	if len(opts) > 0 {
		option = opts[0]
	}
	option.filePath = strPath
	if option.FileSize == 0 {
		option.FileSize = DefaultLogSize
	}
	if option.MaxBackups == 0 {
		option.MaxBackups = DefaultMaxBackups
	}
	return m.createFile() //创建文件
}

// 关闭日志文件
func (m *logInfo) closeFile() error {
	m.locker.Lock()
	defer m.locker.Unlock()
	if loginf.logFile == nil {
		return nil
	}
	_ = loginf.logFile.Close()
	loginf.logFile = nil
	return nil
}

// 清理已过期备份
func (m *logInfo) cleanBackupLog() error {
	var files []os.FileInfo
	strPath := option.filePath
	if strPath == "" {
		return nil
	}
	dir, filename := filepath.Split(strPath)
	if dir == "" {
		dir = "."
	}
	match := filename + "."
	filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if info.IsDir() {
				return nil
			}
			if strings.Index(info.Name(), match) != -1 {
				files = append(files, info)
			}
			return nil
		})
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime().Unix() > files[j].ModTime().Unix()
	})
	count := len(files)
	if count > option.MaxBackups && option.MaxBackups != 0 {
		for i := option.MaxBackups - 1; i < count; i++ {
			strFilePath := filepath.Join(dir, files[i].Name())
			_ = os.Remove(strFilePath)
		}
	}
	return nil
}

// 关闭日志文件
func (m *logInfo) renameFile() (err error) {
	if loginf.logFile == nil {
		return nil
	}
	fi, err := os.Stat(option.filePath)
	if err != nil {
		return err
	}
	fs := fi.Size()
	renameSize := option.FileSize * 1024 * 1024

	m.locker.Lock()
	defer m.locker.Unlock()

	if fs > int64(renameSize) {
		_ = loginf.logFile.Close()
		datetime := time.Now().Format("20060102150405")
		var strPath string
		strPath = fmt.Sprintf("%v.%v", option.filePath, datetime) //日志文件有后缀(日志备份文件名格式不能随意改动)
		err = os.Rename(option.filePath, strPath)                 //将文件备份
		if err != nil {
			return err
		}
		loginf.logFile, err = os.OpenFile(option.filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		loginf.logger = log.New(loginf.logFile, "", log.Lmicroseconds|log.LstdFlags)
	}
	return nil
}

// 创建日志文件
func (m *logInfo) createFile() error {
	var err error
	m.locker.Lock()
	defer m.locker.Unlock()
	loginf.logFile, err = os.OpenFile(option.filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return Error("Open log file ", option.filePath, " failed ", err)
	}
	loginf.logger = log.New(loginf.logFile, "", log.Lmicroseconds|log.LstdFlags)
	return nil
}

// 截取函数名称
func getFuncName(pc uintptr) (name string) {

	n := runtime.FuncForPC(pc).Name()
	ns := strings.Split(n, ".")
	name = ns[len(ns)-1]
	return
}

// 通过级别名称获取索引
func getLevel(name string) (idx int) {

	name = "[" + name + "]"
	switch name {
	case LevelName[LEVEL_TRACE]:
		idx = LEVEL_TRACE
	case LevelName[LEVEL_DEBUG]:
		idx = LEVEL_DEBUG
	case LevelName[LEVEL_INFO]:
		idx = LEVEL_INFO
	case LevelName[LEVEL_WARN]:
		idx = LEVEL_WARN
	case LevelName[LEVEL_ERROR]:
		idx = LEVEL_ERROR
	case LevelName[LEVEL_FATAL]:
		idx = LEVEL_FATAL
	case LevelName[LEVEL_PANIC]:
		idx = LEVEL_PANIC
	default:
		idx = LEVEL_INFO
	}
	return
}

func getCaller(skip int) (strFile, strFunc string, nLineNo int) {
	pc, file, line, ok := runtime.Caller(skip)
	if ok {
		strFile = path.Base(file)
		nLineNo = line
		strFunc = getFuncName(pc)
	}
	return
}

func getStack(skip, n int) string {
	var strStack string
	strStack += "\t###CALLSTACK### { "
	for i := 0; i < n; i++ {
		pc, file, line, ok := runtime.Caller(skip + i)
		if ok {
			strFile := path.Base(file)
			nLineNo := line
			strFunc := getFuncName(pc)
			strStack += fmt.Sprintf("%s:%d %s(); ", strFile, nLineNo, strFunc)
		}
	}
	strStack += "}"
	strStack = color.CyanString(strStack)
	return strStack
}

// 内部格式化输出函数
func output(level int, fmtstr string, args ...interface{}) (strFile, strFunc string, nLineNo int) {
	var inf, code string
	var colorTimeName string

	strTimeFmt := fmt.Sprintf("%v", time.Now().Format("2006-01-02 15:04:05.000000"))
	strRoutine := fmt.Sprintf("{%v}", getRoutineId())
	strPID := fmt.Sprintf("PID:%d", os.Getpid())
	Name := LevelName[level]
	switch level {
	case LEVEL_TRACE:
		colorTimeName = fmt.Sprintf("\033[38m%v %s %s", strTimeFmt, strPID, Name)
	case LEVEL_DEBUG:
		colorTimeName = fmt.Sprintf("\033[34m%v %s %s", strTimeFmt, strPID, Name)
	case LEVEL_INFO:
		colorTimeName = fmt.Sprintf("\033[32m%v %s %s", strTimeFmt, strPID, Name)
	case LEVEL_WARN:
		colorTimeName = fmt.Sprintf("\033[33m%v %s %s", strTimeFmt, strPID, Name)
	case LEVEL_ERROR:
		colorTimeName = fmt.Sprintf("\033[31m%v %s %s", strTimeFmt, strPID, Name)
	case LEVEL_FATAL:
		colorTimeName = fmt.Sprintf("\033[35m%v %s %s", strTimeFmt, strPID, Name)
	case LEVEL_PANIC:
		colorTimeName = fmt.Sprintf("\033[35m%v %s %s", strTimeFmt, strPID, Name)
	default:
		colorTimeName = fmt.Sprintf("\033[34m%v %s %s", strTimeFmt, strPID, Name)
	}

	if fmtstr != "" {
		inf = fmt.Sprintf(fmtstr, args...)
	} else {
		inf = fmt.Sprint(args...)
	}

	strFile, strFunc, nLineNo = getCaller(3)
	code = "<" + strFile + ":" + strconv.Itoa(nLineNo) + " " + strFunc + "()" + ">"
	if level < option.LogLevel {
		return
	}

	var outstr string

	switch runtime.GOOS {
	//case "windows": //Windows终端（无颜色）
	//outstr = strTimeFmt + " " + Name + " " + strRoutine + " " + code + " " + inf
	default: //Unix类终端支持颜色显示
		outstr = "\033[1m" + colorTimeName + " " + strRoutine + " " + code + "\033[0m " + inf
	}

	if level >= LEVEL_ERROR && level != LEVEL_JSON {
		outstr += getStack(3, 10)
	}
	//打印到终端屏幕
	if !option.CloseConsole {
		_, _ = fmt.Fprintln(colorStdout /*os.Stdout*/, outstr)
	}

	//输出到文件（如果Open函数传入了正确的文件路径）
	loginf.Println(Name + " " + strRoutine + " " + code + " " + inf)

	return
}

func fmtString(args ...interface{}) (strOut string) {
	if len(args) > 0 {
		switch args[0].(type) {
		case string:
			if strings.Contains(args[0].(string), "%") {
				strOut = fmt.Sprintf(args[0].(string), args[1:]...)
			} else {
				strOut = fmt.Sprint(args...)
			}
		default:
			strOut = fmt.Sprint(args...)
		}
	}
	return
}

func fmtStringW(args ...interface{}) (strOut string) {
	var strArgs []string
	for _, v := range args {
		strArgs = append(strArgs, fmt.Sprintf("%v", v))
	}
	return strings.Join(strArgs, " ")
}

func Printf(args ...interface{}) {
	strPrint := fmtString(args...)
	fmt.Println(strPrint)
}

// 输出调试级别信息
func Trace(args ...interface{}) {
	output(LEVEL_TRACE, fmtString(args...))
}

// 输出调试级别信息
func Debug(args ...interface{}) {
	output(LEVEL_DEBUG, fmtString(args...))
}

// 输出运行级别信息
func Info(args ...interface{}) {
	output(LEVEL_INFO, fmtString(args...))
}

// 输出警告级别信息
func Warn(args ...interface{}) {
	output(LEVEL_WARN, fmtString(args...))
}

// 输出警告级别信息
func Warning(args ...interface{}) {
	output(LEVEL_WARN, fmtString(args...))
}

// 输出错误级别信息
func Error(args ...interface{}) error {
	err := fmt.Errorf(fmtString(args...))
	stic.error(output(LEVEL_ERROR, err.Error()))
	return err
}

// 输出危险级别信息
func Fatal(args ...interface{}) error {
	err := fmt.Errorf(fmtString(args...))
	stic.error(output(LEVEL_FATAL, err.Error()))
	return err
}

// panic
func Panic(args ...interface{}) {
	panic(fmt.Sprintf(fmtString(args...)))
}

// 输出调试级别信息
func Tracef(fmtstr string, args ...interface{}) {
	output(LEVEL_TRACE, fmtstr, args...)
}

// 输出调试级别信息
func Debugf(fmtstr string, args ...interface{}) {
	output(LEVEL_DEBUG, fmtstr, args...)
}

// 输出运行级别信息
func Infof(fmtstr string, args ...interface{}) {
	output(LEVEL_INFO, fmtstr, args...)
}

// 输出警告级别信息
func Warnf(fmtstr string, args ...interface{}) {
	output(LEVEL_WARN, fmtstr, args...)
}

// 输出警告级别信息
func Warningf(fmtstr string, args ...interface{}) {
	output(LEVEL_WARN, fmtstr, args...)
}

// 输出错误级别信息
func Errorf(fmtstr string, args ...interface{}) error {
	err := fmt.Errorf(fmtstr, args...)
	stic.error(output(LEVEL_ERROR, err.Error()))
	return err
}

// 输出危险级别信息
func Fatalf(fmtstr string, args ...interface{}) error {
	err := fmt.Errorf(fmtstr, args...)
	stic.error(output(LEVEL_FATAL, err.Error()))
	return err
}

// 输出Trace级别信息
func Tracew(args ...interface{}) {
	output(LEVEL_DEBUG, fmtStringW(args...))
}

// 输出调试级别信息
func Debugw(args ...interface{}) {
	output(LEVEL_DEBUG, fmtStringW(args...))
}

// 输出运行级别信息
func Infow(args ...interface{}) {
	output(LEVEL_INFO, fmtStringW(args...))
}

// 输出警告级别信息
func Warnw(args ...interface{}) {
	output(LEVEL_WARN, fmtStringW(args...))
}

// 输出警告级别信息
func Warningw(args ...interface{}) {
	output(LEVEL_WARN, fmtStringW(args...))
}

// 输出错误级别信息
func Errorw(args ...interface{}) {
	stic.error(output(LEVEL_ERROR, fmtStringW(args...)))
}

// 输出危险级别信息
func Fatalw(args ...interface{}) {
	stic.error(output(LEVEL_FATAL, fmtStringW(args...)))
}

// panic
func Panicw(args ...interface{}) {
	panic(fmt.Sprintf(fmtStringW(args...)))
}

// 输出到空设备
func Null(fmtstr string, args ...interface{}) {
}

func Truncate(level, size int, fmtstr string, args ...interface{}) {
	strOutput := fmt.Sprintf(fmtstr, args...)
	if len(strOutput) > size {
		strOutput = strOutput[:size] + "..."
	}
	output(level, strOutput)
}

// 进入方法（统计）
func Enter(args ...interface{}) {
	output(LEVEL_INFO, "enter ", args...)
	stic.enter(getCaller(2))
}

// 离开方法（统计）
// 返回执行时间：h 时 m 分 s 秒 ms 毫秒 （必须先调用Enter方法才能正确统计执行时间）
func Leave() (h, m, s int, ms float32) {

	if nSpendTime, ok := stic.leave(getCaller(2)); ok {
		h, m, s, ms = getSpendTime(nSpendTime)
		output(LEVEL_INFO, "leave (%vh %vm %vs %.3fms)", h, m, s, ms)
	}
	return
}

// 打印结构体JSON
func Json(args ...interface{}) {

	var strOutput string

	for _, v := range args {

		data, _ := json.MarshalIndent(v, "", "\t")
		strOutput += "\n...................................................\n" + string(data)
	}

	output(LEVEL_JSON, strOutput+"\n...................................................\n")
}

func JsonDebugString(v interface{}) string {
	data, _ := json.MarshalIndent(v, "", "\t")
	return string(data)
}

// args: a string of function name or nil for all
func Report(args ...interface{}) string {
	return stic.report(args...)
}

// 打印结构体
func Struct(args ...interface{}) {

	var strLog string
	for i := range args {
		arg := args[i]
		typ := reflect.TypeOf(arg)
		val := reflect.ValueOf(arg)
		if typ.Kind() == reflect.Ptr { //如果是指针类型则先转为对象

			typ = typ.Elem()
			val = val.Elem()
		}

		var nDeep int
		switch typ.Kind() {

		case reflect.Struct:
			strLog = fmtStruct(nDeep, typ, val) //遍历结构体成员标签和值存到map[string]string中
		case reflect.String:
			strLog += fmt.Sprintf("%v (string) = \"%+v\" \n", typ.Name(), val.Interface())
		default:
			strLog += fmt.Sprintf("%v (%v) = <%+v> \n", typ.Name(), typ.Kind(), val.Interface())
		}

		output(LEVEL_DEBUG, strLog)
	}
}

// 将字段值存到其他类型的变量中
func fmtStruct(deep int, typ reflect.Type, val reflect.Value, args ...interface{}) (strLog string) {

	kind := typ.Kind()
	nCurDeep := deep

	var bPointer bool
	var strParentName string
	if len(args) > 0 {
		bPointer = args[0].(bool)
		strParentName = args[1].(string)
	}

	if !val.IsValid() {
		if bPointer { //this variant is a struct pointer
			strLog = fmt.Sprintf("%v%v (*%v) = <nil>\n", fmtDeep(deep), strParentName, typ.String())
		} else {
			strLog = fmt.Sprintf("%v%v (%v) = <nil>\n", fmtDeep(deep), strParentName, typ.String())
		}
		return
	}

	if bPointer { //this variant is a struct pointer
		//strLog = fmt.Sprintf("%v%v (*%v) {\n", fmtDeep(deep) , typ.Kind().String(), typ.String())
		strLog = fmt.Sprintf("%v%v (*%v) {\n", fmtDeep(deep), strParentName, typ.String())
	} else {
		strLog = fmt.Sprintf("%v%v (%v) {\n", fmtDeep(deep), strParentName, typ.String())
	}

	if kind == reflect.Struct {
		deep++
		NumField := val.NumField()
		for i := 0; i < NumField; i++ {

			var isPointer bool
			typField := typ.Field(i)
			valField := val.Field(i)
			if typField.Type.Kind() == reflect.Ptr { //如果是指针类型则先转为对象

				typField.Type = typField.Type.Elem()
				valField = valField.Elem()
				isPointer = true
			}

			if typField.Type.Kind() == reflect.Struct {

				strLog += fmtStruct(deep, typField.Type, valField, isPointer, typField.Name) //结构体需要递归调用
			} else {
				//var strLog string
				if !valField.IsValid() { //字段为空指针
					strLog += fmtDeep(deep) + fmt.Sprintf("%v (%v) = <nil> \n", typField.Name, typField.Type)
				} else if !valField.CanInterface() { //非导出字段
					strLog += fmtDeep(deep) + fmt.Sprintf("%v (%v) = <%+v> \n", typField.Name, typField.Type, valField)
				} else {

					switch typField.Type.Kind() {
					case reflect.String:
						strLog += fmtDeep(deep) + fmt.Sprintf("%v (%v) = \"%+v\" \n", typField.Name, typField.Type, valField.Interface())
					default:
						strLog += fmtDeep(deep) + fmt.Sprintf("%v (%v) = <%+v> \n", typField.Name, typField.Type, valField.Interface())
					}
				}
			}
		}
	}
	strLog += fmtDeep(nCurDeep) + "}\n"
	return
}

func fmtDeep(nDeep int) (s string) {

	for i := 0; i < nDeep; i++ {
		s += fmt.Sprintf("... ")
	}
	return
}
