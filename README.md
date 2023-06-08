# a colorful logging package

## Quick Start

```go
package main
import "github.com/civet148/log"

type Student struct {
    Age int `json:"age"`
    Name string `json:"name"`
}

func main() {
    log.SetLevel("trace") // set log level
    log.Tracef("This is trace message") //trace log
    log.Debugf("This is debug message") //debug log
    log.Infof("This is info message") //info log
    log.Warnf("This is warn message") //warn log
    log.Errorf("This is error message") //error log
    log.Fatalf("This is fatal message") //fatal log
    log.Truncate(log.LEVEL_INFO, 16, "this is a truncate message log [%s]", "hello") //truncate long message
	
    var student = &Student{
        Name:"lory",
        Age: 18
    }
    log.Json(student) //print student to json
}
```

## Open log file

```go
package main
import "github.com/civet148/log"
func main() {
    //write log to file test.log and set log level TRACE
    //the log file max size is 20MB and keeping 3 backups
    log.Open("test.log", log.Option{
        LogLevel:   log.LEVEL_TRACE,
        FileSize:   20, //MB
        MaxBackups: 3,
    })
    defer log.Close()
    for i := 0; i < 100000000; i++ {
        log.Tracef("This is trace message")
        log.Debugf("This is debug message")
        log.Infof("This is info message")
        log.Warnf("This is warn message")
        log.Errorf("This is error message")
        log.Fatalf("This is fatal message")
        log.Truncate(log.LEVEL_INFO, 16, "this is a truncate message log [%s]", "hello")
        time.Sleep(50 * time.Millisecond)
    }	
}
```

## Statistics

print function execute statistics 

```go
package main
import (
	"time"
	"github.com/civet148/log"
)
func main() {
    log.Enter() //start statistics
    defer log.Leave() //defer stop and print statistics
}
```


## Start pprof

```go
import (
	"time"
	"github.com/civet148/log"
)
func main() {
    log.StartProf("127.0.0.1:4000") //listen a http server and provider pprof debug information
}
```