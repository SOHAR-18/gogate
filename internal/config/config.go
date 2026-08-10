package config

import (
	"log"

	"github.com/spf13/viper"
)

type Config struct {
	GatewayPort    string
	AdminPort      string
	RedisHost      string
	RedisPort      string
	RedisPassword  string
	EtcdEndpoints  string
	JWTSecret      string
	AdminAPIKey    string
	LogLevel       string
	Env            string
	JaegerEndpoint string
}

func Load() *Config {
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		log.Println("No .env file found, reading from environment")
	}

	return &Config{
		GatewayPort:    viper.GetString("GATEWAY_PORT"),
		AdminPort:      viper.GetString("ADMIN_PORT"),
		RedisHost:      viper.GetString("REDIS_HOST"),
		RedisPort:      viper.GetString("REDIS_PORT"),
		RedisPassword:  viper.GetString("REDIS_PASSWORD"),
		EtcdEndpoints:  viper.GetString("ETCD_ENDPOINTS"),
		JWTSecret:      viper.GetString("JWT_SECRET"),
		AdminAPIKey:    viper.GetString("ADMIN_API_KEY"),
		LogLevel:       viper.GetString("LOG_LEVEL"),
		Env:            viper.GetString("ENV"),
		JaegerEndpoint: viper.GetString("JAEGER_ENDPOINT"),
	}
}
