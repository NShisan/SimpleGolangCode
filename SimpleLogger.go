package utils

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

var (
	Logger   *Mylogs
	MlColors = MylogColors{
		Reset:    "\033[0m",
		Red:      "\033[31m",
		Green:    "\033[32m",
		Yellow:   "\033[33m",
		Blue:     "\033[34m",
		Magenta:  "\033[35m",
		Cyan:     "\033[36m",
		White:    "\033[37m",
		BlackBG:  "\033[40m",
		RedBG:    "\033[41m",
		GreenBG:  "\033[42m",
		YellowBG: "\033[43m",
		BlueBG:   "\033[44m",
		PurpleBG: "\033[45m",
		CyanBG:   "\033[46m",
		WhiteBG:  "\033[47m",
	}
)

type MylogColors struct {
	Reset    string
	Red      string
	Green    string
	Yellow   string
	Blue     string
	Magenta  string
	Cyan     string
	White    string
	BlackBG  string
	RedBG    string
	GreenBG  string
	YellowBG string
	BlueBG   string
	PurpleBG string
	CyanBG   string
	WhiteBG  string
}

type MylogsConfigSaveFile struct {
	FilePath    string //日志文件绝对路径
	SavePath    string //日志保存目录
	MaxSize     int64  //日志文件最大大小，单位字节B
	ClearOrBack rune   //日志文件达到最大大小时，是否清空或备份，'c'清空，'b'备份
}

type MylogsConfig struct {
	IsFullPrint     bool                 //多行打印，true为逐行打印，false在页面上只有一行不停刷新
	IsTimePrint     bool                 //是否打印时间
	IsColorPrint    bool                 //是否打印颜色
	IsSaveFile      bool                 //是否保存到文件
	IsPrintCodeLine int                  //是否打印代码行数，1打印“包名/文件名:行号”，2打印“绝对路径:行号”，其余任意值均不打印
	WaitTime        int                  //打印间隔时间，单位秒，0为不间隔
	ConfigSaveFile  MylogsConfigSaveFile //保存文件配置
}

type Mylogs struct {
	Configs        MylogsConfig //配置
	Colors         MylogColors  //颜色配置
	LogFileHandler *os.File     //日志文件句柄
}

func getCurrDir() string {
	currDir, err := os.Executable()
	if err != nil {
		return err.Error()
	}
	currDir = filepath.Dir(currDir)
	return currDir
}

func NewMylogs() *Mylogs {
	if Logger != nil {
		return Logger
	}

	mcsf := MylogsConfigSaveFile{
		FilePath:    filepath.Join(getCurrDir(), "logs", "current.log"),
		SavePath:    filepath.Join(getCurrDir(), "logs"),
		MaxSize:     5242880,
		ClearOrBack: 'c',
	}
	mc := MylogsConfig{
		IsFullPrint:     true,
		IsTimePrint:     true,
		IsColorPrint:    true,
		IsSaveFile:      true,
		IsPrintCodeLine: 2,
		WaitTime:        0,
		ConfigSaveFile:  mcsf,
	}

	Logger = &Mylogs{
		Configs: mc,
		Colors:  MlColors,
	}

	Logger.createLogsDir()
	Logger.getSaveFileHandler()

	return Logger
}

