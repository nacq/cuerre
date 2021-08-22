package lib

import (
	"os"
)

type Configuration struct {
	APP_URL string
	DB_URL string
	MODE string
	PORT string
}

var prodConfig = &Configuration{
	APP_URL: os.Getenv("APP_URL"),
	DB_URL: os.Getenv("DB_URL"),
	MODE: os.Getenv("MODE"),
	PORT: os.Getenv("PORT"),
}

var defaultConfig = &Configuration{
	APP_URL: "http://localhost",
	DB_URL: os.Getenv("DB_URL"),
	MODE: "development",
	PORT: "3030",
}

func GetConfig() *Configuration {
	if os.Getenv("MODE") == "production" {
		return prodConfig
	}

	return defaultConfig
}
