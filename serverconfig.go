package webfw

import (
	"path"
	"runtime"
	"time"
)

var config *ServerConfig

// The configuration to set up a webfw-server.
type ServerConfig struct {
	RootDir         string
	MaxResponseTime time.Duration
	TimeoutMessage  string
	//	T                  translate.Translater
	TimeConfig   *TimeConfig
	RedisAddress string
	HttpAddress  string
	SharedDir    string
	Version      string
	SubDir       string
	WebUrl       string
	GitLabPath   string
	SmtpHost     string
	SmtpPort     string
	Environment  string
}

// NewServerConfig returns a new ServerConfig with some values.
func NewServerConfig() *ServerConfig {
	c := &ServerConfig{
		MaxResponseTime: time.Second * 20,
		TimeoutMessage:  "Your request timed out. Please try again. If this error reoccures, please contact us.",
		TimeConfig:      GetDefaultTimeConfig(),
		RedisAddress:    ":6379",
		HttpAddress:     ":4000",
		SubDir:          "/alpha",
		SmtpHost:        "mail.opendriverslog.de",
		SmtpPort:        "587",
	}

	_, filename, _, _ := runtime.Caller(1)
	c.RootDir = path.Dir(filename)

	return c
}

// SetConfig sets the ServerConfig.
func SetConfig(c *ServerConfig) {
	config = c
	V = NewViewEngine()
}

// Config gets the ServerConfig.
func Config() *ServerConfig {
	if config == nil {
		config = NewServerConfig()
	}
	return config
}

// TimeConfig determines how to print user-friendly date/timestamps.
type TimeConfig struct {

	// Long time format, contains month and year
	LongTimeFormatString string
	// Short time format, contains only time
	ShortTimeFormatString string

	// Time format for file names
	FileTimeFormatString string
	TimeLocation         *time.Location
}

// GetDefaultTimeConfig returns the default TimeConfig.
func GetDefaultTimeConfig() *TimeConfig {
	return &TimeConfig{
		LongTimeFormatString:  time.Stamp,
		ShortTimeFormatString: time.Kitchen,
		FileTimeFormatString:  "2006_01_02_15_04_05",
		TimeLocation:          time.UTC,
	}
}
