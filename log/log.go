/**
* A log package for golang
* 
* not instead of golang's log but a replenish
*/

package log

import (
	"sync"
	"fmt"
	"path/filepath"
	"errors"
	"runtime"
	"time"
)

type Level int   // loose enum type . maybe have some other define method
const (
	// content of emnu Level ,level of log
	LNULL   =  iota
	LDEBUG
	LINFO
	LWARNING
	LERROR
	LFATAL
)

const (
	MAX_LOG_FILE_SIZE = 5*1024*1024 // Default max log file size is 500M
)

type Outputer int 
const (
	OUT_STD   = 1<<iota
	OUT_FILE
)

var EOutput error = errors.New("Output is invalied!")

type Logger struct {
	level      Level
	logDevice  *FileDevice
	errDevice  *FileDevice
	conDevice  *ConsoleDevice
	outputer   Outputer

	mtx        sync.Mutex
	buf        []byte
	callDepth  int
}

var _logger *Logger

func init(){
	var err error
	_logger, err = NewConsoleLogger()
	if err != nil {
		//TODO:
	}
	_logger.SetCallDepth(3)
}

func NewConsoleLogger() (*Logger, error) {
	var err error
	logger := &Logger{level:LDEBUG, outputer:OUT_STD, callDepth:3}
	logger.conDevice,err = NewConsoleDevice()
	return logger,err
}

func NewFileLogger(logPath, fileName string)( *Logger,error){
	var err error
	logger := &Logger{level:LDEBUG, outputer:OUT_FILE, callDepth:2}
	logFileName := filepath.Join(logPath, fileName+".log")
	logger.logDevice, err = NewFileDevice(logFileName)
	if err != nil {
		return nil, err
	}

	errFileName := filepath.Join(logPath, fileName+".error")
	logger.errDevice, err = NewFileDevice(errFileName)
	if err != nil {
		return nil,err
	}

	return logger,nil
}

func (l *Logger) SetCallDepth(d int){
	l.callDepth = d
}

func (l *Logger) getFileLine() string{
	_, file, line, ok := runtime.Caller(l.callDepth)
	if !ok {
		file = "???"
		line = 0
	}
	
	return file+":"+itoa(line,-1)
}

/**
* Change from Golang's log.go
* Cheap integer to fixed-width decimal ASCII.  Give a negative width to avoid zero-padding.
* Knows the buffer has capacity.
*/
func itoa(i int, wid int) string {
	var u uint = uint(i)
    if u == 0 && wid <= 1 {
		return "0"
	}

    // Assemble decimal in reverse order.
    var b [32]byte
    bp := len(b)
    for ; u > 0 || wid > 0; u /= 10 {
		bp--
		wid--
		b[bp] = byte(u%10) + '0'
	}
	return string(b[bp:])
}

func (l *Logger) getTime() string {
	// Time is yyyy-mm-dd hh:mm:ss.microsec
	var buf  []byte
	t := time.Now()
	year, month, day := t.Date()
	buf = append(buf, itoa(int(year),4)+"-" ...)
	buf = append(buf,itoa(int(month),2)+ "-" ...)
	buf = append(buf, itoa(int(day),2)+" " ...)
	
	hour, min, sec := t.Clock()
	buf = append(buf,itoa(hour,2)+ ":" ...)
	buf = append(buf,itoa(min,2)+ ":" ...) 
	buf = append(buf,itoa(sec,2) ...)

	buf = append(buf, '.')
	buf = append(buf,itoa(t.Nanosecond()/1e3,6) ...)

	return string(buf[:])
}

func (l *Logger) output(level Level, prefix string,format string,v... interface{}) (err error) {
	var levelStr string 
	if level == LDEBUG {
		levelStr = "[DEBUG]"
	} else if level == LINFO {
		levelStr = "[INFO]"
	} else if level == LWARNING {
		levelStr = "[WARNING]"
	} else if level == LERROR {
		levelStr = "[ERROR]"
	} else if level == LFATAL {
		levelStr = "[FATAL]"
	} else {
		levelStr = "[UNKNOWN LEVEL]"
	}

	var msg string
	if format== ""  {
		msg = fmt.Sprintln(v...)
	} else {
		msg = fmt.Sprintf(format,v...)
	}

	l.mtx.Lock()
	defer l.mtx.Unlock()
	l.buf = l.buf[:0]
	
//	l.buf = append(l.buf,"["+l.logName+"]" ...)
	l.buf = append(l.buf,levelStr ...)
	l.buf = append(l.buf,prefix ...)

	l.buf = append(l.buf,":"+msg ... )
	if len(msg)>0 && msg[len(msg)-1]!= '\n'{
		l.buf = append(l.buf,'\n')
	}

	if l.outputer == OUT_STD {
		_,err = l.conDevice.Write(l.buf)
	} else if l.outputer == OUT_FILE {
		if level <= LWARNING {
			_,err = l.logDevice.Write(l.buf)
		} else {
			_,err = l.errDevice.Write(l.buf)
		} 
	} else {
		err = EOutput
	}
	return 
}

func (l *Logger) SetMaxFileSize(fileSize uint64) {
	l.logDevice.SetFileSize(fileSize)
	l.errDevice.SetFileSize(fileSize)
}

func (l *Logger) SetLevel (level Level) {
	l.level = level
}

/** Nothing to change */

func (l *Logger) Debug(format string,v... interface{}) error {
	if l.level > LDEBUG {
		return nil
	}

	err := l.output(LDEBUG, "["+l.getTime()+"]["+l.getFileLine()+"]", format, v...)
	return err
}

func (l *Logger) Info(format string,v...interface{}) error{
	if l.level > LINFO {
		return nil
	}

	err := l.output(LINFO,"["+l.getTime()+"]["+l.getFileLine()+"]", format, v...)
	return err
}

func (l *Logger) Warning(format string,v...interface{}) error{
	if l.level > LWARNING {
		return nil
	}
	err := l.output(LWARNING,"["+l.getTime()+"]["+l.getFileLine()+"]", format, v...)
	return err
}

func (l *Logger) Error(format string,v...interface{}) error{
	if l.level > LERROR {
		return nil
	}
	err := l.output(LERROR,"["+l.getTime()+"]["+l.getFileLine()+"]", format, v...)
	return err
}

func (l *Logger) Fatal(format string,v... interface{}) error{
	if l.level > LFATAL{
		return nil
	}
	
	err := l.output(LFATAL,"["+l.getTime()+"]["+l.getFileLine()+"]", format, v...)
	return err
}

func DEBUG(format string,v... interface{}) error{
	return _logger.Debug(format,v...)
}

func INFO(format string,v... interface{}) error{
	return _logger.Info(format,v...)
}

func WARNING(format string,v... interface{}) error{
	return _logger.Warning(format,v...)
}

func ERROR(format string,v... interface{}) error{
	return _logger.Error(format,v...)
}

func FATAL(format string,v... interface{}) error{
	return _logger.Fatal(format,v...)
}