func (m *Mylogs) createLogsDir() {
	if _, err := os.Stat(m.Configs.ConfigSaveFile.SavePath); os.IsNotExist(err) {
		err = os.MkdirAll(m.Configs.ConfigSaveFile.SavePath, 0755)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func (m *Mylogs) clearLog() {
	err := m.LogFileHandler.Truncate(0)
	if err != nil {
		log.Fatal(err)
	}
}

func (m *Mylogs) backLog() {
	var (
		fileHandlerBak *os.File
		pathLogBak     string
		err            error
	)

	pathLogBak = filepath.Join(m.Configs.ConfigSaveFile.SavePath, fmt.Sprintf("%s.log", time.Now().Format("2006-01-02_15:04:05")))
	fileHandlerBak, err = os.OpenFile(pathLogBak, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer fileHandlerBak.Close()

	buf, err := io.ReadAll(m.LogFileHandler)
	if err != nil {
		log.Fatal(err)
	}

	_, err = fileHandlerBak.Write(buf)
	if err != nil {
		log.Fatal(err)
	}

	m.clearLog()
}
func (m *Mylogs) getSaveFileHandler() {
	fileInfo, err := os.Stat(m.Configs.ConfigSaveFile.FilePath)
	if os.IsNotExist(err) {
		m.LogFileHandler, err = os.OpenFile(m.Configs.ConfigSaveFile.FilePath, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
		if err != nil {
			log.Fatal(err)
			return
		}
	} else {
		if fileInfo.Size() >= m.Configs.ConfigSaveFile.MaxSize {
			m.LogFileHandler.Close()
			m.LogFileHandler, err = os.OpenFile(m.Configs.ConfigSaveFile.FilePath, os.O_RDWR|os.O_TRUNC, 0644)
			if m.Configs.ConfigSaveFile.ClearOrBack == 'c' {
				m.clearLog()
			} else if m.Configs.ConfigSaveFile.ClearOrBack == 'b' {
				m.backLog()
			}

		} else {
			m.LogFileHandler, err = os.OpenFile(m.Configs.ConfigSaveFile.FilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
			if err != nil {
				log.Fatal(err)
			}
		}

	}

}

func (m *Mylogs) beforePrint(status, statusColor, data interface{}) {

	if !m.Configs.IsColorPrint {
		statusColor = ""
	}

	currTime := ""
	if m.Configs.IsTimePrint {
		currTime = time.Now().Format("2006-01-02_15:04:05")
	}

	startChar := ""
	endChar := ""
	if m.Configs.IsFullPrint {
		startChar = ""
		endChar = "\n"
	} else {
		startChar = "\r"
		endChar = ""
	}

	codeLine := ""
	switch m.Configs.IsPrintCodeLine {
	case 2:
		_, file, line, ok := runtime.Caller(1)
		if !ok {
			log.Println("Mylogs>beforePrint() 无法获取代码行数")
			return
		}
		codeLine = fmt.Sprintf(" | %s:%d |", file, line)
	case 1:
		_, fileDir, line, ok := runtime.Caller(1)
		if !ok {
			log.Println("Mylogs>beforePrint() 无法获取代码行数")
			return
		}
		fileDir, fileName := filepath.Split(fileDir)
		if strings.Contains(fileDir, "/") {
			tmpList := strings.Split(fileDir, "/")
			fileDir = tmpList[len(tmpList)-2]
		} else if strings.Contains(fileDir, "\\") {
			tmpList := strings.Split(fileDir, "\\")
			fileDir = tmpList[len(tmpList)-2]
		}
		codeLine = fmt.Sprintf(" | %s/%s:%d |", fileDir, fileName, line)
	default:
		codeLine = ""
	}

	time.Sleep(time.Second * time.Duration(m.Configs.WaitTime))
	m.printLog(codeLine, status, statusColor, currTime, startChar, endChar, data)
}

func (m *Mylogs) Run(data interface{}) {
	m.beforePrint("RUN", m.Colors.Blue, data)
}

func (m *Mylogs) Success(data interface{}) {
	m.beforePrint("+", m.Colors.Green, data)
}

func (m *Mylogs) Faild(data interface{}) {
	m.beforePrint("-", m.Colors.Yellow, data)
}

func (m *Mylogs) Warrning(data interface{}) {
	m.beforePrint("!", m.Colors.Yellow, data)
}

func (m *Mylogs) Error(data interface{}) {
	m.beforePrint("ERR", m.Colors.Red, data)
}

func (m *Mylogs) Info(data interface{}) {
	m.beforePrint("INFO", m.Colors.Blue, data)
}

func (m *Mylogs) printLog(CodeLine, Status, StatusColor, TimeStr, StartChar, EndChar, data interface{}) {
	if m.Configs.IsSaveFile {
		_, err := m.LogFileHandler.WriteString(fmt.Sprintf("%s%s [%s] %v\n", TimeStr, CodeLine, Status, data))
		if err != nil {
			log.Println("Mylogs>printLog() 写入日志文件失败 ", err.Error())
			return
		} else {
			err = m.LogFileHandler.Sync()
			if err != nil {
				log.Println("Mylogs>printLog() 写入日志文件失败 ", err.Error())
				return
			}
		}
	}

	fmt.Printf("%s%s%s %s[%s]%s %s%s", StartChar, TimeStr, CodeLine, StatusColor, Status, MlColors.Reset, data, EndChar)
}
