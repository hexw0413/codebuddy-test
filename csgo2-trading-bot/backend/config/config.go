package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Steam    SteamConfig    `mapstructure:"steam"`
	Trading  TradingConfig  `mapstructure:"trading"`
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type SteamConfig struct {
	APIKey        string `mapstructure:"api_key"`
	LoginURL      string `mapstructure:"login_url"`
	CallbackURL   string `mapstructure:"callback_url"`
	SharedSecret  string `mapstructure:"shared_secret"`
	IdentitySecret string `mapstructure:"identity_secret"`
}

type TradingConfig struct {
	BuffAPI struct {
		Enabled   bool   `mapstructure:"enabled"`
		BaseURL   string `mapstructure:"base_url"`
		AppID     string `mapstructure:"app_id"`
		AppSecret string `mapstructure:"app_secret"`
		Cookie    string `mapstructure:"cookie"`
	} `mapstructure:"buff"`
	
	YouPin struct {
		Enabled   bool   `mapstructure:"enabled"`
		BaseURL   string `mapstructure:"base_url"`
		APIKey    string `mapstructure:"api_key"`
		APISecret string `mapstructure:"api_secret"`
	} `mapstructure:"youpin"`
	
	AutoTrade struct {
		Enabled          bool    `mapstructure:"enabled"`
		MaxOrdersPerDay  int     `mapstructure:"max_orders_per_day"`
		MinProfitPercent float64 `mapstructure:"min_profit_percent"`
		MaxInvestment    float64 `mapstructure:"max_investment"`
	} `mapstructure:"auto_trade"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("../config")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	// 设置默认值
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", 5432)
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("redis.host", "localhost")
	viper.SetDefault("redis.port", 6379)
	viper.SetDefault("redis.db", 0)

	// 自动绑定环境变量
	viper.AutomaticEnv()

	var config Config
	
	if err := viper.ReadInConfig(); err != nil {
		// 如果配置文件不存在，使用默认值
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	if err := viper.Unmarshal(&config); err != nil {
		return nil, err
	}

	return &config, nil
}