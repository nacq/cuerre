package lib

import (
	"os"
)

type Configuration struct {
	APP_URL string
	DB_NAME string
	DB_PASS string
	DB_USER string
	MODE string
	PORT string
}

var prodConfig = &Configuration{
	APP_URL: os.Getenv("APP_URL"),
	DB_NAME: os.Getenv("DB_NAME"),
	DB_PASS: os.Getenv("DB_PASS"),
	DB_USER: os.Getenv("DB_USER"),
	MODE: os.Getenv("MODE"),
	PORT: os.Getenv("PORT"),
}

var defaultConfig = &Configuration{
	APP_URL: "http://localhost",
	DB_NAME: os.Getenv("DB_NAME"),
	DB_PASS: os.Getenv("DB_PASS"),
	DB_USER: os.Getenv("DB_USER"),
	MODE: os.Getenv("MODE"),
	PORT: os.Getenv("PORT"),
}

func GetConfig() *Configuration {
	if os.Getenv("MODE") == "production" {
		return prodConfig
	}

	return defaultConfig
}
